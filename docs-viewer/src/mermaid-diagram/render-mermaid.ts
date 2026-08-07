/** 読み込みは 1 回に畳む。図が増えても mermaid 本体を取り直さない。 */
let mermaidImport: Promise<typeof import("mermaid").default> | null = null;

function loadMermaid() {
  if (!mermaidImport) {
    mermaidImport = import("mermaid").then(({ default: mermaid }) => mermaid);
  }

  return mermaidImport;
}

/**
 * 図の配色を、面の配色と同じ 2 経路から決めます。
 *
 * @remarks
 * mermaid は自前の配色を持ち、既定では明るい面を前提にした色で描きます。暗い面にそのまま
 * 置くと図だけが白く浮くため、tokens.css が semantic token を切り替えるのと同じ条件
 * （OS の設定と `data-theme`）で図側の配色も選びます。
 *
 * @param prefersDark - OS の設定が暗い配色を求めているか
 * @param dataTheme - `data-theme` の宣言値。宣言が無ければ `null`
 */
export function resolveMermaidTheme(prefersDark: boolean, dataTheme: string | null): "dark" | "default" {
  if (dataTheme === "dark") {
    return "dark";
  }

  if (dataTheme === "light") {
    return "default";
  }

  return prefersDark ? "dark" : "default";
}

function currentTheme(): "dark" | "default" {
  return resolveMermaidTheme(
    window.matchMedia("(prefers-color-scheme: dark)").matches,
    document.documentElement.dataset.theme ?? null,
  );
}

/**
 * mermaid を遅延読み込みして、図の定義を SVG へ変換します。
 *
 * @remarks
 * mermaid は初期表示に要りません。図を含む文書を開いて初めて必要になるため、動的 import で
 * 分割し、ポータルの一覧表示までの読み込みから外します。
 *
 * 設定は図ごとに与えます。読み込み時に 1 度だけ与えると、配色を切り替えた後に開いた図が
 * 切り替え前の配色のまま描かれます。
 *
 * @param id - 生成する SVG の識別子。同じ面に複数の図が並ぶため、呼び出し側が一意な値を渡す
 * @param code - mermaid の図の定義
 * @returns 描画結果の SVG
 */
export async function renderMermaid(id: string, code: string): Promise<string> {
  const mermaid = await loadMermaid();

  mermaid.initialize({
    // 図は文書を開いたときにこちらから描画する。読み込み時に DOM を走査させない。
    startOnLoad: false,
    // 図の定義に書かれた HTML と script を実行しない。描画対象はリポジトリのドキュメントだが、
    // 本文の sanitize と同じく、前提が崩れたときに描画側が最後の防波堤になる状態を保つ。
    securityLevel: "strict",
    theme: currentTheme(),
  });

  const { svg } = await mermaid.render(id, code);

  return svg;
}
