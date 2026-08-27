// 初期化（`docs/get-started/setup-repository.md` の Phase 5）が過不足なく終わったかを判定する規則。
// 入口はファイル読み込みと終了コードだけを担う。
//
// このモジュールは検証成功後、verify-setup 自身と一緒に消える（理由は同 Phase 5）。

/** ボイラープレート由来の名残。置換し損ねると利用者のリポジトリに残る。 */
export const BOILERPLATE_MODULE = "go-boilerplate";

/** 初期化で置換されるべき値。利用者が指定したものを、検証側でもそのまま突き合わせる。 */
export type ExpectedIdentity = {
  module: string;
  repository: string;
  copyrightHolder: string;
  copyrightYear: string;
  codeOwners: string;
};

/** 検証の入力。読み取りは入口が行い、ここは内容だけを見る。 */
export type VerificationInput = {
  expected: ExpectedIdentity;
  goMod: string;
  license: string;
  codeowners: string;
  readme: string;
  openapi: string;
  /**
   * `go-boilerplate` が残っている「置換対象ファイル」の一覧。空であるべき。
   *
   * @remarks
   * 何が置換対象かは `replace-module` の `isReplacementTarget` が決めます。生成物は置換後に
   * 再生成される前提で対象外なので、そこに名前が残っていても未完了ではありません。検証側が
   * 別の除外規則を持つと、置換器が対象を変えたときに黙ってずれます。
   */
  boilerplateReferences: readonly string[];
};

/** モジュール名が置換され、ボイラープレート名が残っていないか。 */
export function findModuleFailures(goMod: string, expected: string): string[] {
  const failures: string[] = [];

  if (!new RegExp(`^module ${escapeForRegExp(expected)}$`, "m").test(goMod)) {
    failures.push(`go.mod の module が ${expected} になっていません`);
  }
  if (expected !== BOILERPLATE_MODULE && goMod.includes(BOILERPLATE_MODULE)) {
    failures.push(`go.mod に ${BOILERPLATE_MODULE} が残っています`);
  }

  return failures;
}

/**
 * 置換対象のファイルにボイラープレート名が残っていないか。
 *
 * @remarks
 * 再生成前は生成物に名前が残るため `go build` は通りません。「置換器が対象と宣言した範囲を
 * 取りこぼしていないか」の方が、この時点で確かめられる本来の契約です。コメントや文字列に
 * 残った名残もここで捕まります。
 */
export function findLeftoverReferences(references: readonly string[]): string[] {
  return references.length === 0
    ? []
    : [`${BOILERPLATE_MODULE} を参照している箇所が残っています: ${references.join(", ")}`];
}

/** LICENSE の著作権表示が指定の権利者・年になっているか。 */
export function findLicenseFailures(
  license: string,
  holder: string,
  year: string,
): string[] {
  const expected = `Copyright (c) ${year} ${holder}`;

  return license.includes(expected) ? [] : [`LICENSE に "${expected}" がありません`];
}

/**
 * CODEOWNERS の全ルールが指定の所有者になっているか。
 *
 * @remarks
 * コメント行は例示の所有者を残すため対象外です（`replace-codeowners` と同じ扱い）。
 */
export function findCodeownersFailures(codeowners: string, owners: string): string[] {
  const rules = codeowners
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "" && !line.startsWith("#"));

  if (rules.length === 0) {
    return ["CODEOWNERS にルールが 1 件もありません（検査が的を外しています）"];
  }

  const stale = rules.filter((line) => !line.includes(owners));

  return stale.length === 0 ? [] : [`CODEOWNERS に所有者が ${owners} でないルールがあります: ${stale.join(" / ")}`];
}

/** README と OpenAPI がリポジトリ参照を置換済みか。 */
export function findRepositoryFailures(
  readme: string,
  openapi: string,
  repository: string,
): string[] {
  const failures: string[] = [];

  if (!readme.includes(repository)) {
    failures.push(`README に ${repository} への参照がありません`);
  }
  if (!openapi.includes(repository)) {
    failures.push(`openapi.yaml に ${repository} への参照がありません`);
  }

  return failures;
}

/** 初期化が完了しているかを判定し、満たしていない条件を全て返す。 */
export function collectFailures(input: VerificationInput): string[] {
  return [
    ...findModuleFailures(input.goMod, input.expected.module),
    ...findLeftoverReferences(input.boilerplateReferences),
    ...findLicenseFailures(input.license, input.expected.copyrightHolder, input.expected.copyrightYear),
    ...findCodeownersFailures(input.codeowners, input.expected.codeOwners),
    ...findRepositoryFailures(input.readme, input.openapi, input.expected.repository),
  ];
}

/** 初期化ツールの在否で行を切り替えるマーカー名。 */
export const LOCALIZATION_MARKER = "setup-localize";

/**
 * 初期化ツールの撤去に合わせてマーカー行を落とすファイル。
 *
 * @remarks
 * ディレクトリを消すだけでは、消えたスクリプトを呼ぶ make ターゲットとその説明が残ります。
 * `make help` に並び続け、叩けば失敗するだけのものになるので、宣言側も同時に落とします。
 * 生成物（`docs/portal/guides/**`）は再生成で追随するため挙げません。
 */
export const LOCALIZATION_MARKER_FILES: readonly string[] = [
  ".makefiles/github/operation/setup-repository.mk",
  ".makefiles/README.md",
  ".makefiles/README.ja.md", // doc-pair:line
  "scripts/README.md",
  "scripts/README.ja.md", // doc-pair:line
];

/** `setup/` 配下の共有モジュール。使う側が全て消えたときだけ道連れにする。 */
export const SETUP_SHARED_DIR = "lib";

/** サンプル削除ツール。まだ残っていれば `setup/lib` を使い続けている。 */
export const SAMPLE_REMOVER_DIR = "remove-sample-api";

/**
 * 初期化ツールを実際に起動する CI（リポジトリルートからの相対）。
 *
 * @remarks
 * `setup-scripts-check.yaml` は `replace-*` と `verify-setup` を叩いて初期化が主張どおりに
 * 動くことを検証します。ツールが自消滅した後も残ると、消えた入口を叩き続けて作成先で恒久的に
 * 赤くなります。`remove-boilerplate-identity` 側も同じパスを消します。あちらも同じ CI に
 * 依存しており、2 つの撤去はどちらが先に走るか決まっていないためです。先に走ったほうが持って
 * 行き、後から走ったほうの削除は空振りします。
 */
export const LOCALIZATION_CI_PATHS: readonly string[] = [
  ".github/workflows/setup-scripts-check.yaml",
];

/** 初期化ツール（`replace-*`）のディレクトリ名。 */
export const LOCALIZATION_TOOL_DIRS: readonly string[] = [
  "replace-module",
  "replace-app-metadata",
  "replace-license-copyright",
  "replace-repository-reference",
  "replace-codeowners",
];

/**
 * 検証成功後に消す対象（`setup/` からの相対）を返す。
 *
 * @remarks
 * `setup/lib` は共有なので、使う側が全て消えたときにだけ道連れにします。判定は「今 `setup/`
 * に在るディレクトリのうち、この撤去で消えないものが残るか」で行い、利用者の名前は列挙しません。
 * 名前を挙げる形だと、`lib` を使うツールを足すたびにこちらへも足す必要があり、漏らしても
 * 静かに壊れます——実際 `lib` の利用者は 10 ディレクトリありながら、宣言は 1 つだけを見ていました。
 *
 * ディレクトリ単位で挙げるのは、ファイルを列挙すると判定モジュールを足したときに漏れ、
 * 消えたはずのツールの一部だけが利用者のリポジトリへ居座るためです。
 */
export function selfDestructTargets(selfDir: string, setupEntries: readonly string[]): string[] {
  const targets = [...LOCALIZATION_TOOL_DIRS, selfDir];
  const survives = setupEntries.some(
    (entry) => entry !== SETUP_SHARED_DIR && !targets.includes(entry),
  );

  return survives ? targets : [...targets, SETUP_SHARED_DIR];
}

function escapeForRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
