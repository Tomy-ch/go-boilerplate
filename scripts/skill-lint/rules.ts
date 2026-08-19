// スキル / エージェント定義に対する個々の検査。
//
// ファイルシステムへは触らず、読み込み済みの内容と存在の可否だけを受け取る
//（この分割の理由は scripts/README.md の Test Strategy が持つ）。
import path from "node:path";

import {
  CONFIG_FILE_RE,
  IGNORE_DIRECTIVE,
  type Heading,
  compareHeadingStructure,
  eachLineOutsideFence,
  extractHeadings,
  extractInlineCode,
  extractMakeTargets,
  hasTranslationNote,
  onlyIn,
  parseFrontmatterKeys,
  splitFrontmatter,
} from "./checks";

/** 違反 1 件。`rule` は違反の種別で、出力に添えて直し方の見当をつけさせる。 */
export type Finding = {
  file: string;
  line: number;
  rule: string;
  message: string;
};

/** frontmatter に必須のキー。空文字も欠落として扱う。 */
const REQUIRED_FRONTMATTER_KEYS = ["name", "description"] as const;

/**
 * frontmatter の必須キーと、配置名との一致を検査する。
 *
 * @remarks
 * `name` が配置名（ディレクトリ名 / ファイル名）とずれると、エージェントは定義を見つけられない
 * まま「そのスキルは無い」と判断します。読み込み側は黙って落ちるので、ここで止めます。
 */
export function checkFrontmatter(rel: string, content: string, expectedName: string): Finding[] {
  const frontmatter = splitFrontmatter(content);

  if (!frontmatter) {
    return [{ file: rel, line: 1, rule: "frontmatter", message: "frontmatter (`---` で囲まれたブロック) がありません" }];
  }

  const keys = parseFrontmatterKeys(frontmatter.lines);
  const findings: Finding[] = [];

  for (const required of REQUIRED_FRONTMATTER_KEYS) {
    if (!keys.has(required) || keys.get(required) === "") {
      findings.push({
        file: rel,
        line: 1,
        rule: "frontmatter",
        message: `frontmatter に \`${required}\` がありません（または空です）`,
      });
    }
  }

  const name = keys.get("name");
  if (name !== undefined && name !== "" && name !== expectedName) {
    findings.push({
      file: rel,
      line: 1,
      rule: "frontmatter",
      message: `frontmatter の \`name: ${name}\` が配置名 \`${expectedName}\` と一致しません`,
    });
  }

  return findings;
}

/** 見出しを「L12 ## 見出し」の形で表す。差分のどちら側が欠けているかを読めるようにする。 */
function describeHeading(heading: Heading | null): string {
  return heading ? `L${heading.lineNo} ${"#".repeat(heading.level)} ${heading.text}` : "（無し）";
}

/**
 * 対訳が canonical と 1:1 であることを検査する。`translation` が null なら対訳が存在しない。
 *
 * @remarks
 * 見るのは frontmatter の不在・翻訳注記・見出し構造の 3 点です。本文の一致までは求めません。
 * 対訳は逐語コピーではないので、そこまで求めると意図した訳し分けを違反として弾くだけになります。
 */
export function checkTranslationPair(
  canonicalRel: string,
  translationRel: string,
  canonicalContent: string,
  translation: string | null,
): Finding[] {
  if (translation === null) {
    return [
      {
        file: canonicalRel,
        line: 1,
        rule: "translation",
        message: `対訳 \`${path.basename(translationRel)}\` がありません`,
      },
    ];
  }

  const findings: Finding[] = [];

  if (splitFrontmatter(translation)) {
    findings.push({
      file: translationRel,
      line: 1,
      rule: "translation",
      message: "対訳に frontmatter があります（スキルとして読み込まれるのは canonical 側だけです）",
    });
  }

  if (!hasTranslationNote(translation, path.basename(canonicalRel))) {
    findings.push({
      file: translationRel,
      line: 1,
      rule: "translation",
      message: `冒頭に canonical (\`${path.basename(canonicalRel)}\`) を指す翻訳注記（引用行）がありません`,
    });
  }

  const canonicalHeadings = extractHeadings(canonicalContent);
  const translationHeadings = extractHeadings(translation);
  const mismatch = compareHeadingStructure(canonicalHeadings, translationHeadings);

  if (mismatch === null) return findings;

  findings.push({
    file: translationRel,
    line: mismatch.translation ? mismatch.translation.lineNo : 1,
    rule: "translation",
    message:
      `見出し構造が canonical とずれています（${mismatch.index + 1} 番目 / canonical ${canonicalHeadings.length} 見出し・対訳 ${translationHeadings.length} 見出し）\n` +
      `      canonical: ${describeHeading(mismatch.canonical)}\n` +
      `      対訳:      ${describeHeading(mismatch.translation)}`,
  });

  return findings;
}

/** 環境間の対応検査で使うディレクトリ配置。環境ごとに違う置き場を呼び出し側が決める。 */
export type EnvLayout = {
  claudeSkills: string;
  codexSkills: string;
  claudeAgents: string;
  codexAgents: string;
};

/**
 * スキルが両環境に揃っているかを検査する。
 *
 * @remarks
 * 見るのは存在だけ。本文の追随を見ない理由は scripts/README.md の Skill Lint 節が持ちます。
 */
export function checkSkillParity(
  claudeSkills: readonly string[],
  codexSkills: readonly string[],
  allowlist: ReadonlyMap<string, string>,
  layout: EnvLayout,
  allowlistRel: string,
): Finding[] {
  const guidance = `      移植するなら \`sync-ai\` を実行し、意図的に片側だけへ置くなら ${allowlistRel} の \`PLATFORM_ONLY_SKILLS\` へ理由付きで登録してください`;
  const findings: Finding[] = [];

  const missing = (names: readonly string[], from: string, to: string): void => {
    for (const name of names) {
      if (allowlist.has(name)) continue;
      findings.push({
        file: path.join(from, name),
        line: 1,
        rule: "cross-env",
        message: `対応する \`${path.join(to, name)}/\` がありません\n${guidance}`,
      });
    }
  };

  missing(onlyIn(claudeSkills, codexSkills), layout.claudeSkills, layout.codexSkills);
  missing(onlyIn(codexSkills, claudeSkills), layout.codexSkills, layout.claudeSkills);

  return findings;
}

/**
 * エージェント定義が両環境に揃っているかを検査する。
 *
 * @remarks
 * 拡張子は環境ごとに違う（Claude: `.md` / Codex: `.toml`）ため、拡張子を落とした名前で
 * 突き合わせます。例外の仕組みを持たない理由は scripts/README.md の Skill Lint 節が持ちます。
 */
export function checkAgentParity(
  claudeAgents: readonly string[],
  codexAgents: readonly string[],
  layout: EnvLayout,
): Finding[] {
  const findings: Finding[] = [];

  for (const name of onlyIn(claudeAgents, codexAgents)) {
    const counterpart = path.join(layout.codexAgents, `${name}.toml`);

    findings.push({
      file: path.join(layout.claudeAgents, `${name}.md`),
      line: 1,
      rule: "cross-env",
      message: `対応する \`${counterpart}\` がありません`,
    });
  }

  for (const name of onlyIn(codexAgents, claudeAgents)) {
    const counterpart = path.join(layout.claudeAgents, `${name}.md`);

    findings.push({
      file: path.join(layout.codexAgents, `${name}.toml`),
      line: 1,
      rule: "cross-env",
      message: `対応する \`${counterpart}\` がありません`,
    });
  }

  return findings;
}

/**
 * 例外リスト自体の腐りを検査する。
 *
 * @remarks
 * 理由の無い登録・移植が済んだ登録・どちらの環境からも消えた登録を削除させます
 *（理由は scripts/README.md の Skill Lint 節）。
 */
export function checkPlatformOnlyAllowlist(
  claudeSkills: readonly string[],
  codexSkills: readonly string[],
  allowlist: ReadonlyMap<string, string>,
  allowlistRel: string,
): Finding[] {
  const claude = new Set(claudeSkills);
  const codex = new Set(codexSkills);
  const findings: Finding[] = [];
  const report = (message: string): void => {
    findings.push({ file: allowlistRel, line: 1, rule: "cross-env", message });
  };

  for (const [name, reason] of allowlist) {
    if (reason.trim() === "") {
      report(`\`PLATFORM_ONLY_SKILLS\` の \`${name}\` に理由がありません（理由なしの片側運用は認めません）`);
    }

    if (claude.has(name) && codex.has(name)) {
      report(`\`PLATFORM_ONLY_SKILLS\` の \`${name}\` は両環境に存在します（移植済みなので登録を削除してください）`);
      continue;
    }

    if (!claude.has(name) && !codex.has(name)) {
      report(`\`PLATFORM_ONLY_SKILLS\` の \`${name}\` はどちらの環境にも存在しません（登録を削除してください）`);
    }
  }

  return findings;
}

/** Codex の 1 スキルを構成するファイルの有無と内容。読み取りは呼び出し側が行う。 */
export type CodexSkillFiles = {
  canonical: string | null;
  hasMetadata: boolean;
  translation: string | null;
};

/**
 * Codex の 1 スキルの必須ファイルを検査する（`.codex/README.md` の Layout 表が正）。
 *
 * @remarks
 * 対訳は Claude 側と同じく必須で、欠落そのものを報告します。任意だった頃は 24 スキルが
 * 「対訳は `SKILL.ja.md` にある」と書きながらファイルを持たない状態で緑を返し続けました。
 */
export function checkCodexSkillStructure(
  skillDir: string,
  files: CodexSkillFiles,
): Finding[] {
  const findings: Finding[] = [];

  if (files.canonical === null) {
    findings.push({ file: skillDir, line: 1, rule: "structure", message: "`SKILL.md` がありません" });
  }
  if (!files.hasMetadata) {
    findings.push({
      file: skillDir,
      line: 1,
      rule: "structure",
      message: "`agents/openai.yaml`（Codex UI メタデータ）がありません",
    });
  }

  if (files.canonical !== null) {
    findings.push(
      ...checkTranslationPair(
        path.join(skillDir, "SKILL.md"),
        path.join(skillDir, "SKILL.ja.md"),
        files.canonical,
        files.translation,
      ),
    );
  }

  return findings;
}

/** 参照の実在性を答える述語一式。実解決は入口が持ち、ここは問い合わせるだけ。 */
export type ReferenceResolvers = {
  makeTargetExists: (target: string) => boolean;
  repoPathExists: (candidate: string, fromDir: string) => boolean;
  configFileExists: (name: string) => boolean;
  asRepoPath: (span: string) => string | null;
};

/**
 * Markdown 本文が参照する make ターゲット / パスの実在性を検査する。
 *
 * @remarks
 * 読む範囲（フェンス外のみ）と ignore ディレクティブの扱いは scripts/README.md の
 * Skill Lint 節が持ちます。
 */
export function checkReferences(
  rel: string,
  content: string,
  resolvers: ReferenceResolvers,
): Finding[] {
  const fromDir = path.dirname(rel);

  return [...eachLineOutsideFence(content)]
    .filter(({ line }) => !line.includes(IGNORE_DIRECTIVE))
    .flatMap(({ line, lineNo }) =>
      extractInlineCode(line).flatMap((span) => [
        ...missingMakeTargets(span, resolvers).map((target) => ({
          file: rel,
          line: lineNo,
          rule: "make-ref",
          message: `存在しない make ターゲットを参照しています: \`make ${target}\``,
        })),
        ...missingPath(span, fromDir, resolvers).map((message) => ({
          file: rel,
          line: lineNo,
          rule: "path-ref",
          message,
        })),
      ]),
    );
}

/** inline code span 1 つが参照している make ターゲットのうち、実在しないもの。 */
function missingMakeTargets(span: string, resolvers: ReferenceResolvers): string[] {
  return extractMakeTargets(span).filter((target) => !resolvers.makeTargetExists(target));
}

/**
 * inline code span 1 つがパスとして読めるとき、実在しなければその違反文を返す。
 *
 * リポジトリ相対のパスとして読めた場合はそちらの実在だけを見て、設定ファイル名としての
 * 解決は行いません。両方に当たると同じ span へ 2 つの違反が出ます。
 */
function missingPath(span: string, fromDir: string, resolvers: ReferenceResolvers): string[] {
  const repoPath = resolvers.asRepoPath(span);

  if (repoPath !== null) {
    return resolvers.repoPathExists(repoPath, fromDir)
      ? []
      : [`存在しないパスを参照しています: \`${span}\``];
  }

  return CONFIG_FILE_RE.test(span) && !resolvers.configFileExists(span)
    ? [`リポジトリに存在しない設定ファイルを参照しています: \`${span}\``]
    : [];
}

/** ルートの makefile として読むファイル名か。綴りは処理系依存なので実エントリ名で拾う。 */
export function isRootMakefile(fileName: string): boolean {
  return fileName === "makefile" || fileName === "Makefile" || fileName === "GNUmakefile";
}

/** `.makefiles/` 配下でターゲット宣言を読むファイルか。 */
export function isMakefileFragment(fileName: string): boolean {
  return fileName.endsWith(".mk");
}

/** Claude 側のエージェント定義ファイルか。対訳（`*.ja.md`）は定義ではない。 */
export function isClaudeAgentDefinition(fileName: string): boolean {
  return fileName.endsWith(".md") && !fileName.endsWith(".ja.md");
}

/** Codex 側のエージェント定義ファイルか。 */
export function isCodexAgentDefinition(fileName: string): boolean {
  return fileName.endsWith(".toml");
}

/** 定義ファイル名からエージェント名を取り出す（拡張子は環境ごとに違う）。 */
export function agentName(fileName: string): string {
  return fileName.replace(/\.(md|toml)$/, "");
}

/**
 * 違反一覧を失敗出力へ整形する。
 *
 * @remarks
 * ファイル名・行番号の順に整列してから並べます。走査順は環境ごとの `readdir` に依存するため、
 * そのまま出すと同じ違反集合でも実行ごとに並びが変わり、CI の失敗差分が読めなくなります。
 */
export function formatFindings(findings: readonly Finding[]): string {
  const sorted = [...findings].sort(
    (left, right) => left.file.localeCompare(right.file) || left.line - right.line,
  );
  const lines: string[] = [];
  let current: string | null = null;

  for (const finding of sorted) {
    if (finding.file !== current) {
      if (current !== null) lines.push("");
      lines.push(`  ${finding.file}`);
      current = finding.file;
    }
    lines.push(`    :${finding.line}  [${finding.rule}] ${finding.message}`);
  }

  return lines.join("\n");
}
