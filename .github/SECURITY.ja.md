# セキュリティポリシー

> このファイルは canonical な英語版 [SECURITY.md](SECURITY.md) の日本語訳です。内容の正本は英語版であり、
> 更新は英語版から反映してください。

このリポジトリは boilerplate テンプレートです。本ファイルはテンプレートとして提供される初期ポリシーであり、
**継承先は連絡先・対象バージョン等を自環境に合わせて修正**してください（環境依存箇所は `※環境に合わせて修正` と明記）。

## 脆弱性の報告

> [!IMPORTANT]
> 以下の連絡先はプレースホルダです。継承先で必ず実際の窓口へ差し替えてください。

- 脆弱性は **公開 Issue / Pull Request では報告しないでください**（情報が公開されてしまうため）。
- 次のいずれかで **非公開** に報告してください。
  - GitHub の Private Vulnerability Reporting（リポジトリの `Security` → `Advisories` → `Report a
    vulnerability`）
  - セキュリティ窓口: `security@example.com` ※環境に合わせて修正
- 報告には以下を含めてください。
  - 再現手順 / PoC
  - 影響範囲（想定される被害・前提条件）
  - 該当するバージョン or コミット SHA
- 初回応答の目安: **X 営業日以内** ※運用に合わせて修正

### サポート対象

- 最新リリース: ✅ サポート対象
- それ以前: ❌ サポート対象外 ※運用に合わせて修正

## 成果物の検証

`.github/workflows/deploy-app.yaml` は GHCR へ push する `runtime` イメージに対し、
**push 済みイメージの digest を対象に** 次を付与します。

- cosign keyless 署名（OIDC → Fulcio → Rekor）
- SLSA provenance attestation（`actions/attest-build-provenance`）
- SBOM(SPDX) attestation（`actions/attest-sbom`）

タグは可変だが digest は不変なので、**検証は必ず digest 基準**で行ってください。
以下コマンド中の `<owner>` / `<repo>` / `<tag>` / `<digest>` は環境に合わせて置換します
（GHCR 以外へ移した場合は `ghcr.io/<owner>` 部分も読み替え）。

### 0. 対象 digest の取得

検証はタグではなく digest 基準で行うため、対象イメージのタグから digest を解決する。
マイグレーションは別イメージではなく、この同じイメージをコマンド上書き
（`docker run <image> /app/server migrate-up`）で走らせるため、解決すべき digest は 1 つ。

```bash
docker buildx imagetools inspect ghcr.io/<owner>/app:<tag> --format '{{.Manifest.Digest}}'
crane digest ghcr.io/<owner>/app:<tag>
```

### 1. cosign 署名の検証

keyless 署名は「どのワークフローが署名したか（証明書の identity）」と「OIDC 発行者」で検証します。

```bash
cosign verify \
  --certificate-identity-regexp "^https://github.com/<owner>/<repo>/\.github/workflows/deploy-app\.yaml@.*$" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  ghcr.io/<owner>/app@<digest>
```

### 2. provenance attestation の検証（どこで・何からビルドされたか）

```bash
gh attestation verify oci://ghcr.io/<owner>/app@<digest> \
  --repo <owner>/<repo> \
  --predicate-type https://slsa.dev/provenance/v1
```

### 3. SBOM(SPDX) attestation の検証（同梱物の証明）

```bash
gh attestation verify oci://ghcr.io/<owner>/app@<digest> \
  --repo <owner>/<repo> \
  --predicate-type https://spdx.dev/Document
```

> [!NOTE]
> `gh attestation verify` は GitHub Attestation store を参照するため、レジストリが OCI referrer に
> 非対応でも検証できます（deploy 側で `push-to-registry: false` にした場合も GitHub 側の記録で検証可能）。
> 一方 `cosign verify` はレジストリ上の署名（referrer）を参照します。

### 検証のデプロイゲート化（推奨）

> [!IMPORTANT]
> 検証は **デプロイ前のゲート** として CD パイプライン側に組み込むことを推奨します
> （署名・provenance・SBOM が確認できないイメージはデプロイしない）。本ファイルは検証手順のみを示します。
