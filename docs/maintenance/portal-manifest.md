# Documentation Portal — Manifest Contract

English | [日本語](portal-manifest.ja.md)

The documentation portal at `docs/portal/` is driven by an explicit **contract**: the visible structure (which groups exist, what each section is called, in what order things appear, which links go in the sidebar Reference block) is defined in `docs/portal/manifest.yaml`, and the build scripts under `scripts/` only **construct** the portal data by reading that manifest plus scanning the documentation filesystem.

```txt
manifest.yaml   ← structure   (what goes where, how it is labeled)
scripts/portal/*.ts ← construction (how it gets assembled)
```

This document is the single reference for that contract. When you add, move, rename, or remove documentation, the change you actually need to make almost always lives in `manifest.yaml`.

## 1. Files involved

| File | Role |
| --- | --- |
| `docs/portal/manifest.yaml` | Single source of truth for portal structure |
| `docs-viewer/src/**` | Source of the React viewer (reads `docs.json`). Built with Vite |
| `docs/portal/index.html` + `dist/**` | **Generated.** Build output of `docs-viewer` (`make gen-portal-build`). Both are gitignored and rebuilt on deploy |
| `docs/portal/docs.json` | **Generated.** Do not edit. Output of `gen-docs-json.ts` |
| `docs/portal/guides/**` | **Generated.** Do not edit. Output of `gen-portal-docs.ts` (flat copies of source READMEs) |
| `scripts/portal/gen-portal-docs.ts` | Copies each manifest entry's `src` to `dst` under `docs/portal/guides/` |
| `scripts/portal/gen-docs-json.ts` | Reads manifest + scans `docs/` and writes `docs/portal/docs.json` |

## 2. Manifest schema

`docs/portal/manifest.yaml` has two parts:

1. A top-level `meta:` block that defines the **visible structure**.
2. Top-level **section entries** whose key is the section id and whose value is a list of `{src, dst}` copy pairs.

### 2.1. The `meta:` block

```yaml
meta:
  groups:           # ordered list of top-level pages in the sidebar
    - title: "<Page title>"
      sections: [<section_id>, <section_id>, ...]   # rendered in this order on the page

  subgroups:        # optional: split a section into role-based subgroupings
    <section_id>:
      - title: "<Subgroup title>"
        items: [<guide_id>, <guide_id>, ...]        # guide_id = guides/<id>.md without extension

  section_titles:   # optional: override the auto-titled section display name
    <section_id>: "<Display name>"

  reference_links:  # ordered list of section ids whose top item appears in the sidebar Reference block
    - <section_id>
    - <section_id>
```

| Key | Purpose | Required |
| --- | --- | --- |
| `meta.groups` | Defines which top-level pages exist and which sections each page contains. The order here is the order in the sidebar. | Yes |
| `meta.subgroups` | Subdivides a section into role-based subgroups (e.g., `Layer Top / HTTP Stack / Error Response`). Items not listed fall into an auto-appended `Other` subgroup. | Optional |
| `meta.section_titles` | Overrides the default section heading (which is auto-derived from the section id). Use this to fix capitalization (`DI`, `CLI`, `DB Schema`, `OpenAPI Reference`, ...) or to give a section a friendlier name. | Optional |
| `meta.reference_links` | Section ids whose **single representative item** appears as a persistent quick link in the sidebar. Used for generated HTML (godoc, db-schema, coverage, openapi). These sections are pulled out of the normal group/page flow. | Optional |

### 2.2. Section entries

Every key at the top level **other than `meta`** is a section id. The value is a list of copy pairs:

```yaml
<section_id>:
  # English
  - src: <source path in repo>
    dst: docs/portal/guides/<flat-name>.md
  - src: <another EN source>
    dst: docs/portal/guides/<another>.md
  # Japanese
  - src: <source path>.ja.md
    dst: docs/portal/guides/ja/<flat-name>.ja.md
  - ...
```

`src` is the canonical README in the repo. `dst` is where the build copies it to under `docs/portal/guides/`. The viewer fetches the `dst` path at runtime; the `src` path is shown as the small "where this came from" line on each card.

**Naming rule for `dst`**: the basename (without `.md` / `.ja.md`) is the **guide id** that `meta.subgroups` references. Keep guide ids unique within the portal.

## 3. Filesystem auto-discovery (TypeScript construction side)

For documentation that lives under `docs/` directly (rather than as `**/README.md` in a code package), `gen-docs-json.ts` discovers files by scanning the filesystem. The placement and titles for these discovered sections still come from `meta:` — only the **enumeration of files** is TypeScript-side.

| Filesystem location | Becomes section | Notes |
| --- | --- | --- |
| `docs/*.md` | section id `architecture` | Root-level architecture docs (rules / decisions / development-flow / ...) |
| `docs/*.ja.md` | items of section `architecture` (lang: ja) | Japanese counterparts |
| `docs/<dir>/*.md` | section id `<dir>` | Auto for any subdir; e.g. `docs/maintenance/*.md` → section `maintenance` |
| `docs/<dir>/*.ja.md` | items of section `<dir>` (lang: ja) | Japanese counterparts |
| `docs/<dir>/index.html` | section id `<dir>`, single HTML item (lang: all) | For generated reference sites (godoc, coverage, ...) |

To control where an auto-discovered section appears in the portal, reference its id from `meta.groups` (for placement) and optionally `meta.section_titles` (for display name) / `meta.subgroups` (for subdivision) / `meta.reference_links` (to pull it out as a quick link).

A section id that is referenced from `meta:` but has no matching manifest entry **and** no matching filesystem location will be skipped with a `⚠` warning. A section id discovered from the filesystem that is not referenced from `meta.groups` (and not in `meta.reference_links`) is collected into an `Uncategorized` group at the end as a fallback.

## 4. How to make common changes

### 4.1. Add a new README to the portal

Most code-package READMEs (`internal/<layer>/<sub>/README.md`, `pkg/<sub>/README.md`, ...) are surfaced this way.

1. Pick the section it belongs to (e.g., `controller`, `infrastructure`).
2. Add an EN + JA pair under that section in `manifest.yaml`:

   ```yaml
   controller:
     # English
     - src: internal/controller/<new-package>/README.md
       dst: docs/portal/guides/controller-<new-package>.md
     # Japanese
     - src: internal/controller/<new-package>/README.ja.md
       dst: docs/portal/guides/ja/controller-<new-package>.ja.md
   ```

3. If the section uses `meta.subgroups`, also list the new guide id (`controller-<new-package>`) in the appropriate subgroup — otherwise it lands in `Other`.
4. Run `make gen-portal-docs && make gen-docs-json`.

### 4.2. Add a new section

Two paths depending on where the source lives:

- **Source is a README in code** (`internal/<x>/README.md` etc.) → add the section as a new top-level key in `manifest.yaml`, then add the section id to `meta.groups`.
- **Source is a markdown file under `docs/<new-dir>/`** → just put the file there; auto-discovery picks it up. Add `<new-dir>` to `meta.groups` to control where it appears, and optionally to `meta.section_titles` to give it a nicer name.

### 4.3. Move a doc to a different group

Edit `meta.groups` only. The section id stays the same; just move it from one group's `sections:` list to another's.

### 4.4. Rename a section heading

Add or change `meta.section_titles.<section_id>`. The id itself does **not** change.

### 4.5. Subdivide a hot section by role

Add an entry under `meta.subgroups`:

```yaml
meta:
  subgroups:
    <section_id>:
      - title: "<Role A>"
        items: [<guide_id>, <guide_id>]
      - title: "<Role B>"
        items: [<guide_id>]
```

Any item not listed lands in an auto-appended `Other` subgroup, so partial coverage is safe.

### 4.6. Add a new reference link (generated HTML)

1. Make sure the source generator puts the HTML at `docs/<name>/index.html`.
2. Add `<name>` to `meta.reference_links` in the desired display order.
3. Optionally override the title via `meta.section_titles.<name>`.

The HTML is **not copied** into `docs/portal/guides/` — the link points directly at the original location and opens in a new tab.

## 5. Build commands

| Command | What it does |
| --- | --- |
| `make gen-portal-docs` | Copies every `src` → `dst` declared in manifest entries. Validates that all sources exist and that no `dst` escapes `docs/portal/guides/`. |
| `make gen-docs-json` | Reads `manifest.yaml` + scans `docs/`, writes `docs/portal/docs.json`. |
| `make gen-docs` | Runs both of the above plus the OpenAPI redocly build. |

The `*-ci` variants (`make gen-portal-docs-ci`, `make gen-docs-json-ci`) call the Node scripts directly without the Docker tool runner — used by CI.

## 6. Local preview

Serve the `docs/` directory with any static HTTP server and open the portal:

```sh
# pick any static server you have handy
python3 -m http.server 8082 -d docs
# then open
open http://localhost:8082/portal/
```

The portal is a single-page React app whose source lives in `docs-viewer/` and is bundled by Vite, so a static server only ever shows the last build. To iterate on the viewer itself, run `pnpm --dir docs-viewer dev` and let Vite serve it with hot reload; run `make gen-portal-build` when you want to see exactly what gets published. `docs/portal/dist/` is gitignored — `deploy-docs.yaml` rebuilds it before uploading to Pages, so a viewer change reaches the site without any build output being committed.

## 7. Anti-patterns

- **Do not hand-edit `docs/portal/docs.json` or `docs/portal/guides/**`.** They are regenerated wholesale; your edits will vanish.
- **Do not bypass the manifest** by adding sections directly inside `gen-docs-json.ts`. Anything structural belongs in `meta:`.
- **Do not introduce a third level of grouping** inside subgroups. If a subgroup is getting too dense, split the section instead.
- **Do not give two manifest items the same `dst`.** Guide ids must be unique.

## 8. Summary

| Concern | Lives in |
| --- | --- |
| Which pages exist (group titles + order) | `manifest.yaml` → `meta.groups` |
| Which sections appear on which page (in what order) | `manifest.yaml` → `meta.groups[].sections` |
| Role-based subdivision inside a section | `manifest.yaml` → `meta.subgroups` |
| Section heading override | `manifest.yaml` → `meta.section_titles` |
| Sidebar Reference quick links | `manifest.yaml` → `meta.reference_links` |
| Which source files are copied to `guides/` | `manifest.yaml` flat entries |
| File enumeration under `docs/<dir>/` and `docs/*.md` | `scripts/portal/gen-docs-json.ts` (FS scan) |
| Filename → card title (`autoTitle`) | `scripts/portal/gen-docs-json.ts` (deterministic) |
| EN / JA item ordering and slugification | `scripts/portal/gen-docs-json.ts` (deterministic) |

If a change touches anything in the upper half of the table, the change is **in the manifest**. If a change touches the lower half, the change is **in the script**. Splitting the responsibility this way keeps the manifest small enough to scan in one sitting while letting the build stay self-maintaining for the long tail of READMEs.
