# Test Perspectives Subagent

各 scaffold-X skill が実装に入る前に **必ず subagent を起動して test 観点を立てさせる**。

## 起動方法

- `Agent` tool で起動（`subagent_type=general-purpose`、または専用 subagent が定義されていればそれ）
- **入力**: 該当 layer README の Test Strategy 節 + 該当層の spec
- **出力**: 「この層で見るべきテスト観点リスト」を返す
- **利用**: skill 本体がこの観点リストを input としてテスト + 実装を書く

## 層固有の主要観点

層によって観点が大きく違うため、共通 prompt ではなく **layer 固有の subagent 呼び出し** にする。

| layer | 主な test 観点 |
| --- | --- |
| domain | invariant 保護、状態遷移の正当性、value object の境界値 |
| usecase | workflow 順序、mock 戦略、transaction 境界、error 伝播、DTO 変換 |
| controller | HTTP I/O 変換、validation、apperror → status mapping、middleware 連携 |
| infra-db | real DB + rollback、sqlc gen ラップ、pgerror 正規化、観測性 |
