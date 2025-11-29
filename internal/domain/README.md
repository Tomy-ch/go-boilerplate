# ドメイン層（`internal/domain`）ガイド

## オニオンアーキテクチャでの役割

- **ビジネスの中心（核）**。エンティティ、値オブジェクト、ドメインサービス、ドメインイベントなどの**本質的なルール**を表現する。
- 外界（HTTP/DB/UI）への関心は一切持たず、**純粋なモデルと言語**で振る舞いを定義する。
- 変更に最も強い層。**ここが壊れない限りプロダクトは保守できる**という前提で守る。

## このboilerplateでの役割

- `internal/domain/<bounded-context>/<aggregate>/`配下に **Entity / ValueObject / DomainService / Repository(IF)** を配置。
- 例）`internal/domain/user/`
  - `entity.go`：`User`（不変条件・ドメインメソッド）
  - `value.go`：`Email` など値オブジェクト
  - `service.go`：集約横断の**純関数**ドメインサービス
  - `repository.go`：**インフラ実装に依存しない**永続化の抽象（IF のみ）
- **副作用を持たない関数（純関数）**でルールを記述するのが原則。
  I/O・時刻・乱数は**引数で注入された値**に依存させる。
- 状態変更は**エンティティのメソッド**で行い、外部リソースへのアクセスはしない（それはUsecase/Infraの責務）。
- 原則全ての型は**不変（immutable）**にし、getter経由で読み取り、振る舞いメソッドでのみ状態変更する。
- 依存関係は**コンストラクタで注入**し、グローバル変数やシングルトンは使用しない。
- **ドメインイベント**や**ビジネスルール**はここで定義し、発行はUsecase層で行う。
- 外部ライブラリは**標準ライブラリのみ**（context, time, errors, fmt など）。ORM・SQL実行・HTTPクライアントなどI/O系は一切持ち込まない。
  - 例外として、go言語でディファクトスタンダードな型のライブラリは`pkg`を通じて薄いラッパーを作成し、ドメインで使用することを許容。
    - 例1: UUID → `github.com/google/uuid` → `pkg/uuid`の`UUID`として定義
    - 例2: Decimal → `github.com/shopspring/decimal` → `pkg/decimal`の`Decimal`として定義

## 実装上の注意点

### 命名/構造

- 構造体は`Entity`、複数形は`Entities`とする。
- インフラ層で実装されるリポジトリのインターフェースは`Repository`とする。
- パッケージ名は機能名（例：user）で統一する。
- インスタンスの生成関数名は `New` で統一する。

### コンストラクタ経由以外でセットしない

- 必須フィールドや不変条件の検証は `NewXxx(...)` コンストラクタでのみ実施。
- 生成後に整合性を壊す setter は置かない。**ミューテータは振る舞い（メソッド）として提供**する。

### 取得は getter 経由

- 外部からの読み取りは `ID()`, `Name()`, `Email()` のような **明示的 getter** 経由にする（直接フィールド公開禁止）。
- 値オブジェクトは**不変**（immutable）を基本にする。

### 構造体にタグ（`json`, `db`, `validate` 等）を打たない

- ドメインは外界の都合を知らない。シリアライズやDBマッピングのタグは**DTO/Infra**側にのみ置く。

### 時刻・ID の扱い

- 生成時刻やIDは**引数で受け取る**

### バリデーション

- 形式チェックは**値オブジェクト**へ（例：`NewEmail` 内でメール形式を検証）。
- ルール検証は**エンティティ/ドメインサービス**へ（例：状態遷移・在庫引当）。

## インフラ層の依存性逆転

- `repository.go`には**永続化の抽象（interface）**のみ定義する。

  ```go
  type Repository interface {
      Save(ctx context.Context, u *Entity) error
      FindByID(ctx context.Context, id uuid.UUID) (*Entity, error)
  }
  ```

- 実装は **Infra 層**（例：`internal/infrastructure/persistence/postgres/user/repository.go`）で行い、`sqlc` 生成物を使って**ドメインにマップ**する。

## 呼び出せる層

- **呼び出し元**：Usecase層（アプリケーションサービス）からのみ直接利用される想定。
- **ドメインが呼べるのはドメイン内のみ**（他層を呼ばない）。他集約に触れる場合は**ドメインサービス（純関数）**で表現し、参照データはUsecaseが取得して引数で渡す。
  - 例外として、`Product`に対する`ProductStatus`のような**参照専用の集約**は直接参照を許容。

## やっていいこと / いけないこと(まとめ)

### Do

- **コンストラクタで完全性を保証**し、**振る舞いメソッドでのみ状態遷移**させる
- **値オブジェクト**で表現力を上げ、形式/整合性を**型で担保**する
- 複数集約にまたがるルールは**ドメインサービス（純関数）**で表現し、必要なデータは**引数で受ける**
- 永続化は**Repository IF**に抽象化し、**実装は Infra**へ委ねる
- テストは**テーブル駆動**で、**外部I/Oなし**に完結する（エンティティ/VO/サービスの単体テスト）

### Don’t

- `http.*`, `echo.*`, `sqlc` 型、`sql.*`、ログ出力、環境変数読み込み等の**外部要素を持ち込む**
- `json`/`db`/`validate` など**タグを付ける**
- setter を乱立させて**不変条件を破れる状態**にする
- DB主導の設計（列名・外部キー設計）を**ドメインの形に直接反映**する
- インフラの実装（SQLやトランザクション）を**ドメインメソッドの内側で前提にする**

```go
// constant.go

// ドメイン名
package user

// バリデーションで使う検証用の定数
const (
    minLength           = 1
    maxFirstNameLength  = 100
    maxLastNameLength   = 100
    maxPasswordLength   = 255
)

```

```go
// error.go

// ドメイン名
package user

var (
    // このドメインの基底エラー
    // apperror.ErrValidation(422)をラップし、フィールドごとに特定できるようにする
    errInvalid               = xerrors.Wrap(apperror.ErrValidation, "invalid user")
    // 基底エラーをラップしてフィールドごとに特定できるようにする
    ErrInvalidID             = xerrors.Wrap(errInvalid, "id: ")
    ErrInvalidFirstName      = xerrors.Wrap(errInvalid, "first name: ")
    ErrInvalidLastName       = xerrors.Wrap(errInvalid, "last name: ")
    ErrInvalidPassword       = xerrors.Wrap(errInvalid, "password: ")
)
```

```go
// user_domain.go

// Package user は、ユーザー関連のドメインを提供します。
package user

// Entityの名称固定
type Entity struct {
    id             uuid.UUID
    firstName      string
    lastName       string
    password       string
}

// 複数のエンティティをまとめて扱う場合のスライス型
// 別途定義することで、この構造体のメソッドとして振る舞いを追加できる
type Entities []Entity

// Newの名称固定
func New(
    idStr string,
    firstName string,
    lastName string,
    password string,
) (*Entity, error) {
    // IDはuuid.Parseでパースして検証する
    id, err := uuid.Parse(idStr)
    if err != nil {
        return nil, xerrors.Wrap(ErrInvalidID, err.Error())
    }

    // Stringはstringkit.InRangeで長さを検証する
    if !stringkit.InRange(firstName, minLength, maxFirstNameLength) {
        return nil, xerrors.Wrap(ErrInvalidFirstName, stringkit.ErrorMsgInRange(minLength, maxFirstNameLength, firstName))
    }

    if !stringkit.InRange(lastName, minLength, maxLastNameLength) {
        return nil, xerrors.Wrap(ErrInvalidLastName, stringkit.ErrorMsgInRange(minLength, maxLastNameLength, lastName))
    }

    if !stringkit.InRange(password, minLength, maxPasswordLength) {
        return nil, xerrors.Wrap(ErrInvalidPassword, stringkit.ErrorMsgInRange(minLength, maxPasswordLength, password))
    }

    return &Entity{
        id:             id,
        firstName:      firstName,
        lastName:       lastName,
        password:       password,
    }, nil
}

// データの取得はgetter経由で行う
func (e *Entity) ID() uuid.UUID { return e.id }
func (e *Entity) FirstName() string { return e.firstName }
func (e *Entity) LastName() string { return e.lastName }
func (e *Entity) Password() string { return e.password }

// ビジネスロジックはメソッドとして提供する
func (e *Entity) FullName() string { return e.firstName + " " + e.lastName }

```

```go
// user_repository.go

// ↓はモック生成用のgo:generateコメント
//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
package user

// インフラ層で実装されるリポジトリのインターフェース
type Repository interface {
    GetAllUsers(ctx context.Context, limit, offset int) (Entities, error)
}

```
