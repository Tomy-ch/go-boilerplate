// graphify の `affected`（変更影響の逆引き）をシンボル名で叩けるようにするラッパー。
//
// `graphify affected` は一意なノード id しか受け付けず、`NormalizeError()` のような
// シンボル名では "No unique node match" で止まる。id は graph.json の中にしか無いため、
// 素の CLI では利用者が 14MB の JSON を自力で漁ることになる。ここで名前→id を解決し、
// 曖昧なときは候補を出して選ばせる。
//
// 使い方:
//   node .claude/scripts/graph-affected.ts <symbol> [--depth N] [--graph PATH] [-- <graphify の追加引数>]
//
// 前提: `graphify update .` 済み（graphify-out/graph.json が存在する）。

import { spawnSync } from "node:child_process";
import fs from "node:fs";

const DEFAULT_GRAPH = "graphify-out/graph.json";
const USAGE =
  "使い方: node .claude/scripts/graph-affected.ts <symbol> [--depth N] [--graph PATH] [-- <graphify の追加引数>]";
/** 曖昧なときに並べる候補の上限。全件出すと端末が流れて選べなくなる。 */
const MAX_LISTED_CANDIDATES = 20;

/** 終了コード。2 は使い方・前提の誤り、1 は解決できなかった場合。 */
const EXIT_USAGE = 2;
const EXIT_UNRESOLVED = 1;

/** graph.json のノードのうち、この用途で読む面だけ。 */
type GraphNode = {
  id: string;
  label?: string;
  norm_label?: string;
  source_file?: string;
  source_location?: string;
};

type CliOptions = {
  depth?: string;
  graph?: string;
};

type ParsedArgs = {
  symbol?: string;
  options: CliOptions;
  /** `--` 以降。graphify へそのまま渡す。 */
  passThrough: string[];
};

/**
 * 引数を解釈する。`--` 以降は解釈せず graphify への素通しにする。
 *
 * @remarks
 * 素通しを先に切り出すのは、graphify 側のオプションをここで知らずに済ませるため。
 * 知っていると graphify が増やすたびにこのラッパーも直すことになる。
 */
function parseArgs(argv: readonly string[]): ParsedArgs {
  const separator = argv.indexOf("--");
  const passThrough = separator === -1 ? [] : [...argv.slice(separator + 1)];
  const args = separator === -1 ? argv : argv.slice(0, separator);

  const options: CliOptions = {};
  const positional: string[] = [];

  for (let i = 0; i < args.length; i += 1) {
    if (args[i] === "--depth" || args[i] === "--graph") {
      const key = args[i] === "--depth" ? "depth" : "graph";
      options[key] = args[i + 1];
      i += 1;
      continue;
    }
    positional.push(args[i]);
  }

  return { symbol: positional[0], options, passThrough };
}

/** 末尾の `()` を落とす。`NormalizeError()` と `NormalizeError` を同じ指定として扱う。 */
function stripCallSuffix(value: string | undefined): string {
  return (value ?? "").replace(/\(\)$/, "");
}

function fold(value: string | undefined): string {
  return stripCallSuffix(value).toLowerCase();
}

/**
 * シンボル名に一致するノードを絞り込む。
 *
 * @remarks
 * 3 段で降りていき、最初に当たった段の結果を返します。Go はシンボルをケースで区別する
 * （`NormalizeError` と `normalizeError` は別物）ため、ケースを保った完全一致を最優先します。
 * 段を降りるのは前の段が 0 件のときだけなので、厳密な一致がある限り緩い一致に汚されません。
 */
function findCandidates(nodes: readonly GraphNode[], symbol: string): GraphNode[] {
  const wanted = stripCallSuffix(symbol);
  const passes: Array<(node: GraphNode) => boolean> = [
    (node) => stripCallSuffix(node.label) === wanted,
    (node) => fold(node.label) === fold(wanted) || fold(node.norm_label) === fold(wanted),
    (node) => fold(node.label).includes(fold(wanted)),
  ];

  for (const matches of passes) {
    const candidates = nodes.filter(matches);

    if (candidates.length > 0) {
      return candidates;
    }
  }

  return [];
}

/** ノードを「id (label @ file:location)」の 1 行で表す。 */
function describeNode(node: GraphNode): string {
  return `${node.id}  (${node.label} @ ${node.source_file}:${node.source_location})`;
}

/** graphify へ渡す `mise exec` の引数列を組み立てる。 */
function buildGraphifyArgs(
  nodeId: string,
  options: CliOptions,
  passThrough: readonly string[],
): string[] {
  const args = ["exec", "pipx:graphifyy[sql]", "--", "graphify", "affected", nodeId];

  if (options.depth !== undefined) {
    args.push("--depth", options.depth);
  }
  if (options.graph !== undefined) {
    args.push("--graph", options.graph);
  }

  return [...args, ...passThrough];
}

function readNodes(graphPath: string): GraphNode[] {
  const graph = JSON.parse(fs.readFileSync(graphPath, "utf8")) as { nodes?: GraphNode[] };

  return graph.nodes ?? [];
}

function reportMissingGraph(graphPath: string): void {
  console.error(`✘ グラフがありません: ${graphPath}`);
  console.error('    対処: `mise exec "pipx:graphifyy[sql]" -- graphify update . --no-cluster` で生成する。');
}

function reportNoMatch(symbol: string, nodeCount: number): void {
  console.error(`✘ '${symbol}' に一致するノードがありません（${nodeCount} ノードを検索）`);
  console.error("    グラフが古い可能性があります。`graphify update .` を実行してから再試行してください。");
}

function reportAmbiguous(symbol: string, candidates: readonly GraphNode[]): void {
  console.error(`✘ '${symbol}' が ${candidates.length} 件に一致します。id を指定して再実行してください:`);

  for (const node of candidates.slice(0, MAX_LISTED_CANDIDATES)) {
    console.error(`    ${describeNode(node)}`);
  }

  if (candidates.length > MAX_LISTED_CANDIDATES) {
    console.error(`    ... 他 ${candidates.length - MAX_LISTED_CANDIDATES} 件`);
  }
}

function main(argv: readonly string[]): number {
  const { symbol, options, passThrough } = parseArgs(argv);

  if (symbol === undefined || symbol === "") {
    console.error(USAGE);

    return EXIT_USAGE;
  }

  const graphPath = options.graph ?? DEFAULT_GRAPH;

  if (!fs.existsSync(graphPath)) {
    reportMissingGraph(graphPath);

    return EXIT_USAGE;
  }

  const nodes = readNodes(graphPath);
  const candidates = findCandidates(nodes, symbol);

  if (candidates.length === 0) {
    reportNoMatch(symbol, nodes.length);

    return EXIT_UNRESOLVED;
  }

  if (candidates.length > 1) {
    reportAmbiguous(symbol, candidates);

    return EXIT_UNRESOLVED;
  }

  const target = candidates[0];
  // 解決結果は stderr へ出す。stdout は graphify の出力だけにして、パイプで受けられるようにする。
  console.error(`→ ${describeNode(target)}`);

  const run = spawnSync("mise", buildGraphifyArgs(target.id, options, passThrough), {
    stdio: "inherit",
  });

  return run.status ?? EXIT_UNRESOLVED;
}

process.exit(main(process.argv.slice(2)));
