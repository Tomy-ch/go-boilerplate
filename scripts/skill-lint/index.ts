#!/usr/bin/env -S tsx
// `.claude/**` のスキル / エージェント定義を意味的に検査し、あわせて `.codex/**` との対応を検査する
// lint スクリプト。markdownlint は体裁しか見ないため、「書いてある内容が実態と合っているか」は誰も
// 検査していない。スキル定義はエージェントの挙動を決める指示書であり、腐った参照はそのまま誤った
// 手順の実行につながる。片側の環境にしか無い定義も同様で、その環境を使ったときだけ古い手順を踏む。
//
// 検査は Makefile のターゲット一覧・ファイルシステム・見出し抽出から導出できるものだけに限る
// （判断を含めない）。1 件でも違反があれば非 0 で終了する。

import fs from "node:fs";
import path from "node:path";

import {
  CONFIG_FILE_RE,
  EXCLUDE_DIRS,
  IGNORE_DIRECTIVE,
  PATH_ROOT_DENY,
  PLATFORM_ONLY_SKILLS,
  WILDCARD_RE,
  allowlistLocation,
  asRepoPath,
  collectMakeTargets,
  compareHeadingStructure,
  eachLineOutsideFence,
  expandBraces,
  extractHeadings,
  extractInlineCode,
  extractMakeTargets,
  hasTranslationNote,
  makeTargetExists,
  onlyIn,
  parseFrontmatterKeys,
  placeholderToRegExp,
  splitFrontmatter,
} from "./checks";

const REPO_ROOT = process.cwd();
const CLAUDE_DIR = ".claude";
const SKILLS_DIR = path.join(CLAUDE_DIR, "skills");
const AGENTS_DIR = path.join(CLAUDE_DIR, "agents");
const CODEX_DIR = ".codex";
const CODEX_SKILLS_DIR = path.join(CODEX_DIR, "skills");
const CODEX_AGENTS_DIR = path.join(CODEX_DIR, "agents");

const ALLOWLIST_REL = allowlistLocation(REPO_ROOT);

type Finding = {
  file: string;
  line: number;
  rule: string;
  message: string;
};

const findings: Finding[] = [];

function report(file: string, line: number, rule: string, message: string): void {
  findings.push({ file, line, rule, message });
}

function readFile(rel: string): string {
  return fs.readFileSync(path.join(REPO_ROOT, rel), "utf8");
}

function exists(rel: string): boolean {
  return fs.existsSync(path.join(REPO_ROOT, rel));
}

function listDirs(rel: string): string[] {
  if (!exists(rel)) return [];

  return fs
    .readdirSync(path.join(REPO_ROOT, rel), { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .sort();
}

// repoRoot 配下の全エントリ（ファイル + ディレクトリ）をリポジトリ相対パスで索引化する。
// glob / プレースホルダを含む参照は実パス解決ができないため、この索引に対する正規表現照合で判定する。
function buildEntryIndex(): string[] {
  const entries: string[] = [];

  const walk = (dir: string) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = path.join(dir, entry.name);
      entries.push(path.relative(REPO_ROOT, abs));

      if (entry.isDirectory()) {
        if (EXCLUDE_DIRS.has(entry.name)) continue;
        walk(abs);
      }
    }
  };

  walk(REPO_ROOT);
  return entries;
}

// -----------------------------------------------------------------------------
// 個別の検査
// -----------------------------------------------------------------------------

// name / description の必須検査とディレクトリ（ファイル）名との一致検査。
function checkFrontmatter(rel: string, content: string, expectedName: string): void {
  const frontmatter = splitFrontmatter(content);

  if (!frontmatter) {
    report(rel, 1, "frontmatter", "frontmatter (`---` で囲まれたブロック) がありません");
    return;
  }

  const keys = parseFrontmatterKeys(frontmatter.lines);

  for (const required of ["name", "description"]) {
    if (!keys.has(required) || keys.get(required) === "") {
      report(rel, 1, "frontmatter", `frontmatter に \`${required}\` がありません（または空です）`);
    }
  }

  const name = keys.get("name");
  if (name !== undefined && name !== "" && name !== expectedName) {
    report(
      rel,
      1,
      "frontmatter",
      `frontmatter の \`name: ${name}\` が配置名 \`${expectedName}\` と一致しません`,
    );
  }
}

// 対訳（SKILL.ja.md）が canonical（SKILL.md）と 1:1 であることを検査する。
function checkTranslationPair(canonicalRel: string, translationRel: string): void {
  if (!exists(translationRel)) {
    report(canonicalRel, 1, "translation", `対訳 \`${path.basename(translationRel)}\` がありません`);
    return;
  }

  const translation = readFile(translationRel);

  if (splitFrontmatter(translation)) {
    report(
      translationRel,
      1,
      "translation",
      "対訳に frontmatter があります（スキルとして読み込まれるのは canonical 側だけです）",
    );
  }

  if (!hasTranslationNote(translation, path.basename(canonicalRel))) {
    report(
      translationRel,
      1,
      "translation",
      `冒頭に canonical (\`${path.basename(canonicalRel)}\`) を指す翻訳注記（引用行）がありません`,
    );
  }

  const canonicalHeadings = extractHeadings(readFile(canonicalRel));
  const translationHeadings = extractHeadings(translation);
  const mismatch = compareHeadingStructure(canonicalHeadings, translationHeadings);

  if (mismatch === null) return;

  const describe = (heading: (typeof canonicalHeadings)[number] | null) =>
    heading ? `L${heading.lineNo} ${"#".repeat(heading.level)} ${heading.text}` : "（無し）";

  report(
    translationRel,
    mismatch.translation ? mismatch.translation.lineNo : 1,
    "translation",
    `見出し構造が canonical とずれています（${mismatch.index + 1} 番目 / canonical ${canonicalHeadings.length} 見出し・対訳 ${translationHeadings.length} 見出し）\n` +
      `      canonical: ${describe(mismatch.canonical)}\n` +
      `      対訳:      ${describe(mismatch.translation)}`,
  );
}

// 環境間で検査するのは「存在」だけで、本文の追随は見ない。
// `sync-ai` は逐語コピーではなく意味ポートであり、`CLAUDE.md` ↔ `AGENTS.md` の言い換え・Claude 固有機構
// （`AskUserQuestion` / `Agent`）の適応・凝縮スタイルへの書き下ろしといった意図的な差分が常に残る。
// 見出しテキスト集合の一致まで求めても、その意図的な差分を違反として弾くだけで検査にならない。
// 存在の対応だけなら例外は宣言可能な件数に収まり、片側だけをマージする事故は確実に捕まる。
function checkSkillParity(claudeSkills: string[], codexSkills: string[]): void {
  const guidance =
    `      移植するなら \`sync-ai\` を実行し、意図的に片側だけへ置くなら ${ALLOWLIST_REL} の \`PLATFORM_ONLY_SKILLS\` へ理由付きで登録してください`;

  for (const name of onlyIn(claudeSkills, codexSkills)) {
    if (PLATFORM_ONLY_SKILLS.has(name)) continue;
    report(
      path.join(SKILLS_DIR, name),
      1,
      "cross-env",
      `対応する \`${path.join(CODEX_SKILLS_DIR, name)}/\` がありません\n${guidance}`,
    );
  }

  for (const name of onlyIn(codexSkills, claudeSkills)) {
    if (PLATFORM_ONLY_SKILLS.has(name)) continue;
    report(
      path.join(CODEX_SKILLS_DIR, name),
      1,
      "cross-env",
      `対応する \`${path.join(SKILLS_DIR, name)}/\` がありません\n${guidance}`,
    );
  }
}

// agent 定義の環境間対応を検査する。拡張子は環境ごとに異なる（Claude: `.md` / Codex: `.toml`）ため、
// 拡張子を落とした名前で突き合わせる。
function checkAgentParity(claudeAgents: string[], codexAgents: string[]): void {
  for (const name of onlyIn(claudeAgents, codexAgents)) {
    report(
      path.join(AGENTS_DIR, `${name}.md`),
      1,
      "cross-env",
      `対応する \`${path.join(CODEX_AGENTS_DIR, `${name}.toml`)}\` がありません`,
    );
  }

  for (const name of onlyIn(codexAgents, claudeAgents)) {
    report(
      path.join(CODEX_AGENTS_DIR, `${name}.toml`),
      1,
      "cross-env",
      `対応する \`${path.join(AGENTS_DIR, `${name}.md`)}\` がありません`,
    );
  }
}

// allowlist 自体の腐りを検査する。理由の無い登録を認めず、移植が済んだ登録は消させる。
function checkPlatformOnlyAllowlist(claudeSkills: string[], codexSkills: string[]): void {
  const claude = new Set(claudeSkills);
  const codex = new Set(codexSkills);

  for (const [name, reason] of PLATFORM_ONLY_SKILLS) {
    if (reason.trim() === "") {
      report(
        ALLOWLIST_REL,
        1,
        "cross-env",
        `\`PLATFORM_ONLY_SKILLS\` の \`${name}\` に理由がありません（理由なしの片側運用は認めません）`,
      );
    }

    if (claude.has(name) && codex.has(name)) {
      report(
        ALLOWLIST_REL,
        1,
        "cross-env",
        `\`PLATFORM_ONLY_SKILLS\` の \`${name}\` は両環境に存在します（移植済みなので登録を削除してください）`,
      );
      continue;
    }

    if (!claude.has(name) && !codex.has(name)) {
      report(
        ALLOWLIST_REL,
        1,
        "cross-env",
        `\`PLATFORM_ONLY_SKILLS\` の \`${name}\` はどちらの環境にも存在しません（登録を削除してください）`,
      );
    }
  }
}

// Codex の 1 スキルを構成する必須ファイルを検査する（`.codex/README.md` の Layout 表が正）。
// 対訳（`SKILL.ja.md`）は Codex 側では任意なので、欠落は報告せず、在るときだけ構造を検査する。
function checkCodexSkillStructure(name: string): void {
  const canonicalRel = path.join(CODEX_SKILLS_DIR, name, "SKILL.md");
  const metadataRel = path.join(CODEX_SKILLS_DIR, name, "agents", "openai.yaml");
  const canonicalExists = exists(canonicalRel);

  if (!canonicalExists) {
    report(path.join(CODEX_SKILLS_DIR, name), 1, "structure", "`SKILL.md` がありません");
  }
  if (!exists(metadataRel)) {
    report(
      path.join(CODEX_SKILLS_DIR, name),
      1,
      "structure",
      "`agents/openai.yaml`（Codex UI メタデータ）がありません",
    );
  }

  const translationRel = path.join(CODEX_SKILLS_DIR, name, "SKILL.ja.md");
  if (canonicalExists && exists(translationRel)) {
    checkTranslationPair(canonicalRel, translationRel);
  }
}

// -----------------------------------------------------------------------------
// 参照の実在性
// -----------------------------------------------------------------------------

function readMakefileSources(): string[] {
  const files: string[] = [];

  const walk = (dir: string) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        walk(abs);
        continue;
      }
      if (entry.name.endsWith(".mk")) files.push(abs);
    }
  };

  const makefilesDir = path.join(REPO_ROOT, ".makefiles");
  if (fs.existsSync(makefilesDir)) walk(makefilesDir);

  // ルートの makefile は綴りが処理系依存（`makefile` / `Makefile`）。macOS は大文字小文字を
  // 区別しないため、Linux コンテナで初めて読み落とすことがないよう実エントリ名で拾う。
  for (const name of fs.readdirSync(REPO_ROOT)) {
    if (name === "makefile" || name === "Makefile" || name === "GNUmakefile") {
      files.push(path.join(REPO_ROOT, name));
    }
  }

  return files.map((file) => fs.readFileSync(file, "utf8"));
}

const makeTargets = collectMakeTargets(readMakefileSources());
const entryIndex = buildEntryIndex();
const rootEntries = new Set(fs.readdirSync(REPO_ROOT).filter((name) => !PATH_ROOT_DENY.has(name)));
const basenameIndex = new Set(entryIndex.map((entry) => path.basename(entry)));

function configFileExists(name: string): boolean {
  return expandBraces(name).some((candidate) => {
    if (!WILDCARD_RE.test(candidate)) return basenameIndex.has(candidate);

    const pattern = placeholderToRegExp(candidate, { segmentSeparator: true });
    return [...basenameIndex].some((base) => pattern.test(base));
  });
}

// パス参照の実在性を判定する。スキルは自身が同梱するファイル（`scripts/run.sh` など）も
// 同じ表記で参照するため、リポジトリルート相対に加えて参照元ファイルのディレクトリ相対でも解決する。
function repoPathExists(candidate: string, fromDir: string): boolean {
  const bases = [REPO_ROOT, path.join(REPO_ROOT, fromDir)];

  return expandBraces(candidate).some((text) => {
    if (!WILDCARD_RE.test(text)) return bases.some((base) => fs.existsSync(path.join(base, text)));

    const pattern = placeholderToRegExp(text, { segmentSeparator: true });
    return entryIndex.some((entry) => pattern.test(entry));
  });
}

// `.claude/**` の Markdown 本文が参照する make ターゲット / ファイルパスの実在性を検査する。
function checkReferences(rel: string): void {
  const content = readFile(rel);
  const fromDir = path.dirname(rel);

  for (const { line, lineNo } of eachLineOutsideFence(content)) {
    if (line.includes(IGNORE_DIRECTIVE)) continue;

    for (const span of extractInlineCode(line)) {
      for (const target of extractMakeTargets(span)) {
        if (!makeTargetExists(target, makeTargets)) {
          report(rel, lineNo, "make-ref", `存在しない make ターゲットを参照しています: \`make ${target}\``);
        }
      }

      const repoPath = asRepoPath(span, rootEntries, exists);

      if (repoPath !== null && !repoPathExists(repoPath, fromDir)) {
        report(rel, lineNo, "path-ref", `存在しないパスを参照しています: \`${span}\``);
        continue;
      }
      if (repoPath === null && CONFIG_FILE_RE.test(span) && !configFileExists(span)) {
        report(rel, lineNo, "path-ref", `リポジトリに存在しない設定ファイルを参照しています: \`${span}\``);
      }
    }
  }
}

function collectClaudeMarkdown(): string[] {
  const out: string[] = [];

  const walk = (dir: string) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = path.join(dir, entry.name);

      if (entry.isDirectory()) {
        if (EXCLUDE_DIRS.has(entry.name)) continue;
        walk(abs);
        continue;
      }
      if (entry.name.endsWith(".md")) out.push(path.relative(REPO_ROOT, abs));
    }
  };

  if (!exists(CLAUDE_DIR)) {
    console.error(`✘ skill-lint: ${CLAUDE_DIR}/ が見つかりません（リポジトリルートで実行してください）`);
    process.exit(2);
  }

  walk(path.join(REPO_ROOT, CLAUDE_DIR));
  return out.sort();
}

// -----------------------------------------------------------------------------
// 実行
// -----------------------------------------------------------------------------

const skillDirs = listDirs(SKILLS_DIR);

for (const name of skillDirs) {
  const canonicalRel = path.join(SKILLS_DIR, name, "SKILL.md");

  if (!exists(canonicalRel)) {
    report(path.join(SKILLS_DIR, name), 1, "structure", "`SKILL.md` がありません");
    continue;
  }

  checkFrontmatter(canonicalRel, readFile(canonicalRel), name);
  checkTranslationPair(canonicalRel, path.join(SKILLS_DIR, name, "SKILL.ja.md"));
}

const agentFiles = exists(AGENTS_DIR)
  ? fs
      .readdirSync(path.join(REPO_ROOT, AGENTS_DIR))
      .filter((name) => name.endsWith(".md") && !name.endsWith(".ja.md"))
      .sort()
  : [];

for (const file of agentFiles) {
  const rel = path.join(AGENTS_DIR, file);
  checkFrontmatter(rel, readFile(rel), file.replace(/\.md$/, ""));
}

const codexSkillDirs = listDirs(CODEX_SKILLS_DIR);
for (const name of codexSkillDirs) checkCodexSkillStructure(name);

const codexAgentFiles = exists(CODEX_AGENTS_DIR)
  ? fs
      .readdirSync(path.join(REPO_ROOT, CODEX_AGENTS_DIR))
      .filter((name) => name.endsWith(".toml"))
      .sort()
  : [];

checkSkillParity(skillDirs, codexSkillDirs);
checkAgentParity(
  agentFiles.map((file) => file.replace(/\.md$/, "")),
  codexAgentFiles.map((file) => file.replace(/\.toml$/, "")),
);
checkPlatformOnlyAllowlist(skillDirs, codexSkillDirs);

const markdownFiles = collectClaudeMarkdown();
for (const rel of markdownFiles) checkReferences(rel);

const summary =
  `${CLAUDE_DIR} ${skillDirs.length} スキル / ${agentFiles.length} エージェント / ${markdownFiles.length} Markdown、` +
  `${CODEX_DIR} ${codexSkillDirs.length} スキル / ${codexAgentFiles.length} エージェント`;

if (findings.length > 0) {
  console.error(`✘ skill-lint: ${findings.length} 件の違反\n`);
  let current: string | null = null;

  for (const finding of findings.sort(
    (left, right) => left.file.localeCompare(right.file) || left.line - right.line,
  )) {
    if (finding.file !== current) {
      if (current !== null) console.error("");
      console.error(`  ${finding.file}`);
      current = finding.file;
    }
    console.error(`    :${finding.line}  [${finding.rule}] ${finding.message}`);
  }

  console.error(`\n検査 ${summary} 中 ${findings.length} 件 NG`);
  process.exit(1);
}

console.log(`✓ skill-lint: ${summary} すべて OK`);
