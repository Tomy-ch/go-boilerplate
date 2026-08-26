// 言語を畳む撤去で、機械規則に載らないものを宣言する manifest（データ定義）。
// 規則は doc-language.ts、計画は plan.ts、実行は index.ts を参照。
//
// 対訳ペアそのものは列挙しない。ペアは `<name>.md` と `<name>.ja.md` の隣接という規則で決まり、
// 422 組を書き並べても増減のたびに宣言が腐るだけである。ここに書くのは規則で決まらないもの
// —— 対訳の存在が主題であるがゆえに撤去ごと必要になるツールと、機械では畳めない散文だけ。

import { type Mode } from "./doc-language";

/** `--lang` が受け付ける値。 */
export const MODES: readonly (Mode | "both")[] = ["en", "ja", "both"];

/** 撤去のコミット件名。commitlint の type-enum に合わせる。 */
export const COMMIT_SUBJECTS: Readonly<Record<Mode, string>> = {
  en: "Docs: 日本語の対訳を撤去し英語 1 本へ畳む",
  ja: "Docs: 英語正本を撤去し日本語 1 本へ畳む",
};

/**
 * 対訳ペアの存在そのものが主題で、畳んだ後は主語を失うもの。
 *
 * @remarks
 * `canonicalize-doc` は正本と対訳の組を作って同期するスキルです。組が無くなれば、この
 * スキルが何をするのかを言える人はいなくなります。散文を削って残す道もありますが、
 * 削り終えた後に残るのは「1 つの文書を 1 つの文書へ同期するスキル」で、それは機能ではありません。
 *
 * ポータルの言語フィルタも同じです。畳んだ後の `docs.json` は 1 言語しか持たないため、
 * 切り替え先の無い切り替えボタンだけが残ります。
 */
export const REMOVED_PATHS: readonly string[] = [
  // ポータルの日本語複製。manifest から作られる生成物だが、作られなくなるだけでは消えない。
  "docs/portal/guides/ja",
  ".claude/skills/canonicalize-doc",
  ".codex/skills/canonicalize-doc",
  "docs-viewer/src/lang-filter",
];

/**
 * 撤去し終えた後に自分自身を落とす対象。
 *
 * @remarks
 * 一度きりの操作なので、ツールと、それを呼ぶ make ターゲットの宣言と、それを叩く CI を
 * 一緒に落とします。作成先のリポジトリに残しても、叩けば「畳むものが無い」で失敗するだけです。
 * make ターゲット側の宣言とレシピは `doc-pair` マーカーで同時に落ちます。
 */
export const SELF_DESTRUCT_PATHS: readonly string[] = [
  ".github/workflows/doc-language-removal-check.yaml",
  "scripts/setup/remove-doc-language",
];

/** 行は残したまま、文字列だけを差し替える宣言。 */
export type DocReplacement = {
  file: string;
  from: string;
  to: string;
  /**
   * この宣言が効くモード。省略すると両方で効く。
   *
   * @remarks
   * 大半の宣言は両モードで同じですが、`ja` では改名によって「同じパスの中身が別の言語になる」
   * ため、そのパスへ完全一致で書き込む宣言は言語ごとに別物になります。`remove-licensed-scanners`
   * が `.github/workflows/README.md` の英文を宣言しているのがその一例で、`ja` を選ぶと同じ
   * パスに日本語が入り、宣言は当たらなくなります。
   */
  mode?: Mode;
};

/**
 * 表のセルのように、行ごと落とせない場所の差し替え。
 *
 * @remarks
 * マーカーは自分の行にコメントを置くため、表の中では使えません（コメント行がそこで表を終わらせる）。
 * 表全体を `replace` で包む道も、`tool-map` の表が既に HTML コメントを 1 つ持っているため塞がって
 * います —— HTML コメントの退避コメントに HTML コメントは入れられず、最初の `-->` が外側を閉じます。
 *
 * 完全一致にしているのは、本文が動いたときに空振りではなく停止させるためです。差し替え先が
 * 見つからなければ撤去は止まり、宣言を直すまで進みません。
 */
export const DOC_REPLACEMENTS: readonly DocReplacement[] = [
  {
    file: ".claude/skills/tool-map/SKILL.md",
    from: " (skip `SKILL.ja.md` and other `*.ja.md` translation files)",
    to: "",
  },
  {
    file: ".codex/skills/tool-map/SKILL.md",
    from: " (skip `SKILL.ja.md` and other `*.ja.md` translation files)",
    to: "",
  },
  {
    file: ".github/workflows/README.md",
    from: " and its `README.ja.md` translation",
    to: "",
  },
  {
    file: ".github/workflows/README.md",
    from: "| `make md-lint` checks the pair, not the rows |",
    to: "| `make md-lint` does not check these rows |",
  },
  {
    file: ".claude/skills/back-prop/SKILL.md",
    from: ", minus `*.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/back-prop/SKILL.md",
    from: ", excluding `*.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/commit/SKILL.md",
    from: ", `*.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/context-map/SKILL.md",
    from: " and its `.ja.md` pair",
    to: "",
  },
  {
    file: ".codex/skills/context-map/SKILL.md",
    from: " and `docs/design/context-map.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/glossary/SKILL.md",
    from: "、`.ja.md` ペアの作成",
    to: "",
  },
  {
    file: ".codex/skills/glossary/SKILL.md",
    from: " Do not create a `.ja.md` pair for the glossary: this spec tree uses one Japanese file with English headings.",
    to: "",
  },
  {
    file: ".claude/skills/go-upgrade/SKILL.md",
    from: " / `README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/go-upgrade/SKILL.md",
    from: " / `README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/go-upgrade/SKILL.md",
    from: " / `docker/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/go-upgrade/SKILL.md",
    from: " / `docker/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/go-upgrade/SKILL.md",
    from: " / `docker/server/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/go-upgrade/SKILL.md",
    from: " / `docker/server/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/go-upgrade/SKILL.md",
    from: " / `docker/tools/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/go-upgrade/SKILL.md",
    from: " / `docker/tools/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/new-env/SKILL.md",
    from: ", `env/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.md",
    from: ", `env/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/new-env/SKILL.md",
    from: " and `README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.md",
    from: " and `README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/new-env/SKILL.md",
    from: ", env/README.ja.md",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.md",
    from: ", env/README.ja.md",
    to: "",
  },
  {
    file: ".claude/skills/new-env/SKILL.md",
    from: " then `env/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.md",
    from: " then `env/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/new-env/SKILL.md",
    from: " and `env/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.md",
    from: " and `env/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/portal-manifest-sync/SKILL.md",
    from: " / `README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/portal-manifest-sync/SKILL.md",
    from: " / `README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/readme-review/SKILL.md",
    from: "\n- Cross-reference to translation (`README.ja.md`) — its existence and sync convention compliance",
    to: "",
  },
  {
    file: ".codex/skills/readme-review/SKILL.md",
    from: "\n- Cross-reference to translation (`README.ja.md`) — its existence and sync convention compliance",
    to: "",
  },
  {
    file: ".claude/skills/repo-ops/SKILL.md",
    from: "-g '!**/*.ja.md' ",
    to: "",
  },
  {
    file: ".codex/skills/repo-ops/SKILL.md",
    from: "-g '!**/*.ja.md' ",
    to: "",
  },
  {
    file: ".claude/skills/tools-upgrade/SKILL.md",
    from: ", `docker/**/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/tools-upgrade/SKILL.md",
    from: ", `docker/**/README.ja.md`",
    to: "",
  },
  {
    file: "docs/adr/0000-record-architecture-decisions.md",
    from: " (each ADR also needs its `.ja.md` translation)",
    to: "",
  },
  {
    file: "scripts/README.md",
    from: ", translation pairs (`SKILL.ja.md` exists, carries no frontmatter, opens with a sync note, and its heading-level sequence matches `SKILL.md`)",
    to: "",
  },
  {
    file: ".claude/skills/comment-sweep/SKILL.md",
    from: "   Whichever is chosen, the English canonical file and its `.ja.md` translation — plus the log table in\n   `docs/adr/README.md` and `docs/adr/README.ja.md` — are updated in the same change.",
    to: "   Whichever is chosen, the file and the log table in `docs/adr/README.md` are updated in the same\n   change.",
  },
  {
    file: ".claude/skills/new-env/SKILL.md",
    from: "; the skill translates for the other",
    to: "",
  },
  {
    file: ".claude/skills/new-env/SKILL.md",
    from: "\nResolution rules:\n\n- If only Japanese provided → skill writes Japanese to `env/README.ja.md` row, then translates to English for `env/README.md` row.\n- If only English provided → reverse direction.\n- If both provided → use as-is, no translation.\n- Translations are kept short and direct (single-line, technical register matching surrounding rows). If the description is non-trivial or domain-specific, surface the proposed translation in the Step 2 plan summary for user review before writing.\n",
    to: "",
  },
  {
    file: ".claude/skills/portal-manifest-sync/SKILL.md",
    from: "\n- Japanese: `docs/portal/guides/ja/<flat-hyphenated-name>.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/portal-manifest-sync/SKILL.md",
    from: "\n  - src: foo/bar/README.ja.md",
    to: "",
  },
  {
    file: ".claude/skills/repo-ops/SKILL.md",
    from: "Most of the Markdown in this tree is either a Japanese mirror you must not read or generated output\nthat lags the code, so a naive repo-wide search buries the one file that actually decides the answer.\nOf roughly 1,000 tracked `*.md`, **over 40% are `*.ja.md` translations** and **72 are generated\n`docs/portal/guides/**` copies of READMEs**; `docs/godoc/**` adds ~1,250 files and\n`docs/db-schema/**` ~390.",
    to: "Much of the Markdown in this tree is generated output that lags the code, so a naive repo-wide\nsearch buries the one file that actually decides the answer. Of the tracked `*.md`, **72 are\ngenerated `docs/portal/guides/**` copies of READMEs**; `docs/godoc/**` adds ~1,250 files and\n`docs/db-schema/**` ~390.",
  },
  {
    file: ".claude/skills/repo-ops/SKILL.md",
    from: "are *tracked*, so they need these explicit globs. Hitting a `*.ja.md` is still useful as a\n**locator** (it proves the topic is documented); read the English original beside it, per\n`AGENTS.md`'s rule never to read `*.ja.md`.",
    to: "are *tracked*, so they need these explicit globs.",
  },
  {
    file: ".claude/skills/tool-map/SKILL.md",
    from: "\n- `*.ja.md` translation files (they are not loaded as entries).",
    to: "",
  },
  {
    file: ".claude/skills/tool-map/SKILL.md",
    from: "\n- [ ] `*.ja.md` files excluded from the skills scan",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.md",
    from: "; the skill translates for the other",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.md",
    from: "\nResolution rules:\n\n- If only Japanese provided → skill writes Japanese to `env/README.ja.md` row, then translates to English for `env/README.md` row.\n- If only English provided → reverse direction.\n- If both provided → use as-is, no translation.\n- Translations are kept short and direct (single-line, technical register matching surrounding rows). If the description is non-trivial or domain-specific, surface the proposed translation in the Step 2 plan summary for user review before writing.\n",
    to: "",
  },
  {
    file: ".codex/skills/portal-manifest-sync/SKILL.md",
    from: "\n- Japanese: `docs/portal/guides/ja/<flat-hyphenated-name>.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/portal-manifest-sync/SKILL.md",
    from: "\n  - src: foo/bar/README.ja.md",
    to: "",
  },
  {
    file: ".codex/skills/repo-ops/SKILL.md",
    from: "Most of the Markdown in this tree is either a Japanese mirror you must not read or generated output\nthat lags the code, so a naive repo-wide search buries the one file that actually decides the answer.\nOf roughly 1,000 tracked `*.md`, **over 40% are `*.ja.md` translations** and **72 are generated\n`docs/portal/guides/**` copies of READMEs**; `docs/godoc/**` adds ~1,250 files and\n`docs/db-schema/**` ~390.",
    to: "Much of the Markdown in this tree is generated output that lags the code, so a naive repo-wide\nsearch buries the one file that actually decides the answer. Of the tracked `*.md`, **72 are\ngenerated `docs/portal/guides/**` copies of READMEs**; `docs/godoc/**` adds ~1,250 files and\n`docs/db-schema/**` ~390.",
  },
  {
    file: ".codex/skills/repo-ops/SKILL.md",
    from: "are *tracked*, so they need these explicit globs. Hitting a `*.ja.md` is still useful as a\n**locator** (it proves the topic is documented); read the English original beside it, per\n`AGENTS.md`'s rule never to read `*.ja.md`.",
    to: "are *tracked*, so they need these explicit globs.",
  },
  {
    file: ".codex/skills/tool-map/SKILL.md",
    from: "\n- `*.ja.md` translation files (they are not loaded as entries).",
    to: "",
  },
  {
    file: ".codex/skills/tool-map/SKILL.md",
    from: "\n- [ ] `*.ja.md` files excluded from the skills scan",
    to: "",
  },
  {
    file: "AGENTS.md",
    from: "**Documentation scope for agents** — the canonical sources are the English `README.md` and\n`docs/**/*.md`. **Never read `*.ja.md` files: they are human-facing Japanese translations of\nthose canonical sources — read the canonical English original instead.** Also ignore the\ndocumentation-portal UI assets:\n\n```txt\n**/*.ja.md\ndocs/portal/**\n```",
    to: "**Documentation scope for agents** — the canonical sources are `README.md` and `docs/**/*.md`.\nIgnore the documentation-portal UI assets:\n\n```txt\ndocs/portal/**\n```",
  },
  {
    file: "docs/get-started/setup-repository.md",
    from: "1. Rewrite the contents of README.md and README.ja.md according to your project; replace or remove\n   the repository-specific branch-rule exception in the maintainer-policy section.\n2. If your project keeps its documentation in a single language, you may collapse the pair — for\n   example by replacing README.md with the contents of README.ja.md.",
    to: "1. Rewrite the contents of README.md according to your project; replace or remove the\n   repository-specific branch-rule exception in the maintainer-policy section.",
  },
  {
    file: "docs/maintenance/docs-structure.md",
    from: "## 2. Japanese Documents\n\nA Japanese document sits **beside its English canonical**, named `<name>.ja.md`. The suffix is what\nseparates the languages; there is no separate directory, and the generator splits them by suffix.\n\nThese files are displayed in the **Architecture (Japanese)** section.\n\n## 3. Section Documents",
    to: "## 2. Section Documents",
  },
  {
    file: "docs/maintenance/docs-structure.md",
    from: "## 4. Japanese Section Documents\n\nA section's Japanese documents live in that same section directory, as `<name>.ja.md`. Nothing else\nis needed — the generator finds them by suffix and files them under:\n\n```txt\nProject (Japanese)\n```\n\n## 5. Reserved Directories",
    to: "## 3. Reserved Directories",
  },
  {
    file: "docs/maintenance/docs-structure.md",
    from: "\ndocs/security/auth.ja.md",
    to: "",
  },
  {
    file: "docs/maintenance/docs-structure.md",
    from: "\nSecurity (Japanese)",
    to: "",
  },
  {
    file: "docs/maintenance/docs-structure.md",
    from: "\n|docs/*.ja.md|Japanese|",
    to: "",
  },
  {
    file: "docs/maintenance/docs-structure.md",
    from: "\n|docs/<section>/*.ja.md|Japanese セクション|",
    to: "",
  },
  {
    file: "docs/maintenance/portal-manifest.md",
    from: "  # Japanese\n  - src: <source path>.ja.md\n    dst: docs/portal/guides/ja/<flat-name>.ja.md\n  - ...\n",
    to: "  - ...\n",
  },
  {
    file: "docs/maintenance/portal-manifest.md",
    from: " / `.ja.md`",
    to: "",
  },
  {
    file: "docs/maintenance/portal-manifest.md",
    from: "\n| `docs/*.ja.md` | items of section `architecture` (lang: ja) | Japanese counterparts |",
    to: "",
  },
  {
    file: "docs/maintenance/portal-manifest.md",
    from: "\n| `docs/<dir>/*.ja.md` | items of section `<dir>` (lang: ja) | Japanese counterparts |",
    to: "",
  },
  {
    file: "docs/maintenance/portal-manifest.md",
    from: "     # Japanese\n     - src: internal/controller/<new-package>/README.ja.md\n       dst: docs/portal/guides/ja/controller-<new-package>.ja.md\n",
    to: "",
  },
  {
    file: "scripts/README.md",
    from: " Codex-side `SKILL.ja.md` is optional, so it is\nchecked as a translation pair only when present.",
    to: "",
  },
  {
    file: ".claude/README.md",
    from: "- **`skill-lint` does not check it.** The repository's skill conventions — frontmatter, the\n  `SKILL.ja.md` pair, references that resolve — assume a skill this repository writes.",
    to: "- **`skill-lint` does not check it.** The repository's skill conventions — frontmatter and\n  references that resolve — assume a skill this repository writes.",
  },
  {
    file: ".claude/skills/comment-sweep/SKILL.ja.md",
    from: " と `docs/adr/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/comment-sweep/SKILL.ja.md",
    from: "英語の正本・隣の `.ja.md`・英日両方の ADR ログ表を揃えて更新する",
    to: "正本と ADR ログ表を揃えて更新する",
  },
  {
    file: ".claude/skills/commit/SKILL.ja.md",
    from: "、`*.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/go-upgrade/SKILL.ja.md",
    from: " / `docker/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/go-upgrade/SKILL.ja.md",
    from: " / `docker/server/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/go-upgrade/SKILL.ja.md",
    from: " / `docker/tools/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/manage-skill/SKILL.ja.md",
    from: " / `SKILL.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/new-env/SKILL.ja.md",
    from: ", `env/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/new-env/SKILL.ja.md",
    from: "\n- 日本語のみ供与 → `env/README.ja.md` に日本語そのまま、`env/README.md` に英訳を記入",
    to: "",
  },
  {
    file: ".claude/skills/new-env/SKILL.ja.md",
    from: ", env/README.ja.md",
    to: "",
  },
  {
    file: ".claude/skills/new-env/SKILL.ja.md",
    from: " → `env/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/portal-manifest-sync/SKILL.ja.md",
    from: " / `README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/repo-ops/SKILL.ja.md",
    from: "-g '!**/*.ja.md' ",
    to: "",
  },
  {
    file: ".claude/skills/tool-map/SKILL.ja.md",
    from: "\n- [ ] skills スキャンから `*.ja.md` を除外した",
    to: "",
  },
  {
    file: ".claude/skills/tools-upgrade/SKILL.ja.md",
    from: ", `docker/**/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/go-upgrade/SKILL.ja.md",
    from: " / `docker/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/go-upgrade/SKILL.ja.md",
    from: " / `docker/server/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/go-upgrade/SKILL.ja.md",
    from: " / `docker/tools/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/manage-skill/SKILL.ja.md",
    from: " / `SKILL.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.ja.md",
    from: ", `env/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.ja.md",
    from: "\n- 日本語のみ供与 → `env/README.ja.md` に日本語そのまま、`env/README.md` に英訳を記入",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.ja.md",
    from: ", env/README.ja.md",
    to: "",
  },
  {
    file: ".codex/skills/new-env/SKILL.ja.md",
    from: " → `env/README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/portal-manifest-sync/SKILL.ja.md",
    from: " / `README.ja.md`",
    to: "",
  },
  {
    file: ".codex/skills/repo-ops/SKILL.ja.md",
    from: "-g '!**/*.ja.md' ",
    to: "",
  },
  {
    file: ".codex/skills/tool-map/SKILL.ja.md",
    from: "\n- [ ] skills スキャンから `*.ja.md` を除外した",
    to: "",
  },
  {
    file: ".codex/skills/tools-upgrade/SKILL.ja.md",
    from: ", `docker/**/README.ja.md`",
    to: "",
  },
  {
    file: ".claude/skills/sync-ai/SKILL.ja.md",
    from: "、`sh -n` を通し、`SKILL.md` / `SKILL.ja.md` の見出し数が一致していることを確かめる",
    to: "、`sh -n` を通す",
  },
  {
    file: ".codex/skills/sync-ai/SKILL.ja.md",
    from: "さらに `sh -n` を実行し、`SKILL.md` / `SKILL.ja.md` の見出し数が一致することも確認する。",
    to: "さらに `sh -n` を実行する。",
  },
  {
    file: ".codex/skills/back-prop/SKILL.ja.md",
    from: " から `*.ja.md` を除いた",
    to: "",
  },
  {
    file: "docs/maintenance/docs-structure.ja.md",
    from: "\n|docs/*.ja.md|Japanese|",
    to: "",
  },
  {
    file: "docs/maintenance/docs-structure.ja.md",
    from: "\n|docs/<section>/*.ja.md|Japanese セクション|",
    to: "",
  },
  {
    file: "docs/maintenance/portal-manifest.ja.md",
    from: "\n| `docs/*.ja.md` | section `architecture` の item (lang: ja) | 日本語版 |",
    to: "",
  },
  {
    file: "docs/maintenance/portal-manifest.ja.md",
    from: "\n| `docs/<dir>/*.ja.md` | section `<dir>` の item (lang: ja) | 日本語版 |",
    to: "",
  },
  {
    file: ".claude/skills/back-prop/SKILL.ja.md",
    from: "、`docs/architecture.md`。`*.ja.md` は除く）",
    to: "、`docs/architecture.md`）",
  },
  {
    file: ".agents/closed-loop/skill-meta.yaml",
    from: "    opportunity_predicate: \"対訳ペアの片側だけが変更された（`X.md` と `X.ja.md` の一方のみ）\"\n",
    to: "",
  },
  {
    file: ".github/workflows/licensed-scanners-removal-check.yaml",
    from: "      - '.github/workflows/README.ja.md'\n",
    to: "",
  },
  {
    file: ".github/workflows/sync-versions-check.yaml",
    from: "      - 'docker/**/README.ja.md'\n",
    to: "",
  },
  {
    file: ".github/workflows/setup-scripts-check.yaml",
    from: " README.md README.ja.md ",
    to: " README.md ",
  },
  {
    file: ".gitleaksignore",
    from: "env/README.ja.md:generic-api-key:244\n",
    to: "",
  },
  {
    file: ".gitleaksignore",
    from: "env/README.ja.md:generic-api-key:219\n",
    to: "",
  },
  {
    file: ".graphifyignore",
    from: "*.ja.md\n**/*.ja.md\n",
    to: "",
  },
  {
    file: "scripts/graphify-pending/main.go",
    from: "//\t*.ja.md         ファイル名のパターン",
    to: "//\t*.gen.sql       ファイル名のパターン",
  },
  {
    file: "scripts/graphify-pending/main.go",
    from: "// matchesGlob は `*.ja.md` のような、",
    to: "// matchesGlob は `*.gen.sql` のような、",
  },
  {
    file: "scripts/graphify-pending/main_test.go",
    from: "\t\t\tfiles := manifest(`{\"docs/rules.ja.md\":{\"semantic_hash\":\"stale\"},\"docs/rules.md\":{\"semantic_hash\":\"stale\"}}`)\n\t\t\tfiles[\"docs/rules.ja.md\"] = body\n\t\t\tfiles[\"docs/rules.md\"] = body\n\t\t\tfiles[\"ignore\"] = \"*.ja.md\\n**/*.ja.md\\n\"",
    to: "\t\t\tfiles := manifest(`{\"docs/a.gen.sql\":{\"semantic_hash\":\"stale\"},\"docs/rules.md\":{\"semantic_hash\":\"stale\"}}`)\n\t\t\tfiles[\"docs/a.gen.sql\"] = body\n\t\t\tfiles[\"docs/rules.md\"] = body\n\t\t\tfiles[\"ignore\"] = \"*.gen.sql\\n**/*.gen.sql\\n\"",
  },
  {
    file: "scripts/graphify-pending/main_test.go",
    from: "\t\t\"拡張子パターンは深い階層にも当たる\":       {\"docs/adr/0001.ja.md\", \"*.ja.md\", true},\n\t\t\"拡張子パターンは正本に当たらない\":        {\"docs/adr/0001.md\", \"*.ja.md\", false},",
    to: "\t\t\"拡張子パターンは深い階層にも当たる\":       {\"docs/adr/0001.gen.sql\", \"*.gen.sql\", true},\n\t\t\"拡張子パターンは他の綴りに当たらない\":      {\"docs/adr/0001.md\", \"*.gen.sql\", false},",
  },
  {
    file: "scripts/graphify-pending/main_test.go",
    from: "\t\t\tassert.True(t, matchesGlob(\"rules.ja.md\", \"*.ja.md\"))",
    to: "\t\t\tassert.True(t, matchesGlob(\"rules.gen.sql\", \"*.gen.sql\"))",
  },
  {
    file: "AGENTS.ja.md",
    from: "**エージェントにとってのドキュメント範囲** —— 正典は英語の `README.md` と `docs/**/*.md` です。\n**`*.ja.md` ファイルは決して読まないでください。これらは正典の人間向け日本語訳であり、\n正典である英語の原文を読んでください。** ドキュメントポータルの UI アセットも無視します:\n\n```txt\n**/*.ja.md\ndocs/portal/**\n```",
    to: "**エージェントにとってのドキュメント範囲** —— 正典は `README.md` と `docs/**/*.md` です。\nドキュメントポータルの UI アセットは無視します:\n\n```txt\ndocs/portal/**\n```",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_JA,\n        block:\n          \"|SonarQube Cloud Scan|`sonarqube.yaml`|SonarQube Cloud による一次ソースの解析。結果は Web API から読み戻して SARIF へ変換する（**Sonar の品質ゲートでブロックする**。issue の一覧は報告専用。`SONAR_TOKEN` が必要。[資格情報を要するスキャナの撤去](#資格情報を要するスキャナの撤去)を参照）|\\n\",\n      },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_JA,\n        block:\n          \"| SonarQube Cloud | Go / TypeScript / `sonar-project.properties` 変更 PR | 同上 | 週次 |\\n\",\n      },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_JA,\n        block:\n          \"| `sonarqube.yaml` `sonarqube` | 15 | ベンダー側の解析キューが最大 10 分待つため。テストとカバレッジのゲートはそれぞれの所有ワークフローで実行する |\\n\",\n      },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_JA, fragment: \" + `sonarqube.yaml`（SonarQube Cloud） **(gate, 品質ゲート)**\" },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_JA, fragment: \"、`05:00` SonarQube Cloud\" },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_JA,\n        block:\n          \"| CodeQL | Go / TypeScript / Actions 定義の変更 PR | 同上 | 週次 |\\n\",\n      },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_JA,\n        block:\n          \"\\n最後のスロットは SonarQube Cloud です。解析がベンダーのサーバ側で走るため、DAST を全ファイル読み取り系の後ろへ置いたのと同じ理由で最後に並べています。所要時間がこのリポジトリの制御外のキューに左右されるため、自前のランナーで完結するスキャナより前に積む利点がありません。\\n\",\n      },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_JA,\n        block:\n          \"\\n`自前の Go ソース` と `自前の TypeScript ソース` の行にはベンダーホスト型のスキャナも乗っています。Sonar はこの表で唯一「ルール単位で担当 1 つ」から意図的に外れています。品質ゲートは静的解析・重複と Sonar 自身の issue 分類をまとめて判定し、カバレッジの閾値は Go / TypeScript のテストワークフローがそれぞれ担います。両者が認識する検出で PR が 2 回赤くなり得ますが、それを受け入れているのは、ベンダーの判定を捨てると「スキャンは報告するが run はそのままマージされる」状態になるためです。\\n\",\n      },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_JA, fragment: \"、`01:15` CodeQL\" },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_JA, fragment: \"`code-ql.yaml`（`javascript-typescript` レグ）+ \" },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_JA, fragment: \"+ `code-ql.yaml`（`actions` レグ）\" },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_JA, heading: \"#### 資格情報を要するスキャナの撤去\" },\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "const README_JA = \".github/workflows/README.ja.md\";\n",
    to: "",
    mode: "en",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_EN,\n        block:\n          \"|SonarQube Cloud Scan|`sonarqube.yaml`|SonarQube Cloud analysis of first-party source, read back over the Web API and converted to SARIF (**gates on Sonar's quality gate**, issue list report-only; needs `SONAR_TOKEN`, see [Removing the credential-bearing scanners](#removing-the-credential-bearing-scanners))|\\n\",\n      },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_EN,\n        block:\n          \"| SonarQube Cloud | Go / TypeScript / `sonar-project.properties`-change PRs | same as above | weekly |\\n\",\n      },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_EN,\n        block:\n          \"| `sonarqube.yaml` `sonarqube` | 15 | vendor-side analysis can queue for up to 10 minutes; test and coverage gates run in their owning workflows |\\n\",\n      },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_EN, fragment: \" + `sonarqube.yaml` (SonarQube Cloud) **(gate, quality gate)**\" },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_EN, fragment: \", `05:00` SonarQube Cloud\" },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_EN,\n        block:\n          \"| CodeQL | Go / TypeScript / Actions-definition-change PRs | same as above | weekly |\\n\",\n      },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_EN,\n        block:\n          \"| `code-ql.yaml` `codeql` | 30 | the limit covers whichever matrix leg is slowest, and no leg but `go` has a completed run to measure; `security-extended` is also a larger suite than the one the previous value was measured against |\\n\",\n      },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_EN,\n        block:\n          \"\\nSonarQube Cloud takes the last slot. Its analysis runs on a vendor's servers, and it is placed at the end for the same reason DAST is placed behind the file-reading scanners: its duration depends on a queue this repository does not control, so nothing useful is gained by having it queued ahead of a scanner that finishes on its own runner.\\n\",\n      },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      {\n        file: README_EN,\n        block:\n          \"\\nThe `First-party Go source` and `First-party TypeScript source` rows carry the vendor-hosted scanner as well. Sonar is the one deliberate departure from \\\"one owner per rule\\\" in this table. Its quality gate judges static analysis and duplication alongside its own issue taxonomy, while the Go and TypeScript test workflows own coverage thresholds. A finding both engines recognize can still turn a pull request red twice; that is accepted because discarding the vendor's verdict entirely would leave the scan reporting into a run that merged regardless.\\n\",\n      },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_EN, fragment: \", `01:15` CodeQL\" },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_EN, fragment: \"`code-ql.yaml` (`javascript-typescript` leg) + \" },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_EN, fragment: \" + `code-ql.yaml` (`actions` leg)\" },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "      { file: README_EN, heading: \"#### Removing the credential-bearing scanners\" },\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "const README_EN = \".github/workflows/README.md\";\n",
    to: "",
    mode: "ja",
  },
  {
    file: "scripts/setup/remove-licensed-scanners/scanner-manifest.ts",
    from: "const README_JA = \".github/workflows/README.ja.md\";",
    to: "const README_JA = \".github/workflows/README.md\";",
    mode: "ja",
  },
  {
    file: ".claude/skills/manage-skill/SKILL.md",
    from: " + mandatory `SKILL.ja.md` translation pair",
    to: "",
  },
  {
    file: ".claude/skills/manage-skill/SKILL.md",
    from: "ALWAYS use it before hand-editing a `SKILL.md` or `SKILL.ja.md`.",
    to: "ALWAYS use it before hand-editing a `SKILL.md`.",
  },
  {
    file: ".claude/skills/manage-skill/SKILL.md",
    from: "those have `sync-readme` / `canonicalize-doc` / `back-prop`",
    to: "those have `sync-readme` / `back-prop`",
  },
  {
    file: ".codex/skills/manage-skill/SKILL.md",
    from: " + mandatory `SKILL.ja.md` translation pair",
    to: "",
  },
  {
    file: ".codex/skills/manage-skill/SKILL.md",
    from: "ALWAYS use it before hand-editing a `SKILL.md` or `SKILL.ja.md`.",
    to: "ALWAYS use it before hand-editing a `SKILL.md`.",
  },
  {
    file: ".codex/skills/manage-skill/SKILL.md",
    from: "those have `sync-readme` / `canonicalize-doc` / `back-prop`",
    to: "those have `sync-readme` / `back-prop`",
  },
];

/**
 * 畳んだ後も残ってよい、対訳を名指す綴り（前後の空白を除いた完全一致）。
 *
 * @remarks
 * `.ja.md` で終わる綴りがすべて対訳を指すわけではありません。`tool-map` が書き出す
 * `TOOL_MAP.ja.md` は、このリポジトリのドキュメントではなくスキル自身の出力ファイルで、
 * 言語を畳んだ後もそのスキルは日本語版の地図を書き出せます。
 *
 * 除外は宣言でしか表せません。撤去は「消える名前を指す言及」を止めるので、止めてはいけない
 * ものはここで名指す必要があります —— `doc-ref-lint` の翻訳除外と同じ形です。
 */
export const ALLOWED_MENTIONS: readonly string[] = [
  "| `--output-path` | any relative path | `./TOOL_MAP.md` (en) or `./TOOL_MAP.ja.md` (ja) — only used if `--output=file` |",
  "| `--output-path` | 任意の相対パス | `./TOOL_MAP.md`（en）または `./TOOL_MAP.ja.md`（ja）— `--output=file` のときのみ使用 |",
];

/**
 * 機械では畳めず、行ごと落とすと宣言した行（前後の空白を除いた完全一致）。
 *
 * @remarks
 * ここに載るのは「対訳規約を説明している散文」のうち、行として独立していて、消しても前後の
 * 文が成立するものだけです。文の途中で対訳に触れている行——`manage-skill` / `sync-ai` の
 * 手順のように、対訳を含む節を別の言い回しへ差し替える必要があるもの——はここではなく、
 * 本文側の `doc-pair:replace-begin` / `replace-with` / `replace-end` が持ちます。
 *
 * 宣言を最小に保つのは、これが完全一致だからです。本文が 1 文字動けば当たらなくなり、
 * そのとき撤去は黙って素通りせず止まります（それが宣言を完全一致にしている理由でもあります）。
 */
export const DECLARED_LINES: readonly string[] = [
  "| skill | `SKILL.md` | `SKILL.ja.md` |",
  "- `SKILL.ja.md` (translation, Japanese):",
  "- Claude Code skill files: `SKILL.md` / `SKILL.ja.md` (under `.claude/skills/<name>/`)",
  "- If creating a `SKILL.md`, ensure it contains the one-line pointer to `SKILL.ja.md`.",
  "- If creating a `SKILL.ja.md`, ensure the leading blockquote sync note is present.",
];
