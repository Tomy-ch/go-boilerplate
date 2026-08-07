# docs-viewer

English | [日本語](README.ja.md)

The viewer for the documentation portal. It is a standalone static site: it reads the generated
`docs/portal/docs.json`, renders the cards and the document body, and holds no source of truth of
its own.

Its build output is committed under `docs/portal/`, which GitHub Pages serves as the site root
(ADR 0098). The viewer sources live outside `docs/` so that a package manifest, a lockfile, and
`node_modules` never end up inside the published tree.

## Why a separate package

**This is the only JavaScript application in the repository, and the only place that needs a
browser toolchain.** The Node dependencies the repository already had (`docker/tools`) exist to run
generators and linters; they are resolved with npm and installed into the tool-runner image. Mixing
a React application's dependency graph into that set would put a browser framework on the supply
surface of every code-generation run.

Keeping the viewer in its own package with its own package manager (pnpm) means its dependencies are
reachable from nothing else. The boundary is the package, not a convention.

## Relationship to the design system

The UI is built from a design system ported from `nextjs-boilerplate`
(`src/components/design-system`). The parts actually used by the viewer are vendored under
`src/components/`, together with the design tokens in `src/tokens/tokens.css`.

**The vendored subset is maintained here.** There is no generator behind the tokens in this
repository and no Storybook, so the upstream references to both were removed when the subset was
ported. Component behaviour, accessibility contracts, and the tests that lock them came across
unchanged.

## Structure

| Directory | Role |
| --- | --- |
| `src/docs-json/` | Schema and reader for the generated `docs.json`. A shape mismatch is a delivery failure, so it raises rather than recovers |
| `src/lang-filter/` | Filtering by display language. A section with no JA content falls back to EN as a whole, so languages never mix inside one section |
| `src/search/` | Search corpus assembly. Folds the owning section and group titles into each entry |
| `src/hash-route/` | Reading and writing the location hash `#/<group>/<section>` |
| `src/markdown/`, `src/sanitize/` | Markdown to a sanitized hast tree. Only values that went through sanitize carry the `SanitizedDocument` type |
| `src/code-fence/`, `src/code-block/` | Reading a fenced code block out of the tree, and rendering it with syntax highlighting |
| `src/mermaid-diagram/` | Rendering ` ```mermaid ` fences as diagrams |
| `src/components/` | The vendored design system subset and the design tokens |

## Rendering documents

Documents are rendered from the hast tree into React elements directly, never through an HTML
string. Two kinds of fenced code block are routed away from the default `pre`:

- ` ```mermaid ` becomes a diagram. mermaid runs with `securityLevel: "strict"`, and picks its
  palette from the same two signals as the design tokens (the OS setting and `data-theme`)
- every other fence is highlighted by highlight.js, which escapes its input before wrapping it in
  spans

Both libraries are loaded on demand, so neither is part of the payload that renders the card list.
A fence whose language highlight.js does not know, and a diagram mermaid cannot parse, are shown as
plain text rather than dropped.

## Commands

Run these through `make` so they execute in the tool-runner container, which pins Node and pnpm to
the versions declared in `mise.toml`:

| Command | What it does |
| --- | --- |
| `make gen-portal-build` | Builds the viewer into `docs/portal/` |
| `make portal-test` | Runs the test suite |

Working inside this package directly (`pnpm dev`, `pnpm test`, `pnpm typecheck`) is fine for a fast
loop; `pnpm dev` serves the viewer with hot reload, though it needs a `docs.json` next to it to have
anything to show.

## Dependency policy

- **Prefer parts that stand alone.** This viewer was ported from another repository and will likely
  be ported again; every dependency pulled in is a cost that travels with it. Where a component has
  a `-native` and a `-client` counterpart, take `-native` while the requirement allows
- **Keep the pure logic dependency-free apart from zod** (`docs-json` / `lang-filter` / `search` /
  `hash-route` / `code-fence`), so it stays portable as is
- **Versions are pinned exactly**, and `pnpm-workspace.yaml` declares the supply-chain settings:
  a release must be at least 7 days old to be resolved, dependency lifecycle scripts are refused
  unless declared, and non-registry sources are blocked
