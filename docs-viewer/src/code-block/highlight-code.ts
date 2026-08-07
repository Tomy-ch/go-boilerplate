/**
 * highlight.js を遅延読み込みして、コードを強調表示済みの HTML へ変換します。
 *
 * @remarks
 * highlight.js は初期表示に要りません。文書を開いて初めて必要になるため、動的 import で
 * 分割し、ポータルの一覧表示までの読み込みから外します。
 *
 * 言語表記を持たないフェンスと、highlight.js が知らない言語は変換しません。言語を推測させると
 * 設定ファイルやログの断片が別の言語として色付けされ、書かれていない構造を読み手へ見せます。
 *
 * @param code - フェンスの中身
 * @param language - フェンスの言語表記。無い場合は `null`
 * @returns 強調表示済みの HTML。対象外の言語では `null`
 */
export async function highlightCode(code: string, language: string | null): Promise<string | null> {
  if (!language) {
    return null;
  }

  const { default: hljs } = await import("highlight.js/lib/common");

  if (!hljs.getLanguage(language)) {
    return null;
  }

  return hljs.highlight(code, { language }).value;
}
