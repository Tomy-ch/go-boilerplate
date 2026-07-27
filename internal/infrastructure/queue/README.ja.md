# infrastructure/queue

worker シーム（`internal/usecase/boundary/worker`）を具体的なメッセージブローカーに対して
実装するアダプタ群です。

## 配置規約

- **ブローカー非依存の契約**は、この層ではなく上位の `internal/usecase/boundary/worker`
  （シーム）に置きます。infrastructure はそのポートを実装するだけで、抽象そのものは保持しません。
- **ブローカー固有のアダプタ**は `queue/<broker>/`（例: `queue/sqs`）に置きます。パッケージ名を
  ブローカー名にすることで、import 箇所で具体技術が見えるようにします。
- **ブローカー間で共有するコード**は `queue/` 直下に置きます。共有コードは、2 つ以上のアダプタが
  具体的な実装詳細を重複させたときにのみ抽出します。先回りして設計するのではなく、観測された重複から
  ヘルパを引き上げます。

## SQS は実例の一つであって前提ではない

seam はブローカー非依存ですが、抽象だけを配ったテンプレートは何も証明しません。ポートは、実物が 1 本通って初めて信用できるものになります。そこで 1 つのブローカーだけを具体的に実装しており、SQS がそれにあたります。これは**リファレンス**であって前提ではありません。差し替えは `queue/` 配下に兄弟パッケージを足すだけで、この層より上は何も変わりません。

この主張を建前で終わらせないために、各クラウドのキューをローカルで開発する際のコンテナを挙げます。fork 先が初日から同じ開発ループを立ち上げられるようにするためです。

|プロバイダ|サービス|ローカルコンテナ|ライセンス|提供元|
|---|---|---|---|---|
|AWS|SQS|`softwaremill/elasticmq-native`|Apache-2.0|SoftwareMill|
|Azure|Queue Storage|`mcr.microsoft.com/azure-storage/azurite`|MIT|Microsoft|
|GCP|Pub/Sub|`gcr.io/google.com/cloudsdktool/google-cloud-cli:emulators`|Google Cloud SDK の条項|Google|

選定基準はここでの他の依存と同じく「1 コンポーネント 1 責務」（[ADR-0068](../../../docs/adr/0068-library-selection-policy.md)）です。したがって、クラウド一式をエミュレートするスイートよりも単機能のエミュレータを優先します。各選択の補足は次のとおりです。

- **ElasticMQ** は SQS だけをエミュレートし、native イメージはテスト実行ごとに起動できる程度に小さい。多サービス対応の AWS エミュレータはより広い範囲を賄いますが、その代償として 1 コンテナに多数の責務を抱えた依存と、商用ゲートのある機能ラインを持ち込みます
- **Azurite** は Microsoft 自身のエミュレータで、Blob と Queue の**両方**を賄います。したがって Azure は 2 つの seam に対して 1 コンテナで済みます。Service Bus のセマンティクス（トピック / サブスクリプション / セッション）が必要な場合はファーストパーティのエミュレータが別にありますが、proprietary な EULA で配布され SQL Server を同伴するため、Queue Storage では本当に足りない場合にのみ選んでください
- **Pub/Sub** にはファーストパーティの単体イメージがなく、エミュレータは Google Cloud CLI のコンポーネントとして `:emulators` タグで Google から提供されます。他の 2 つより重量級です。第三者による軽量ラッパーも存在しますが、その分だけ責任の所在は弱くなります
