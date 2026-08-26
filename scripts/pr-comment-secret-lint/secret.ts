import type { Finding } from "../lib/lint-report";
import { type WorkflowJob, type WorkflowLine, splitJobs, usesActionPattern } from "../lib/workflow";

/** PR コメントを投稿するローカルアクション。これを使うジョブが検査対象になる。 */
export const COMMENT_ACTION = "./.github/actions/upsert-pr-comment";

/** コメント投稿そのものに必要で、Actions が発行する短命トークン。唯一許可する secret。 */
export const ALLOWED_SECRET = "GITHUB_TOKEN";

const COMMENT_ACTION_USE = usesActionPattern(COMMENT_ACTION, false);
const EXPRESSION = /\$\{\{([\s\S]*?)\}\}/g;
/** `secrets` コンテキストへの参照。名前の有無はこの後ろを見て決める。 */
const SECRET_CONTEXT = /\bsecrets\b/g;

/**
 * `secrets` の直後に続く、参照する名前の書き方。GitHub は属性形と添字形の 2 通りを受け付ける。
 *
 * 名前を後ろから別に読むのは、1 本の正規表現へ詰めるとどの枝が当たったかを捕獲組の番号で
 * 見分けることになり、書き方が増えるたびに条件が枝分かれするためである。どちらにも当たらない
 * 参照はコンテキスト全体の参照で、名前を持たない。
 */
const SECRET_NAME_FORMS = [
  /^\s*\.\s*([A-Za-z0-9_-]+)/,
  /^\s*\[\s*["']([A-Za-z0-9_-]+)["']\s*\]/,
] as const;

/** `secrets` の直後の文字列から、参照している secret の名前を読む。 */
function secretName(rest: string): string | undefined {
  for (const form of SECRET_NAME_FORMS) {
    const matched = form.exec(rest);
    if (matched) return matched[1];
  }

  return undefined;
}

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
    for (const reference of expression[1].matchAll(SECRET_CONTEXT)) {
      const name = secretName(expression[1].slice(reference.index + reference[0].length));
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
