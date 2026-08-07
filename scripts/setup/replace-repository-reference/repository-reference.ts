// README / OpenAPI に散在する GitHub リポジトリ参照を `<owner>/<repo>` へ寄せる規則。
// いずれも関数で置換する（`$&` を含むリポジトリ名を置換パターンとして解釈させない）。

// 見出しと `cd` はいずれも行頭固定・最初の 1 件のみ。README 中の別の h1 相当や
// 手順中の 2 つ目の `cd` を巻き込まないため。
const README_HEADING = /^# .*/m;
const README_CD = /^cd .*/m;
// バッジ URL は同じものが複数箇所に出るため全置換する。`)` と空白で URL の終わりを取る。
const BADGE_GO_VERSION = /https:\/\/img\.shields\.io\/github\/go-mod\/go-version\/[^\s)]+/g;
const BADGE_LICENSE = /https:\/\/img\.shields\.io\/github\/license\/[^\s)]+/g;
// clone 手順の URL。`.git` で終わるものだけを対象にし、通常のリポジトリ URL は触らない。
const CLONE_URL = /https:\/\/github\.com\/[^/\s>]+\/[^/\s>]+\.git/g;

const OPENAPI_TERMS_OF_SERVICE = /^ {2}termsOfService: https:\/\/github\.com\/.*$/m;

/** README の見出し・バッジ・clone URL・`cd` 行のリポジトリ参照を置き換える。 */
export function replaceReadmeReferences(content: string, repository: string): string {
  const repoName = repository.split("/")[1];

  return content
    .replace(README_HEADING, () => `# ${repoName}`)
    .replace(
      BADGE_GO_VERSION,
      () => `https://img.shields.io/github/go-mod/go-version/${repository}`,
    )
    .replace(BADGE_LICENSE, () => `https://img.shields.io/github/license/${repository}`)
    .replace(CLONE_URL, () => `https://github.com/${repository}.git`)
    .replace(README_CD, () => `cd ${repoName}`);
}

/** OpenAPI の `info.termsOfService` を置き換える。 */
export function replaceOpenapiTermsOfService(content: string, repository: string): string {
  return content.replace(
    OPENAPI_TERMS_OF_SERVICE,
    () => `  termsOfService: https://github.com/${repository}`,
  );
}

/**
 * GitHub リポジトリ参照を持つ対象。
 *
 * @remarks
 * README は英日の対、OpenAPI は `termsOfService`。どれか 1 つを置換対象から落とすと、
 * 生成したリポジトリにボイラープレート側の URL が残ります。
 */
export const REPOSITORY_REFERENCE_TARGETS = {
  readmeFiles: ["README.md", "README.ja.md"] as readonly string[],
  openapiFile: "openapi/openapi.yaml",
} as const;
