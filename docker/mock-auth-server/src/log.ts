// log.ts は認可フローの構造化イベントログを提供する。
// token / authorization code / code_verifier / password / 秘密鍵の全文を fields に含めてはならない（§29）。
export function logEvent(event: string, fields: Record<string, unknown> = {}): void {
  console.log(JSON.stringify({ event, ...fields }));
}
