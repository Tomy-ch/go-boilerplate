/**
 * frontend generator（orval）が bundle 済み OpenAPI から component 型を生成できたかの判定。
 *
 * 判定だけを持ち、orval の起動とファイル入出力は index.ts が担う。
 */

/** 生成物に現れなかった期待型。 */
export type Finding = {
  readonly expected: string;
  readonly reason: string;
};

/** 生成物が空（ファイル無し / 中身無し）で判定できない。 */
export class DegenerateOutputError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "DegenerateOutputError";
  }
}

/**
 * 生成された TypeScript の `export interface <Name>` / `export type <Name>` 宣言名を集める。
 *
 * 宣言の形は orval の出力に合わせた最小限で、`export interface DeliveryEvent {` と
 * `export type StreamCursor = string;` の両方を 1 つの名前として数える。
 */
export function declaredTypeNames(sources: readonly string[]): Set<string> {
  const names = new Set<string>();
  const declaration = /^export (?:interface|type) ([A-Za-z_$][\w$]*)\b/gm;
  for (const source of sources) {
    for (const match of source.matchAll(declaration)) {
      names.add(match[1]);
    }
  }
  return names;
}

/**
 * 期待する component 型のうち、生成物に宣言が無いものを返す。
 *
 * 生成物が 1 つも無い、または宣言を 1 つも含まない場合は判定せず DegenerateOutputError を投げる。
 * 「型が無い」を「生成物が壊れている」と同じ結果にすると、生成が空振りした run が緑になるため。
 */
export function missingTypes(sources: readonly string[], expected: readonly string[]): Finding[] {
  if (sources.length === 0) {
    throw new DegenerateOutputError("生成物が 1 つもありません");
  }
  const declared = declaredTypeNames(sources);
  if (declared.size === 0) {
    throw new DegenerateOutputError("生成物に型の宣言が 1 つもありません");
  }
  return expected
    .filter((name) => !declared.has(name))
    .map((name) => ({ expected: name, reason: "生成物に宣言がありません" }));
}
