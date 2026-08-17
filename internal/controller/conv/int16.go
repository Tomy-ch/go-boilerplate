package conv

import (
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"
)

// Int16sPtr は、任意指定の int32 配列クエリパラメータを int16 のスライスへ変換します。
// nil と空配列はいずれも「絞り込みなし」を表す nil を返します。要素が int16 の範囲を外れる場合はエラーを返します。
//
// 範囲外は OpenAPI の minimum / maximum がリクエスト検証で先に弾くため通常は到達しませんが、
// 変換先の DB 列は SMALLINT であり、spec を迂回した呼び出しが静かに切り捨てられるより落ちるほうが安全です。
func Int16sPtr(p *[]int32) ([]int16, error) {
	if p == nil || len(*p) == 0 {
		return nil, nil
	}

	values := make([]int16, len(*p))
	for i, v := range *p {
		converted, err := safecast.IntToInt16(int(v))
		if err != nil {
			return nil, xerrors.Wrap(err, "invalid int16 query parameter")
		}
		values[i] = converted
	}
	return values, nil
}
