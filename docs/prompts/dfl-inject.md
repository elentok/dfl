# dfl inject

Implement the `dfl inject <source-file> <target-file>` command:

example:

- ~/.codex/AGENTS.md:

  ```md
  this is the user-scope Codex AGENTS.md
  ```

- ~/.dotfiles/core/ai/AGENTS.md:

  ```md
  this is the dotfiles' AGENTS.md
  ```

Running `dfl inject path/to/source.md path/to/target.md` will result in target.md becoming:

```md
this is the user-scope Codex AGENTS.md

---injected-from:~/.dotfiles/core/ai/AGENTS.md
this is the dotfiles' AGENTS.md
---end-injection
```

Open question:

- How should the start/end markers look like? (so they don't interfere with codex)
