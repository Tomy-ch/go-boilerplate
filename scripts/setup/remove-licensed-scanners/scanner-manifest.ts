// 資格情報 / ライセンス費用を要するスキャナ 2 件の撤去対象を宣言する manifest（データ定義）。
// 削除ロジックは scanner-removal.ts、git 操作は git-commit.ts を参照。
//
// 対象は 2 分類の和集合である。ベンダーのトークンを要する SonarQube Cloud と、
// private リポジトリで課金されるもの（CodeQL）。Bearer は Elastic License 2.0 で CI 実行その
// ものは無償無制限のため、どちらにも当たらず対象外である。
//
// README の編集を完全一致で宣言するのは、README が動いたときに scanner-removal.ts が投げて
// 「消えたつもりで消えていない」を防ぐため。宣言の重さはその引き換えである。

/** 行内から取り除く完全一致文字列。他ツールと同居する表セルや散文に使う。 */
export type DocFragment = {
  file: string;
  fragment: string;
};

/** 行ごと / 段落ごと消す完全一致文字列（前後の空白を含む）。 */
export type DocBlock = {
  file: string;
  block: string;
};

/** 見出しとその本文をまとめて消す指定。 */
export type DocSection = {
  file: string;
  heading: string;
};

/** 1 製品分の撤去宣言。パスはリポジトリルートからの相対。 */
export type ScannerDomain = {
  key: string;
  label: string;
  /** commitlint の type-enum に適合する prefix + 日本語の subject。 */
  commitSubject: string;
  /** これが無ければ撤去済みと見なし、その製品には手を付けない。 */
  presenceMarker: string;
  paths: readonly string[];
  /** 撤去後に参照が 0 件になったときだけ消す lockfile キー。 */
  pinKeys: readonly string[];
  egressJobs: readonly string[];
  docBlocks: readonly DocBlock[];
  docFragments: readonly DocFragment[];
  docSections: readonly DocSection[];
};

const README_EN = ".github/workflows/README.md";
const README_JA = ".github/workflows/README.ja.md";
const SETUP_MK = ".makefiles/github/operation/setup-repository.mk";
const MISE_TOML = "mise.toml";

export const SCANNER_DOMAINS: readonly ScannerDomain[] = [
  {
    key: "sonarqube",
    label: "SonarQube Cloud",
    commitSubject: "CI: SonarQube Cloud のワークフローを撤去する",
    presenceMarker: ".github/workflows/sonarqube.yaml",
    paths: [".github/workflows/sonarqube.yaml", "sonar-project.properties"],
    pinKeys: ["SonarSource/sonarqube-scan-action@v8.2.1", "actions/download-artifact@v7"],
    egressJobs: [
      'sonarqube.yaml:preflight',
      'sonarqube.yaml:report',
      'sonarqube.yaml:sonarqube',
      'sonarqube.yaml:unconfigured-notice',
    ],
    docBlocks: [
      {
        file: README_EN,
        block:
          "|SonarQube Cloud Scan|`sonarqube.yaml`|SonarQube Cloud analysis of first-party source, read back over the Web API and converted to SARIF (**gates on Sonar's quality gate**, issue list report-only; needs `SONAR_TOKEN`, see [Removing the credential-bearing scanners](#removing-the-credential-bearing-scanners))|\n",
      },
      {
        file: README_EN,
        block:
          "| SonarQube Cloud | Go / TypeScript / `sonar-project.properties`-change PRs | same as above | weekly |\n",
      },
      {
        file: README_EN,
        block:
          "| `sonarqube.yaml` `sonarqube` | 15 | vendor-side analysis can queue for up to 10 minutes; test and coverage gates run in their owning workflows |\n",
      },
      {
        file: README_JA,
        block:
          "|SonarQube Cloud Scan|`sonarqube.yaml`|SonarQube Cloud による一次ソースの解析。結果は Web API から読み戻して SARIF へ変換する（**Sonar の品質ゲートでブロックする**。issue の一覧は報告専用。`SONAR_TOKEN` が必要。[資格情報を要するスキャナの撤去](#資格情報を要するスキャナの撤去)を参照）|\n",
      },
      {
        file: README_JA,
        block:
          "| SonarQube Cloud | Go / TypeScript / `sonar-project.properties` 変更 PR | 同上 | 週次 |\n",
      },
      {
        file: README_JA,
        block:
          "| `sonarqube.yaml` `sonarqube` | 15 | ベンダー側の解析キューが最大 10 分待つため。テストとカバレッジのゲートはそれぞれの所有ワークフローで実行する |\n",
      },
    ],
    docFragments: [
      { file: README_EN, fragment: " + `sonarqube.yaml` (SonarQube Cloud) **(gate, quality gate)**" },
      { file: README_EN, fragment: " + `sonarqube.yaml` (SonarQube Cloud) **(gate, quality gate)**" },
      { file: README_EN, fragment: ", `05:00` SonarQube Cloud" },
      { file: README_JA, fragment: " + `sonarqube.yaml`（SonarQube Cloud） **(gate, 品質ゲート)**" },
      { file: README_JA, fragment: " + `sonarqube.yaml`（SonarQube Cloud） **(gate, 品質ゲート)**" },
      { file: README_JA, fragment: "、`05:00` SonarQube Cloud" },
    ],
    docSections: [],
  },
  {
    // 2 件をまとめて説明する散文はこのドメインが持つため、最後に置く。
    key: "code-ql",
    label: "CodeQL",
    commitSubject: "CI: CodeQL のワークフローを撤去する",
    presenceMarker: ".github/workflows/code-ql.yaml",
    // スクリプト自身とその検証ワークフローも最後の製品と一緒に落とす。撤去済みのリポジトリに
    // 残しても、検証は「撤去するものが無い」で赤くなるだけで意味を持たない。実行中の削除だが、
    // Node は既にモジュールを読み終えているので残りの手順は続く。
    paths: [
      ".github/workflows/code-ql.yaml",
      ".github/codeql",
      ".github/workflows/licensed-scanners-removal-check.yaml",
      "scripts/setup/remove-licensed-scanners",
    ],
    // 他の workflow も upload-sarif に使うので、参照数の判定に委ねる（残っていれば消えない）。
    pinKeys: ["github/codeql-action@v4.37.6"],
    egressJobs: [
      'code-ql.yaml:codeql',
      'licensed-scanners-removal-check.yaml:licensed-scanners-removal-check',
    ],
    docBlocks: [
      {
        file: README_EN,
        block:
          "| CodeQL | Go / TypeScript / Actions-definition-change PRs | same as above | weekly |\n",
      },
      {
        file: README_EN,
        block:
          "| `code-ql.yaml` `codeql` | 30 | the limit covers whichever matrix leg is slowest, and no leg but `go` has a completed run to measure; `security-extended` is also a larger suite than the one the previous value was measured against |\n",
      },
      {
        file: README_EN,
        block:
          "\nSonarQube Cloud takes the last slot. Its analysis runs on a vendor's servers, and it is placed at the end for the same reason DAST is placed behind the file-reading scanners: its duration depends on a queue this repository does not control, so nothing useful is gained by having it queued ahead of a scanner that finishes on its own runner.\n",
      },
      {
        file: README_EN,
        block:
          "\nThe `First-party Go source` and `First-party TypeScript source` rows carry the vendor-hosted scanner as well. Sonar is the one deliberate departure from \"one owner per rule\" in this table. Its quality gate judges static analysis and duplication alongside its own issue taxonomy, while the Go and TypeScript test workflows own coverage thresholds. A finding both engines recognize can still turn a pull request red twice; that is accepted because discarding the vendor's verdict entirely would leave the scan reporting into a run that merged regardless.\n",
      },
      {
        file: README_JA,
        block:
          "| CodeQL | Go / TypeScript / Actions 定義の変更 PR | 同上 | 週次 |\n",
      },
      {
        file: README_JA,
        block:
          "\n最後のスロットは SonarQube Cloud です。解析がベンダーのサーバ側で走るため、DAST を全ファイル読み取り系の後ろへ置いたのと同じ理由で最後に並べています。所要時間がこのリポジトリの制御外のキューに左右されるため、自前のランナーで完結するスキャナより前に積む利点がありません。\n",
      },
      {
        file: README_JA,
        block:
          "\n`自前の Go ソース` と `自前の TypeScript ソース` の行にはベンダーホスト型のスキャナも乗っています。Sonar はこの表で唯一「ルール単位で担当 1 つ」から意図的に外れています。品質ゲートは静的解析・重複と Sonar 自身の issue 分類をまとめて判定し、カバレッジの閾値は Go / TypeScript のテストワークフローがそれぞれ担います。両者が認識する検出で PR が 2 回赤くなり得ますが、それを受け入れているのは、ベンダーの判定を捨てると「スキャンは報告するが run はそのままマージされる」状態になるためです。\n",
      },
      // 撤去後は呼ぶ先が無くなるので、ターゲットの宣言とレシピも一緒に落とす。
      {
        file: SETUP_MK,
        block:
          ".PHONY: setup-remove-licensed-scanners ## 資格情報/課金を要するスキャナ2件を撤去し製品ごとにコミット\n",
      },
      {
        file: SETUP_MK,
        block: `
# 資格情報 / ライセンス費用を要するスキャナ 2 件の撤去。
# 他の setup ターゲットと違いツールランナーを経由せずホストで走らせる。このスクリプトは
# 製品ごとに git commit を積むため、setup-repo と同じくホストの git を使う必要がある
# （worktree では .git がマウント外の実体を指すファイルなので、コンテナ内からは辿れない）。
# プレビューは DRY_RUN=1 を付ける（書き込みもコミットも行わない）。
setup-remove-licensed-scanners:
\t@$(TSX) scripts/setup/remove-licensed-scanners $(SETUP_DRY_RUN_FLAG)
`,
      },
    ],
    docFragments: [
      { file: README_EN, fragment: ", `01:15` CodeQL" },
      { file: README_JA, fragment: "、`01:15` CodeQL" },
      { file: README_EN, fragment: "`code-ql.yaml` (`javascript-typescript` leg) + " },
      { file: README_EN, fragment: " + `code-ql.yaml` (`actions` leg)" },
      { file: README_JA, fragment: "`code-ql.yaml`（`javascript-typescript` レグ）+ " },
      { file: README_JA, fragment: "+ `code-ql.yaml`（`actions` レグ）" },
    ],
    docSections: [
      { file: README_EN, heading: "#### Removing the credential-bearing scanners" },
      { file: README_JA, heading: "#### 資格情報を要するスキャナの撤去" },
    ],
  },
];
