# python

Version declarations and lockfiles for the CLI tools this repository installs from PyPI. Nothing
here is application code, and the repository ships no Python source: these are build-time tools
that happen to be published as Python packages.

## Why these tools are not in `mise.toml`

Every other tool version lives in `mise.toml` ([ADR-0078 (mise-ssot-drift-gate)](../docs/adr/0078-mise-ssot-drift-gate.md)).
A PyPI tool is the exception, because pinning its version pins almost nothing: its dependencies are
resolved at install time, so the same pin installs a different tree on different days, and no
scanner can read a version pin as a lockfile.

Each tool therefore gets a pair of files:

|File|Role|
|---|---|
|`<tool>.in`|The declaration. One `==` pin, plus the reason it is that version|
|`<tool>.txt`|The resolution. Every package in the transitive tree, pinned with its sha256 hashes|

Installation is always `uv pip install --require-hashes -r <tool>.txt`, which refuses a requirement
that lacks a version or a hash — so verification is part of installing, not a separate step that can
be skipped.

## One pair per tool

The tools here are unrelated CLIs that share only an ecosystem, and each is installed into its own
environment. Keeping one pair per tool keeps their resolutions independent: a dependency conflict
between two of them is a conflict that simply does not arise, and the Docker image that needs only
`sqlfluff` does not carry the other tool's tree.

## Changing a version

Edit the `==` pin in `<tool>.in`, then regenerate:

```bash
make py-lock
```

The two files are checked against each other: `make tool-cooldown-audit` (and the same gate on every
pull request) fails when a declaration and its lockfile name different versions. Without that check,
raising a `.in` and forgetting to regenerate would leave the cooldown gate clearing a version that is
never installed.

New versions are also subject to the supply-chain cooldown — 7 days for PyPI, as for any package
registry. A pin held below the latest release says so in the `.in` file.

## Who installs from these

|Consumer|Lockfile|
|---|---|
|`python_tools` image (`docker/tools/Dockerfile`), used by `make sql-lint` / `make sql-fix`|`sqlfluff.txt`|
|SQL Lint workflow (`.github/workflows/sql-lint.yaml`)|`sqlfluff.txt`|
|`.claude/scripts/bootstrap-external-skills.sh`|`graphify.txt`|

The Python runtime the lockfiles are resolved against is the one `mise.toml` declares; `make py-lock`
reads it from there rather than using whichever interpreter happens to be running.
