#!/usr/bin/env -S tsx
// `.claude/**` のスキル / エージェント定義と `.codex/**` との対応を検査する lint スクリプト。
// 検査範囲と機構（判断を含めず機械的に導けるものだけを見る）は scripts/README.md の
// Skill Lint 節が持つ。1 件でも違反があれば非 0 で終了する。
//
// この入口はファイル入出力と終了コードだけを担う。個々の検査は rules.ts、字句の切り出しは
// checks.ts にある。

import fs from "node:fs";
import path from "node:path";

import {
  CONFIG_FILE_RE,
  EXCLUDE_DIRS,
  PATH_ROOT_DENY,
  PLATFORM_ONLY_SKILLS,
  WILDCARD_RE,
  allowlistLocation,
  asRepoPath,
  collectMakeTargets,
  expandBraces,
  literalParentDir,
  makeTargetExists,
  placeholderToRegExp,
} from "./checks";
import {
  type EnvLayout,
  type Finding,
  agentName,
  checkAgentParity,
  checkCodexSkillStructure,
  checkFrontmatter,
  checkPlatformOnlyAllowlist,
  checkReferences,
  checkSkillParity,
  checkTranslationPair,
  formatFindings,
  isClaudeAgentDefinition,
  isCodexAgentDefinition,
  isMakefileFragment,
  isRootMakefile,
} from "./rules";

const REPO_ROOT = process.cwd();
const CLAUDE_DIR = ".claude";
const CODEX_DIR = ".codex";
const LAYOUT: EnvLayout = {
  claudeSkills: path.join(CLAUDE_DIR, "skills"),
  claudeAgents: path.join(CLAUDE_DIR, "agents"),
  codexSkills: path.join(CODEX_DIR, "skills"),
  codexAgents: path.join(CODEX_DIR, "agents"),
};

const ALLOWLIST_REL = allowlistLocation(REPO_ROOT);
const findings: Finding[] = [];

function readFile(rel: string): string {
  return fs.readFileSync(path.join(REPO_ROOT, rel), "utf8");
}

function exists(rel: string): boolean {
  return fs.existsSync(path.join(REPO_ROOT, rel));
}

/** 存在すれば内容を、無ければ null を返す。検査側が「無い」を値として受け取れるようにする。 */
function readIfPresent(rel: string): string | null {
  return exists(rel) ? readFile(rel) : null;
}

function listDirs(rel: string): string[] {
  if (!exists(rel)) return [];

  return fs
    .readdirSync(path.join(REPO_ROOT, rel), { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .sort();
}

function listFiles(rel: string, accept: (name: string) => boolean): string[] {
  if (!exists(rel)) return [];

  return fs.readdirSync(path.join(REPO_ROOT, rel)).filter(accept).sort();
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

function readMakefileSources(): string[] {
  const files: string[] = [];

  const walk = (dir: string) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const abs = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        walk(abs);
        continue;
      }
      if (isMakefileFragment(entry.name)) files.push(abs);
    }
  };

  const makefilesDir = path.join(REPO_ROOT, ".makefiles");
  if (fs.existsSync(makefilesDir)) walk(makefilesDir);

  for (const name of fs.readdirSync(REPO_ROOT)) {
    if (isRootMakefile(name)) files.push(path.join(REPO_ROOT, name));
  }

  return files.map((file) => fs.readFileSync(file, "utf8"));
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
    if (entryIndex.some((entry) => pattern.test(entry))) return true;

    // 一致が無いのは「その形の実体がまだ無い」だけでありうる。置き場所が在れば参照は生きている。
    const parent = literalParentDir(text);

    return parent !== null && bases.some((base) => fs.existsSync(path.join(base, parent)));
  });
}

const resolvers = {
  makeTargetExists: (target: string) => makeTargetExists(target, makeTargets),
  repoPathExists,
  configFileExists,
  asRepoPath: (span: string) => asRepoPath(span, rootEntries, exists),
};

// -----------------------------------------------------------------------------
// 実行
// -----------------------------------------------------------------------------

const skillDirs = listDirs(LAYOUT.claudeSkills);

for (const name of skillDirs) {
  const canonicalRel = path.join(LAYOUT.claudeSkills, name, "SKILL.md");

  if (!exists(canonicalRel)) {
    findings.push({
      file: path.join(LAYOUT.claudeSkills, name),
      line: 1,
      rule: "structure",
      message: "`SKILL.md` がありません",
    });
    continue;
  }

  const canonical = readFile(canonicalRel);
  const translationRel = path.join(LAYOUT.claudeSkills, name, "SKILL.ja.md");

  findings.push(
    ...checkFrontmatter(canonicalRel, canonical, name),
    ...checkTranslationPair(canonicalRel, translationRel, canonical, readIfPresent(translationRel)),
  );
}

const agentFiles = listFiles(LAYOUT.claudeAgents, isClaudeAgentDefinition);

for (const file of agentFiles) {
  const rel = path.join(LAYOUT.claudeAgents, file);
  findings.push(...checkFrontmatter(rel, readFile(rel), agentName(file)));
}

const codexSkillDirs = listDirs(LAYOUT.codexSkills);

for (const name of codexSkillDirs) {
  const skillDir = path.join(LAYOUT.codexSkills, name);

  findings.push(
    ...checkCodexSkillStructure(skillDir, {
      canonical: readIfPresent(path.join(skillDir, "SKILL.md")),
      hasMetadata: exists(path.join(skillDir, "agents", "openai.yaml")),
      translation: readIfPresent(path.join(skillDir, "SKILL.ja.md")),
    }),
  );
}

const codexAgentFiles = listFiles(LAYOUT.codexAgents, isCodexAgentDefinition);

findings.push(
  ...checkSkillParity(skillDirs, codexSkillDirs, PLATFORM_ONLY_SKILLS, LAYOUT, ALLOWLIST_REL),
  ...checkAgentParity(agentFiles.map(agentName), codexAgentFiles.map(agentName), LAYOUT),
  ...checkPlatformOnlyAllowlist(skillDirs, codexSkillDirs, PLATFORM_ONLY_SKILLS, ALLOWLIST_REL),
);

const markdownFiles = collectClaudeMarkdown();
for (const rel of markdownFiles) {
  findings.push(...checkReferences(rel, readFile(rel), resolvers));
}

const summary =
  `${CLAUDE_DIR} ${skillDirs.length} スキル / ${agentFiles.length} エージェント / ${markdownFiles.length} Markdown、` +
  `${CODEX_DIR} ${codexSkillDirs.length} スキル / ${codexAgentFiles.length} エージェント`;

if (findings.length > 0) {
  console.error(`✘ skill-lint: ${findings.length} 件の違反\n`);
  console.error(formatFindings(findings));
  console.error(`\n検査 ${summary} 中 ${findings.length} 件 NG`);
  process.exit(1);
}

console.log(`✓ skill-lint: ${summary} すべて OK`);
