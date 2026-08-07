/** mock-auth-server が起動時に読む固定ユーザー 1 件。 */
export type MockAuthUser = {
  subject: string;
  email: string;
  given_name: string;
  family_name: string;
  name: string;
  status: string;
};

/**
 * 中立な既定ユーザー。
 *
 * @remarks
 * サンプル API を撤去した利用者に残るのはこの内容です。デモ由来の人名・ドメインを含めないのは、
 * 撤去したはずのサンプルが認証の固定データとして残り続けるのを避けるためです。ファイル自体は
 * 削除せず上書きするので、撤去後も mock は起動できます。
 */
export const DEFAULT_USERS: readonly MockAuthUser[] = [
  {
    subject: "user-example",
    email: "user@example.com",
    given_name: "Example",
    family_name: "User",
    name: "Example User",
    status: "active",
  },
];

/** fixture ファイルへ書き込む JSON 本文（末尾改行付き）を組み立てる。 */
export function renderFixture(users: readonly MockAuthUser[]): string {
  return `${JSON.stringify(users, null, 2)}\n`;
}
