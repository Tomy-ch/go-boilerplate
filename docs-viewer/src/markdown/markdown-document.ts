import { marked } from "marked";

import { SanitizedDocument } from "../sanitize/sanitized-document";

/**
 * Markdown を表示できる形へ変換します。
 *
 * marked の出力を検査せずに描画しません。生成物として配信される Markdown は、リポジトリの
 * ドキュメントから機械的に組まれたものですが、その前提が崩れたときに描画側が最後の防波堤に
 * なるよう、常に sanitize を通します。
 */
export function parseMarkdownDocument(markdown: string): SanitizedDocument {
  return SanitizedDocument.from(marked.parse(markdown, { async: false }));
}
