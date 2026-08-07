import type { Finding } from "./lint-report";
import { type WorkflowJob, type WorkflowLine, splitJobs, usesActionPattern } from "./workflow";

/** PR コメントを投稿するローカルアクション。これを使うジョブが検査対象になる。 */
export const COMMENT_ACTION = "./.github/actions/upsert-pr-comment";

/** コメント投稿そのものに必要で、Actions が発行する短命トークン。唯一許可する secret。 */
export const ALLOWED_SECRET = "GITHUB_TOKEN";

const COMMENT_ACTION_USE = usesActionPattern(COMMENT_ACTION, false);
const EXPRESSION = /\$\{\{([\s\S]*?)\}\}/g;
const SECRET_REFERENCE = /\bsecrets\b\s*(?:\.\s*([A-Za-z0-9_-]+)|\[\s*["']([A-Za-z0-9_-]+)["']\s*\])?/g;

/** 検出した secret 参照。`name` が `undefined` なら `secrets` コンテキスト全体の参照。 */
export type SecretReference = {
  number: number;
  name: string | undefined;
};

/** ジョブが PR コメント投稿アクションを使っているか。 */
export function usesCommentAction(job: WorkflowJob): boolean {
  return job.lines.some(({ text }) => COMMENT_ACTION_USE.test(text));
}

/**
 * 1 行から、許可されない secret 参照を取り出す。
 *
 * @remarks
 * `${{ }}` 式の中だけを見ます。散文中の「secrets」に反応させないためです。追えるのは
 * コンテキストの直接参照だけで、別ジョブの `outputs` 経由で渡す間接参照は静的には追えません。
 */
export function secretReferences({ number, text }: WorkflowLine): SecretReference[] {
  const referenced: SecretReference[] = [];

  for (const expression of text.matchAll(EXPRESSION)) {
    for (const reference of expression[1].matchAll(SECRET_REFERENCE)) {
      const name = reference[1] ?? reference[2];
      if (name === ALLOWED_SECRET) continue;
      referenced.push({ number, name });
    }
  }

  return referenced;
}

/** 違反メッセージ内での secret の呼び方。名前を取れない場合はコンテキスト全体として書く。 */
export function describeSecret(name: string | undefined): string {
  return name ? `\`secrets.${name}\`` : "`secrets` コンテキスト全体";
}

/** 1 ワークフロー分の走査結果。`found` が false なら `jobs:` を読めていない（検査対象の取り違え）。 */
export type SecretScan = {
  found: boolean;
  findings: Finding[];
  /** 検査したコメント投稿ジョブの数。0 件のまま終わる実行を「問題なし」と読ませないために数える。 */
  commentingJobs: number;
};

/**
 * ワークフロー 1 本を走査し、渡してはいけない secret 参照を違反として返す。
 *
 * @remarks
 * ジョブ本文とワークフロー全体の `env:` を別々に見ます。前者はそのジョブに閉じますが、後者は
 * コメント投稿ジョブにも届くため、参照している場所がジョブの外でも違反になります。
 */
export function scanWorkflow(file: string, source: string): SecretScan {
  const { jobs, preamble, found } = splitJobs(source);

  if (!found) {
    return { found: false, findings: [], commentingJobs: 0 };
  }

  const commenting = jobs.filter(usesCommentAction);
  const findings: Finding[] = [];

  for (const job of commenting) {
    for (const line of job.lines) {
      for (const { number, name } of secretReferences(line)) {
        findings.push({
          file,
          line: number,
          message: `ジョブ \`${job.id}\` は ${COMMENT_ACTION} を使うため ${describeSecret(name)} を渡せません（マスキングは tee したファイルに効かず、生値が PR コメントに載ります）`,
        });
      }
    }
  }

  if (commenting.length > 0) {
    for (const line of preamble) {
      for (const { number, name } of secretReferences(line)) {
        findings.push({
          file,
          line: number,
          message: `ワークフロー全体に及ぶ ${describeSecret(name)} は ${COMMENT_ACTION} を使うジョブにも届きます（マスキングは tee したファイルに効かず、生値が PR コメントに載ります）`,
        });
      }
    }
  }

  return { found: true, findings, commentingJobs: commenting.length };
}
