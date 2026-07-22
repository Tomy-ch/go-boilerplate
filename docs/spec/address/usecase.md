# Address — Usecase Spec

> address は外部の郵便番号 lookup サービスを参照するだけの非永続 API であり、固有の domain
> エンティティ・テーブルを持たない。そのため lean-a-spec（ADR-0082）の「原則 domain + usecase」
> から domain.md を省略し、本 usecase.md のみで表現する。外部 lookup への意味的 gateway は
> `internal/usecase/boundary/address.Gateway`（boundary IF）として Dependencies 節に明示する。

## Overview

郵便番号住所補完ユースケースは、正規化済み 7 桁の郵便番号から住所候補（都道府県名・市区町村・町域）を
外部 lookup サービス（gateway 経由）で取得し、外部が返す都道府県名（フル表記）を
`prefecture.Repository.FindByName`（完全一致）で解決して `prefecture_id` を埋めた候補一覧を返す thin orchestrator。
外部レスポンスの型・エラーは gateway 内で完結しており、本 usecase は boundary DTO（`address.Candidate`）のみを扱う
（外部型の内層非漏洩）。

本 API の実証目的は「外部サービスが落ちても登録/購入ジャーニーを止めない」degrade であり、exchangerate
（外部不通で 503 = `ErrUnavailable`）とは正反対に、外部 lookup 障害時は error を返さず
`200 + 空候補 + IsFallback: true` へ倒す。応答は 3 状態に分離する:

1. 外部 lookup 障害（gateway が sentinel エラー）→ 空候補 + `IsFallback: true`（usecase がエラーを握り、error を返さない。障害自体は gateway の Infra span と httpclient substrate の downstream 失敗メトリクスで観測できるため、usecase 側では追加のログを出さない）
2. 正常応答・該当なし → 空候補 + `IsFallback: false`
3. 正常応答・候補あり・県名解決不能（`FindByName` が NotFound）→ 候補は返し `PrefectureID: nil` + `IsFallback: false`（市区町村・町域の部分補完は生かす）

ただし `prefecture.Repository` の NotFound 以外のエラー（DB 障害等）は degrade 対象外＝通常どおり error を返す
（degrade は外部 lookup 障害のための機構であり、自 DB 障害を握ると障害検知を阻害するため）。県名は同一郵便番号で
ほぼ単一のため、`FindByName` 呼び出しは県名で重複排除してから行い呼び出し回数を最小化する（高々 1〜2 回）。

## Interface

```yaml
package: internal/usecase/address
name: Usecase
methods:
  - name: LookupByPostalCode
    signature: LookupByPostalCode(ctx context.Context, postalCode string) (*Result, error)
```

## DTOs

```yaml
- name: Result
  description: 住所補完結果の usecase 出力 DTO。
  fields:
    - name: Candidates
      type: "[]*CandidateView"
    - name: IsFallback
      type: bool
- name: CandidateView
  description: 住所候補 1 件。県名解決に成功した場合のみ PrefectureID を埋める。
  fields:
    - name: PrefectureID
      type: "*uuid.UUID"   # 県名解決不能時は nil
    - name: PrefectureName
      type: string
    - name: City
      type: string
    - name: Town
      type: string
```

## Dependencies

```yaml
- tracer                 # observability.TracerFactory -> LayerTracer
- address_gateway        # internal/usecase/boundary/address.Gateway（外部 lookup への意味的 gateway。外部型は gateway 内で消える）
- prefecture_repository  # domain/prefecture.Repository（FindByName で県名→prefecture_id を完全一致解決）
```

## Workflow

### LookupByPostalCode

```yaml
tx_required: false
steps:
  - address_gateway.Lookup で外部 lookup サービスから住所候補（boundary DTO）を取得する
  - 取得に失敗した場合は error を返さず 空候補 + IsFallback:true で degrade する（状態1。障害は gateway span / httpclient メトリクスで観測）
  - 候補ごとに県名を重複排除しつつ prefecture_repository.FindByName で prefecture_id を解決する
  - FindByName が NotFound の場合は PrefectureID: nil のまま候補を残す（状態3の部分 degrade）
  - FindByName が NotFound 以外のエラーの場合は degrade せず error を伝播する
  - 解決結果を CandidateView へ写像し Result{Candidates, IsFallback:false} を返す（状態2は候補 0 件・IsFallback:false）
calls:
  - address_gateway.Lookup
  - prefecture_repository.FindByName
errors:
  - address_gateway.Lookup の障害は握り潰して degrade（error を返さない）
  - prefecture_repository.FindByName の NotFound は部分 degrade（PrefectureID:nil）、それ以外は伝播
```
