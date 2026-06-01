# PRD: `dfl merge-json` command

## Problem Statement

Stable, managed Claude config lives in a tracked `settings.base.json`, but `~/.claude/settings.json`
is owned by Claude at runtime (it writes `model`, `effortLevel`, etc. there), so it can't be
symlinked. Today the dotfiles install script merges them with a raw `jq -s '.[0] * .[1]'` one-liner.
That shells out to `jq`, and jq's `*` operator **replaces** arrays — so merging the managed base
over the live file clobbers list-valued settings like `permissions.allow` instead of combining them.
There is no first-class, testable merge primitive in `dfl`.

## Solution

A new `dfl merge-json <input...> <output>` command, with the same interface conventions as
`dfl inject`, that deep-merges any number of JSON files into an output file. Objects merge
recursively, **arrays union** (so allow-lists combine instead of clobbering), and scalars take the
last file's value. It is in-place safe (output may be one of the inputs), skips writing when nothing
changed, and supports dry-run — letting the dotfiles install script drop its `jq` dependency for a
`dfl merge-json "$claude_settings" "$claude_base" "$claude_settings"` call.

## User Stories

1. As a dotfiles maintainer, I want to merge `settings.base.json` into `~/.claude/settings.json`
   with one `dfl` command, so that I no longer depend on `jq` being installed during setup.
2. As a dotfiles maintainer, I want my managed `permissions.allow` entries to be **added** to the
   live allow-list rather than replacing it, so that both Claude's runtime-added permissions and my
   tracked ones survive a re-install.
3. As a dotfiles maintainer, I want Claude's runtime-only keys (e.g. `model`, `effortLevel`) in the
   live settings preserved when I push managed config, so that re-running install doesn't reset my
   editor choices.
4. As a dotfiles maintainer, I want managed base keys to win on scalar conflicts, so that my tracked
   config is authoritative for the settings it declares.
5. As a dotfiles maintainer, I want to merge more than two files in one call, so that I can compose
   config from several fragments without chaining commands.
6. As a dotfiles maintainer, I want the output path to be allowed to equal one of the input paths,
   so that I can merge a file in place.
7. As a dotfiles maintainer, I want the command to skip writing when the merged result equals the
   current output, so that re-running install is a no-op and reports `already up to date`.
8. As a dotfiles maintainer, I want a `--dry-run` that reports what would happen and writes nothing,
   so that I can preview a merge safely (matching the rest of the runtime commands).
9. As a dotfiles maintainer, I want a clear step line (`Merging X files into <output>...` then
   `done`), so that the merge fits the existing install output format.
10. As a dotfiles maintainer, I want input paths resolved like `inject` sources (relative to the
    component root, with `~`/absolute honored) and the output resolved like an `inject` target, so
    that the command behaves consistently with the rest of `dfl`.
11. As a dotfiles maintainer, I want invalid or missing JSON inputs to fail with a clear error, so
    that I notice a broken fragment instead of silently producing wrong output.
12. As a dotfiles maintainer, I want the command to reject calls with fewer than two inputs plus an
    output, so that I don't accidentally call it like a `copy`.
13. As a dotfiles maintainer, I want nested objects merged at any depth, so that deeply structured
    settings combine correctly rather than the whole subtree being replaced.
14. As a dotfiles maintainer, I want array union to deduplicate by value (including object/number
    elements), so that re-running a merge doesn't accumulate duplicate entries.
15. As a dotfiles maintainer, I want array union to preserve first-seen order, so that diffs stay
    minimal and predictable.
16. As a developer, I want the merge logic as a pure, isolated package, so that I can unit-test all
    the merge semantics without touching the filesystem.
17. As a developer, I want deterministic output (sorted keys, 2-space indent, trailing newline), so
    that the result is stable across runs and easy to diff.

## Implementation Decisions

- **New deep module `internal/jsonmerge`** — pure, no I/O. Holds:
  - A merge function that reduces decoded JSON values (`map[string]any`, `[]any`, scalars) left to
    right. Per key conflict: both objects → recurse; both arrays → **deep-union** (concatenate,
    dedup by deep-equality, keep first-seen order, applied at any depth); otherwise (scalar,
    array↔object, type mismatch) → **last value wins**.
  - A canonical marshal that emits **sorted keys, 2-space indent, trailing newline**. (Diverges from
    jq's insertion-order; accepted — see ADR 0001.)
- **`Runner.MergeJSON(ctx, componentRoot, inputs []string, output string)`** in
  `internal/runtimecmd`, returning `(ResultStatus, message, error)` like `Inject`/`Copy`:
  - Resolve each input like an `inject` source (relative to `componentRoot` unless abs/`~`); resolve
    `output` like an `inject` target (`expandPath`).
  - **Read and parse all inputs fully before writing**, so `output` may overlap an input (in-place).
  - Reduce via `jsonmerge`, marshal canonically.
  - If rendered == current output content → `StatusSkipped`, `already up to date`.
  - Dry-run → `StatusSuccess`, `would merge X files into <output>`, write nothing.
  - Else write (creating parent dirs as the other runtime commands do).
- **CLI**: `newMergeJSONCommand` in `internal/cli/runtime.go`, `Use: "merge-json <input...> <output>"`,
  `Args: cobra.MinimumNArgs(3)`, last positional is output. Step start
  `Merging %d files into %s...`, then `logStepResult` + terse `ui.StepEnd` (`done`). Registered in
  `internal/cli/root.go` next to `newInjectCommand`. **No aliases, no alternate arg syntax**
  (per `.ai/cli.md`).
- **Precedence rule is "last input wins"**, so the dotfiles call orders live settings first and the
  managed base last: `dfl merge-json "$claude_settings" "$claude_base" "$claude_settings"` — base
  keys win on scalar conflicts (matching old jq `.[0] * .[1]`), runtime-only keys preserved, output
  in place. (Call-site swap lives in `~/.dotfiles`, outside this repo.)

## Testing Decisions

Good tests here exercise **external behavior** — given input JSON, assert the merged output —
not internal helper shapes. Three layers, all requested for coverage:

- **`jsonmerge` (core, primary):** table-driven unit tests on the pure functions. Cases: object
  deep-merge; array union with dedup; array union preserving first-seen order; nested arrays at
  depth; scalar last-wins; array↔object and type-mismatch last-wins; multi-file (≥3) reduce;
  dedup of object/number array elements; canonical sorted-key/indent/trailing-newline output.
- **`Runner.MergeJSON` (I/O):** in-place `output == input`; skip-if-unchanged returns `already up to
  date`; dry-run reports `would merge ...` and leaves the file untouched; missing input file errors;
  invalid JSON input errors. Prior art: the `Inject`/`Copy` tests in
  `internal/runtimecmd/runner_test.go` (temp dirs, status/message assertions).
- **CLI command:** arg validation rejects fewer than 3 args; happy path emits the
  `Merging X files into <output>...` step. Prior art: existing command tests in
  `internal/cli/*_test.go`.

## Out of Scope

- A replace-arrays mode or any flag to opt out of union semantics (deferred until a real need;
  ADR 0001 records union as the sole behavior).
- Preserving original object key order (we sort — ADR 0001).
- JSON5 / comments / JSONC support.
- The dotfiles `~/.dotfiles/extra/claude/install` edit itself — tracked as a follow-up in that repo,
  not implemented here.
- Merging non-JSON formats (YAML, TOML).

## Further Notes

- See `docs/adr/0001-merge-json-deep-union-semantics.md` for the union-vs-replace decision and the
  sorted-keys consequence.
- See `docs/plans/merge-json.md` for the task checklist.
- The two `%s`/`%d` in the label are count + output, not source + target — a deliberate deviation
  from `inject`'s `%s into %s` because there can be many sources.
