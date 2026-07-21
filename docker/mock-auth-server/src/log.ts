// log.ts は認可フローの構造化イベントログを提供する。
// token / authorization code / code_verifier / password / 秘密鍵の全文は決して出力しない
// （呼び出し側が渡すフィールドを限定する。当ヘルパは秘匿対象を受け取らない前提）。
export function logEvent(event: string, fields: Record<string, unknown> = {}): void {
  console.log(JSON.stringify({ event, ...fields }));
}
