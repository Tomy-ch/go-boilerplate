// env / OpenAPI / Copilot 指示書のアプリ名・タイトル行を書き換える規則。
//
// いずれの置換も `String.prototype.replace` に関数を渡す。文字列を渡すと `$&` や `$1` が
// 置換パターンとして解釈され、`$` を含むアプリ名がそのまま入らない。

/** env ディレクトリで APP_NAME を持ちうるファイル名か。 */
export function isEnvFile(name: string): boolean {
  return name.startsWith(".env");
}

// 行頭固定。`MOCK_APP_NAME=` のような別キーの一部に当てない。
const ENV_APP_NAME = /^APP_NAME=.*/m;
// OpenAPI の `info.title`。インデント 2 で固定し、`paths` 配下の深い `title:` を拾わない。
const OPENAPI_TITLE = /^ {2}title: .*/m;
// Copilot 指示書の先頭見出し（h1）。`## ` 以降の下位見出しには当たらない。
const COPILOT_TITLE = /^# .*/m;

/** env ファイルの `APP_NAME=` 行を置き換える。該当行が無ければ `null`。 */
export function replaceEnvAppName(content: string, appName: string): string | null {
  if (!ENV_APP_NAME.test(content)) {
    return null;
  }

  return content.replace(ENV_APP_NAME, () => `APP_NAME=${appName}`);
}

/** OpenAPI の `info.title` 行を置き換える。該当行が無ければ `null`。 */
export function replaceOpenapiTitle(content: string, title: string): string | null {
  if (!OPENAPI_TITLE.test(content)) {
    return null;
  }

  return content.replace(OPENAPI_TITLE, () => `  title: ${title}`);
}

/** Copilot 指示書の先頭見出しを置き換える。該当行が無ければ `null`。 */
export function replaceCopilotTitle(content: string, title: string): string | null {
  if (!COPILOT_TITLE.test(content)) {
    return null;
  }

  return content.replace(COPILOT_TITLE, () => `# ${title}`);
}

/**
 * アプリ名・タイトルを持つ対象。
 *
 * @remarks
 * 「どのファイルがアプリのメタデータを持つか」は置換規則の一部です。入口に定数として置くと、
 * 置換規則を直した人がこの一覧を見落とし、片方だけ古い名前が残ります。
 */
export const APP_METADATA_TARGETS = {
  envDir: "env",
  openapiFile: "openapi/openapi.yaml",
  copilotInstructionsFile: ".github/copilot-instructions.md",
} as const;
