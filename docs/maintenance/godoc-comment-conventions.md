# Go Doc-Comment Conventions (Condensed)

Condensed local mirror of <https://go.dev/doc/comment> ("Go Doc Comments"), kept so tooling
(e.g. the `comment-reviewer` agent) reads this instead of fetching the large upstream page.
Update this file if the upstream conventions change.

## Purpose and Scope

- A doc comment is the comment immediately preceding a top-level `package`, `const`, `func`,
  `type`, or `var` declaration (no blank line between).
- Doc comments document the **exported API for its consumer** — they are what `go doc`,
  pkg.go.dev/pkgsite, and gopls render. They are not implementation notes.
- **Every exported (capitalized) name, and every package, should have a doc comment.**

## Universal Format Rules

- Complete sentences, present tense.
- The **first sentence is a one-sentence summary that begins with the name being declared**
  (e.g. `// Quote returns a double-quoted Go string literal representing s.`). This makes
  the symbol searchable and the summary usable in lists.
- Refer to parameters and results by name directly, with no quoting or special syntax.
- Focus on what matters to callers; document special cases and edge cases prominently.

First-sentence patterns by declaration kind:

| Kind | First sentence starts with |
| --- | --- |
| Package | `Package <name> <verb> ...` |
| Command (`package main`) | `<Progname> <verb> ...` (program name, capitalized) |
| Type | `A <Name> ...` / `The <Name> ...` (or `<Name> is ...`) |
| Func / method | `<Name> <verb> ...`; boolean-returning: `<Name> reports whether ...` (never "or not") |
| Ungrouped const / var | `<Name> is/holds/... ...` |

## Per-Declaration Conventions

### Packages

- Every package has a package comment introducing it; first sentence starts `Package <name>`.
- In multi-file packages, put the package comment in only one file (conventionally `doc.go`
  for long ones); multiple comments are concatenated.
- Large packages: give an overview of the most important API parts, with doc links.

### Commands

- A command's package comment describes the **program's behavior**, not Go symbols; first
  sentence starts with the capitalized program name (e.g. `Gofmt formats Go programs.`).
- Conventionally includes a `Usage:` section and flag documentation as indented
  (preformatted) blocks.

### Types

- Explain what each instance of the type represents or provides.
- **Concurrency:** default assumption is safe for use by a single goroutine only; explicitly
  state any stronger guarantee.
- **Zero value:** document its meaning when non-obvious (e.g. "The zero value for Buffer is
  an empty buffer ready to use.").
- Exported struct fields: explain each via the type's doc comment or per-field
  (end-of-line) comments.

### Funcs and Methods

- Explain what the function returns, or what it does and its side effects.
- Multiple/named results may be named in the signature purely to make the doc readable.
- **Do not explain internal implementation or algorithms.** Asymptotic time/space bounds may
  be documented when critical to callers (e.g. `sort.Sort`).
- Top-level functions are assumed callable from multiple goroutines; methods are assumed
  single-goroutine by default — document only deviations.
- Constructors: top-level funcs returning `T` / `*T` (optionally with `error`) are grouped
  with the type by the doc tools; no special comment form required.
- Use consistent receiver names across a type's methods.

### Consts and Vars

- A **group** (`const (...)` / `var (...)`) may be introduced by a single doc comment on the
  group, with short end-of-line comments per member — or need no comment at all when the
  associated type's doc comment covers it (typed constant sets).
- An **ungrouped** const/var gets a full doc comment starting with its name.

## Rendered Syntax (what godoc/pkgsite understands)

- **Paragraphs:** spans of unindented non-blank lines; line breaks are preserved (semantic
  linefeeds allowed); gofmt never rewraps.
- **Headings:** a line `# Heading` (space after `#`, unindented, single line, blank lines
  around it). Anything else with `#` is plain text.
- **Lists:** indented lines starting with a bullet (`-`, `*`, `+`, `•`) or a number
  (`1.` / `1)`), followed by space/tab. List items contain paragraphs only — **no nested
  lists or code blocks**. Numbers are never renumbered.
- **Code blocks:** indented span not starting with a list marker → rendered preformatted.
  Used for example code, usage text, grammars, shell commands.
- **Links:** `[Text]` in prose plus a target line `[Text]: URL` (targets conventionally at
  the end of the comment). Bare URLs auto-link. `[Text]` without a target stays literal.
- **Doc links:** `[Name]`, `[Name1.Name2]` (current package), `[pkg]`, `[pkg.Name]`,
  `[pkg.Name1.Name2]`, optional leading star for pointers (`[*bytes.Buffer]`). Must be
  bounded by punctuation/space/line boundaries.
- **`Deprecated:`** a paragraph beginning with `Deprecated:` marks the symbol deprecated;
  tools warn on use and pkgsite hides it by default. Follow with the reason and the
  recommended replacement. Need not be the last paragraph.
- **Notes:** `MARKER(uid): body` with a 2+ uppercase-letter marker (`TODO(user):`,
  `BUG(user):`) is collected into its own rendered section.
- **Directives** (`//go:generate`, `//nolint`-style `//tool:directive`, `//line`, etc.) are
  **not** part of the rendered doc; gofmt moves them to the end of the comment. Never treat
  them as prose (and never delete them as "bad comments").
- gofmt canonicalizes doc comments (indentation, blank lines around code blocks, `#`
  headings, link-target placement) — formatting need not be policed by hand.

## Not Prescribed by godoc (treat as surplus in a doc comment)

The upstream conventions never ask for any of the following; their presence is noise, not
compliance:

- Step-by-step narration of the internal implementation ("how it works inside") — expressly
  discouraged for funcs.
- Call-site or registration notes ("called from X", "registered in the DI module").
- Change history, migration rationale, or development backstory (belongs in commits/PRs).
- Restating the identifier without adding information (e.g. `// GetUser gets the user.`
  adds nothing beyond the required name-prefixed summary — the summary must still say
  something a caller learns).
