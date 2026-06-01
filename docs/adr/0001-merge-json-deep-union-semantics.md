# merge-json deep-merges objects but unions arrays

`dfl merge-json` replaces the `jq -s '.[0] * .[1]'` one-liner that merged `settings.base.json`
into `~/.claude/settings.json`. We deliberately diverge from jq's `*` semantics in one respect:
**arrays are unioned (concatenate, dedup by deep-equality, first-seen order, at any depth) instead
of replaced.** Objects still deep-merge and scalars/type-mismatches still take the last file's
value, matching jq.

The motivation is `permissions.allow`: with jq's replace-arrays behavior, merging the managed base
over the live settings would clobber one allow-list with the other. Union lets both contribute
entries, which is the whole point of the merge for this file.

## Consequences

- `merge-json` is **not** a drop-in for jq's `*`; anyone diffing against the old behavior should
  expect arrays to grow rather than be overwritten. There is no way to request replace-arrays.
- Object keys are emitted **sorted** (Go's default marshal order), unlike jq which preserves
  insertion order. This causes a one-time reordering of the output file on first run, then is
  stable. Accepted as harmless for a machine-managed file.
