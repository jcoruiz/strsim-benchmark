# strsim-benchmark

Compares strsim with go-edlib, strutil and smetrics on features, edge-case correctness and speed. `go run .` prints a Markdown report; the README keeps the results of one run, arranged into tables.

## Rules

- Every library gets the same deterministic inputs in each scenario. Correctness checks compare against the expected values written in `main.go`.
- The README results come from one run on one machine. When you refresh them, update the date, Go version and machine line, and take the library versions from `go.mod`.
- Keep the comparison fair: the README also says where other libraries win, such as smetrics on ASCII Levenshtein.

## Checks

```bash
gofmt -l .
go vet ./...
go run . > results.md   # about two minutes
```

# Git conventions

All git text is written in **English**: commit messages, PR titles and descriptions, branch names.

## Commits: Conventional Commits

Format: `<type>(<scope>): <subject>`

- **type** (required): `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `style`, `build`, `ci`, `chore`, `revert`.
- **scope** (optional): the area touched, in lowercase, such as `levenshtein`, `jaro`, `hamming`, `soundex`, `batch`, `ascii`, `correctness`, `deps`, `readme`, `ci`.
- **subject**: imperative mood, lowercase start, no trailing period, ≤ 72 chars for the whole header.
- **body** (optional): blank line after the header, then 1–3 short lines or bullets explaining *what/why*. No multi-paragraph prose.
- **Breaking changes**: `feat(scope)!: ...` plus a `BREAKING CHANGE: ...` footer.
- One logical change per commit; don't mix refactors with features.
- No `Co-Authored-By` or AI attribution lines.
- Enforced locally by `scripts/hooks/commit-msg` (enable once per clone with `git config core.hooksPath scripts/hooks`) and in CI by the `commits` job, which runs the same hook on every pushed commit and on pull request titles. (A GitHub ruleset can't do it: this account doesn't support commit message rules.)

Examples:

```
docs(readme): update the results from a fresh run
build(deps): update strsim to v0.2.1
feat(soundex): compare the codes of every library on a name list
fix(correctness): expect 1.0 for two empty strings
```

## Pull requests

- Title follows the same format: it becomes the squash commit, and the `commits` job checks it.
- Description: short summary of the change and how it was verified. No AI attribution.

## Branches

`<type>/<short-kebab-description>`, such as `feat/metaphone-scenario`, `fix/unicode-pair`.
