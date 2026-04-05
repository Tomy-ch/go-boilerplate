# Versions Generator — Maintenance Guide

This document is a guide for users who have added generator tools to the **Tools containers (go_tool_runner / node_tool_runner)** to clearly understand what needs to be done after adding them.

Generator tools are inspected in CI by *GeneratedArtifactsCheck*, and it is designed to automatically determine whether **differences in generated artifacts are due to tool version changes or simply missing regeneration**.

Therefore, when adding a new tool, the following actions are mandatory.

## 1. Add version retrieval logic to `scripts/gen_tools_version.sh`

When you add a tool, first **append the tool version output processing to this script**.

Example (when adding a tool on the Go side):

```sh
go_section() {
  echo "## Go-based tools (go_tool_runner)"
  echo "- golang-migrate: $(normalize \"$(migrate -version 2>&1 || echo 'unknown')\")"
  echo "- mockgen: $(normalize \"$(mockgen -version 2>&1 || echo 'unknown')\")"
  echo "- oapi-codegen: $(normalize \"$(oapi-codegen --version 2>&1 || echo 'unknown')\")"
  echo "- sqlc: $(normalize \"$(sqlc version 2>&1 || echo 'unknown')\")"

  # ★ Add here
  echo "- your-tool: $(normalize \"$(<tool command> <version output subcommand> 2>&1 || echo 'unknown')\")"
}
```

For Node tools, append to `node_section()`.

## 2. Run `make gen-tools-meta` to update version information

Run the following locally:

```bash
make gen-tools-meta
```

This updates the version information written in `docs/meta/generator-versions.txt` to the latest.

## 3. Commit generated artifacts

After adding a new tool, you **must commit the following**:

- `docs/meta/generator-versions.txt`
- Generated code (result of `make gen`)

CI determines differences based on this file.

## 4. Confirm that generated artifact checks pass in CI

When creating a PR, the GitHub Actions generated artifact check workflow (`/.github/workflows/gen-artifacts-check.yaml`) will:

- Check differences in generated artifacts
- Comment on the PR if the differences are caused by version changes
- Fail CI if mismatches remain

When adding a new tool, always verify that CI succeeds on the PR.

## Why is this necessary?

This project contains many generated artifacts.

- Go code from oapi-codegen
- Query/Model from sqlc
- Mocks from mockgen
- Bundles from swagger-cli
- HTML from redocly
- etc.

All of these are **strongly dependent on the version of generator tools**.

Therefore:

- Someone generates with an old local tool → differences occur
- CI container uses a newer tool → differences detected
- Unintended fluctuations in generated artifacts → PR becomes noisy

To address this, by **keeping generator tool versions as a manifest and synchronizing with CI**, it becomes possible to automatically determine whether differences in generated artifacts are caused by tool updates.

## Summary: Required actions when adding a tool

|Task|Required|Description|
|------|------|------|
|Add to `scripts/gen_tools_version.sh`|Required|Output version information|
|Run `make gen-tools-meta`|Required|Update manifest|
|Commit artifacts and manifest|Required|Sync with CI|
|Run `make gen`|Required|If the new tool affects generated artifacts|
|Add tool to docker-compose|As needed|If a new service is required|

## 🔚 Closing

As the number of generator tools increases, fluctuations in generated artifacts due to environment differences also increase.

This manifest-based mechanism minimizes such fluctuations and ensures:

**clean PRs, stable CI, and reproducible development environments**.

When adding a tool, always refer to this README and complete the required steps 🎉
