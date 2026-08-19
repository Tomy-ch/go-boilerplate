package architest

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// closureNotSubject は、properties を 1 つも持たず、守るべきフィールドが無いことを表します。
	closureNotSubject closureVerdict = iota
	// closureEffective は、宣言があり、それが composition の全 properties を見ていることを表します。
	closureEffective
	// closureMissing は、宣言が composition のどこにも無いことを表します。
	closureMissing
	// closureBlind は、宣言はあるが、同じスキーマオブジェクトに無い properties を見ていないことを表します。
	closureBlind
)

// closureVerdict は、composition が未知フィールドを拒否できているかの判定結果です。
type closureVerdict int

// requestSchemaSubject は、検査対象となったスキーマ 1 件と、その報告用の位置です。
type requestSchemaSubject struct {
	location string
	schema   *openapi3.Schema
}

// TestRequestBodyRejectsUnknownFields は、リクエストボディから到達できる全ての object スキーマが
// 未知フィールドを拒否することを機械検証する。この規則は
// openapi/components/requests/README.md と openapi/components/schemas/README.md の Rules が定める。
//
// 突き合わせる集合は「リクエストボディから到達可能な object スキーマ」と「未知フィールドを実効的に
// 拒否するスキーマ」で、前者 ⊆ 後者 を表明する。宣言が無いスキーマは契約に書かれていないフィールドを
// 黙って受け取るため、クライアント側のタイポや古いフィールド名が無視される。
//
// 検査は requestBody 直下で止めず properties / items / allOf を再帰する。ネストした要素スキーマ
// （購入明細・商品画像）はトップレベルの宣言に守られず、そこだけ素通りが残るためである。
//
// 読み出すのは生成物ではなく手書き正本の openapi.yaml で、理由は TestRouteSpecParity と同じ。
//
// 対象が 0 件でも失敗させない。サンプル API を撤去すると requestBody を持つ operation が
// 残らないためで、0 件と壊れた走査を区別するのは Test_walkRequestSchema と Test_evaluateClosure
// が担う（internal/architest/README.md の Notes）。
func TestRequestBodyRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	subjects := collectRequestBodySchemas(t)
	t.Logf("検査した object スキーマ: %d 件", len(subjects))

	missing := make([]string, 0, len(subjects))
	blind := make([]string, 0, len(subjects))
	for _, subject := range subjects {
		verdict, fields := evaluateClosure(subject.schema)
		switch verdict {
		case closureMissing:
			missing = append(missing, fmt.Sprintf("%s (properties: %v)", subject.location, fields))
		case closureBlind:
			blind = append(blind, fmt.Sprintf("%s (見えていない properties: %v)", subject.location, fields))
		case closureNotSubject, closureEffective:
		}
	}
	sort.Strings(missing)
	sort.Strings(blind)

	assert.Empty(t, missing,
		"リクエストボディから到達する object スキーマに additionalProperties: false が無い。"+
			"契約に無いフィールドを送っても拒否されないため、宣言を追加すること")
	assert.Empty(t, blind,
		"additionalProperties: false が、同じスキーマオブジェクトに無い properties を見ていない。"+
			"allOf の兄弟が持つプロパティは宣言から見えず正当な入力まで拒否されるため、"+
			"宣言は properties を持つスキーマオブジェクト自身に置くこと")
}

// evaluateClosure は、allOf を展開した composition 全体として未知フィールドを拒否できているかを判定します。
// 第 2 戻り値は、closureMissing では守られていない properties、closureBlind では宣言から見えていない
// properties で、いずれも辞書順です。
//
// additionalProperties は JSON Schema の意味論で「同じスキーマオブジェクトの properties に一致しない
// フィールド」だけを見るため、宣言と properties が別のスキーマオブジェクトに分かれていると、
// 拒否しすぎるか（closureBlind）拒否できないかのどちらかになります。
func evaluateClosure(schema *openapi3.Schema) (closureVerdict, []string) {
	members := flattenAllOf(schema, nil, map[*openapi3.Schema]struct{}{})

	properties := map[string]struct{}{}
	closures := make([]*openapi3.Schema, 0, len(members))
	for _, member := range members {
		for name := range member.Properties {
			properties[name] = struct{}{}
		}
		if member.AdditionalProperties.Has != nil && !*member.AdditionalProperties.Has {
			closures = append(closures, member)
		}
	}

	if len(properties) == 0 {
		return closureNotSubject, nil
	}
	if len(closures) == 0 {
		return closureMissing, sortedNames(properties)
	}

	unseen := map[string]struct{}{}
	for _, closure := range closures {
		for name := range properties {
			if _, ok := closure.Properties[name]; !ok {
				unseen[name] = struct{}{}
			}
		}
	}
	if len(unseen) > 0 {
		return closureBlind, sortedNames(unseen)
	}
	return closureEffective, nil
}

// flattenAllOf は、スキーマ自身と allOf の枝を再帰的に平坦化して返します。
// composition は allOf の全ての枝を同時に満たす必要があるため、宣言と properties の所在は
// 枝をまたいで数えます。visited は allOf の循環に対する保険です。
func flattenAllOf(
	schema *openapi3.Schema, acc []*openapi3.Schema, visited map[*openapi3.Schema]struct{},
) []*openapi3.Schema {
	if schema == nil {
		return acc
	}
	if _, seen := visited[schema]; seen {
		return acc
	}
	visited[schema] = struct{}{}

	acc = append(acc, schema)
	for _, branch := range schema.AllOf {
		if branch != nil {
			acc = flattenAllOf(branch.Value, acc, visited)
		}
	}
	return acc
}

// collectRequestBodySchemas は、手書き正本の spec を読み、全 operation のリクエストボディから
// 到達できる object スキーマを報告用の位置つきで返します。
func collectRequestBodySchemas(t *testing.T) []requestSchemaSubject {
	t.Helper()

	loader := &openapi3.Loader{IsExternalRefsAllowed: true}
	spec, err := loader.LoadFromFile(filepath.Join(moduleRoot(t), filepath.FromSlash(specSourceFile)))
	require.NoError(t, err)
	require.NotNil(t, spec.Paths)

	subjects := make([]requestSchemaSubject, 0)
	visited := map[*openapi3.Schema]struct{}{}
	for _, path := range sortedNames(pathKeys(spec.Paths)) {
		item := spec.Paths.Value(path)
		operations := item.Operations()
		for _, method := range sortedNames(operationKeys(operations)) {
			body := operations[method].RequestBody
			if body == nil || body.Value == nil {
				continue
			}
			for _, mediaType := range sortedNames(contentKeys(body.Value.Content)) {
				subjects = walkRequestSchema(
					fmt.Sprintf("%s %s [%s]", method, path, mediaType),
					body.Value.Content[mediaType].Schema, visited, subjects,
				)
			}
		}
	}
	return subjects
}

// walkRequestSchema は、スキーマを再帰的に辿り object スキーマを収集します。
// visited は $ref の循環による無限再帰を防ぎます。この spec の $ref はファイルパス参照で、
// kin-openapi は参照ごとに別のスキーマを確保するため、同じ定義を複数の operation から
// 参照しても重複排除はされません（同じ内容が別の位置で 2 度検査されるだけで、判定は変わりません）。
//
// not は辿りません。not が述べるのは拒否したい形であって受理できるフィールドの集合ではなく、
// そこへ additionalProperties: false を要求すると、排他制約を書いただけの spec を違反と判定します。
func walkRequestSchema(
	location string, ref *openapi3.SchemaRef,
	visited map[*openapi3.Schema]struct{}, acc []requestSchemaSubject,
) []requestSchemaSubject {
	if ref == nil || ref.Value == nil {
		return acc
	}
	schema := ref.Value
	if _, seen := visited[schema]; seen {
		return acc
	}
	visited[schema] = struct{}{}

	if len(schema.Properties) > 0 || len(schema.AllOf) > 0 {
		acc = append(acc, requestSchemaSubject{location: location, schema: schema})
	}

	for _, name := range sortedNames(schema.Properties) {
		acc = walkRequestSchema(location+".properties."+name, schema.Properties[name], visited, acc)
	}
	acc = walkRequestSchema(location+".items", schema.Items, visited, acc)
	for i, branch := range schema.AllOf {
		acc = walkRequestSchema(fmt.Sprintf("%s.allOf[%d]", location, i), branch, visited, acc)
	}
	for i, branch := range schema.OneOf {
		acc = walkRequestSchema(fmt.Sprintf("%s.oneOf[%d]", location, i), branch, visited, acc)
	}
	for i, branch := range schema.AnyOf {
		acc = walkRequestSchema(fmt.Sprintf("%s.anyOf[%d]", location, i), branch, visited, acc)
	}
	return walkRequestSchema(location+".additionalProperties", schema.AdditionalProperties.Schema, visited, acc)
}

// sortedNames は、map のキーを辞書順で返します。報告と走査順を決定的にするために使います。
func sortedNames[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for name := range m {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// pathKeys は、spec のパステンプレートを集合として返します。
func pathKeys(paths *openapi3.Paths) map[string]struct{} {
	out := make(map[string]struct{}, paths.Len())
	for path := range paths.Map() {
		out[path] = struct{}{}
	}
	return out
}

// operationKeys は、operation の HTTP メソッドを集合として返します。
func operationKeys(operations map[string]*openapi3.Operation) map[string]struct{} {
	out := make(map[string]struct{}, len(operations))
	for method := range operations {
		out[method] = struct{}{}
	}
	return out
}

// contentKeys は、リクエストボディのメディアタイプを集合として返します。
func contentKeys(content openapi3.Content) map[string]struct{} {
	out := make(map[string]struct{}, len(content))
	for mediaType := range content {
		out[mediaType] = struct{}{}
	}
	return out
}

// Test_evaluateClosure は、TestRequestBodyRejectsUnknownFields の判定精度に直結する読み取り漏れを防ぐため、
// リポジトリの現状に依存しない合成スキーマで各 composition 形の判定を固定する。
func Test_evaluateClosure(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("properties と同じスキーマオブジェクトの宣言は実効的とみなす", func(t *testing.T) {
			t.Parallel()

			verdict, fields := evaluateClosure(closed(withProperties("quantity")))

			assert.Equal(t, closureEffective, verdict)
			assert.Empty(t, fields)
		})

		t.Run("allOf の基底が宣言を持ち兄弟が required だけを足す形は実効的とみなす", func(t *testing.T) {
			t.Parallel()

			verdict, fields := evaluateClosure(composed(closed(withProperties("email")), requiredOnly("email")))

			assert.Equal(t, closureEffective, verdict)
			assert.Empty(t, fields)
		})

		t.Run("allOf が入れ子でも基底の宣言は届く", func(t *testing.T) {
			t.Parallel()

			base := closed(withProperties("email"))
			verdict, fields := evaluateClosure(composed(composed(base), requiredOnly("email")))

			assert.Equal(t, closureEffective, verdict)
			assert.Empty(t, fields)
		})

		t.Run("properties を持たないスキーマは検査対象にしない", func(t *testing.T) {
			t.Parallel()

			verdict, fields := evaluateClosure(composed(requiredOnly("email")))

			assert.Equal(t, closureNotSubject, verdict)
			assert.Empty(t, fields)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("宣言がどこにも無ければ守られていない properties とともに検出する", func(t *testing.T) {
			t.Parallel()

			verdict, fields := evaluateClosure(withProperties("quantity", "productId"))

			assert.Equal(t, closureMissing, verdict)
			assert.Equal(t, []string{"productId", "quantity"}, fields)
		})

		t.Run("宣言が allOf の兄弟にあり基底の properties を見ていない形を検出する", func(t *testing.T) {
			t.Parallel()

			wrapper := composed(withProperties("email"))
			wrapper.AdditionalProperties.Has = new(false)

			verdict, fields := evaluateClosure(wrapper)

			assert.Equal(t, closureBlind, verdict)
			assert.Equal(t, []string{"email"}, fields)
		})

		t.Run("additionalProperties がスキーマの map 表現は宣言として数えない", func(t *testing.T) {
			t.Parallel()

			schema := withProperties("email")
			schema.AdditionalProperties.Schema = openapi3.NewSchemaRef("", openapi3.NewStringSchema())

			verdict, fields := evaluateClosure(schema)

			assert.Equal(t, closureMissing, verdict)
			assert.Equal(t, []string{"email"}, fields)
		})

		t.Run("allOf が循環しても再帰が止まる", func(t *testing.T) {
			t.Parallel()

			base := closed(withProperties("email"))
			root := composed(base)
			base.AllOf = openapi3.SchemaRefs{openapi3.NewSchemaRef("", root)}

			verdict, fields := evaluateClosure(root)

			assert.Equal(t, closureEffective, verdict)
			assert.Empty(t, fields)
		})

		t.Run("additionalProperties: true は宣言として数えない", func(t *testing.T) {
			t.Parallel()

			schema := withProperties("email")
			schema.AdditionalProperties.Has = new(true)

			verdict, fields := evaluateClosure(schema)

			assert.Equal(t, closureMissing, verdict)
			assert.Equal(t, []string{"email"}, fields)
		})
	})
}

// withProperties は、指定した名前のプロパティだけを持つ object スキーマを返します。
func withProperties(names ...string) *openapi3.Schema {
	schema := openapi3.NewObjectSchema()
	for _, name := range names {
		schema.Properties[name] = openapi3.NewSchemaRef("", openapi3.NewStringSchema())
	}
	return schema
}

// requiredOnly は、properties を持たず required だけを宣言する allOf の兄弟スキーマを返します。
func requiredOnly(names ...string) *openapi3.Schema {
	return &openapi3.Schema{Required: names}
}

// closed は、スキーマに additionalProperties: false を宣言して返します。
func closed(schema *openapi3.Schema) *openapi3.Schema {
	schema.AdditionalProperties.Has = new(false)
	return schema
}

// composed は、渡した枝を allOf で合成したスキーマを返します。
func composed(branches ...*openapi3.Schema) *openapi3.Schema {
	schema := &openapi3.Schema{}
	for _, branch := range branches {
		schema.AllOf = append(schema.AllOf, openapi3.NewSchemaRef("", branch))
	}
	return schema
}

// Test_walkRequestSchema は、走査が辿る経路を合成スキーマで固定する。
// oneOf / anyOf / additionalProperties のスキーマ形はこの spec がまだ使っておらず現行の
// リポジトリ内容からは到達しないため、実 spec を読む TestRequestBodyRejectsUnknownFields では
// 落としても緑のままになる。到達しない安全弁はテストで固定する（AGENTS.md の YAGNI 条項）。
func Test_walkRequestSchema(t *testing.T) {
	t.Parallel()

	collect := func(t *testing.T, root *openapi3.Schema) []string {
		t.Helper()
		subjects := walkRequestSchema(
			"root", openapi3.NewSchemaRef("", root), map[*openapi3.Schema]struct{}{}, nil,
		)
		out := make([]string, 0, len(subjects))
		for _, subject := range subjects {
			out = append(out, subject.location)
		}
		return out
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("入れ子の properties を辿る", func(t *testing.T) {
			t.Parallel()

			root := withProperties("outer")
			root.Properties["outer"] = openapi3.NewSchemaRef("", withProperties("inner"))

			assert.Equal(t, []string{"root", "root.properties.outer"}, collect(t, root))
		})

		t.Run("配列の items を辿る", func(t *testing.T) {
			t.Parallel()

			root := withProperties("list")
			array := openapi3.NewArraySchema()
			array.Items = openapi3.NewSchemaRef("", withProperties("element"))
			root.Properties["list"] = openapi3.NewSchemaRef("", array)

			assert.Equal(t, []string{"root", "root.properties.list.items"}, collect(t, root))
		})

		t.Run("allOf の枝を辿る", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, []string{"root", "root.allOf[0]"}, collect(t, composed(withProperties("base"))))
		})

		t.Run("oneOf の枝を辿る", func(t *testing.T) {
			t.Parallel()

			root := &openapi3.Schema{OneOf: openapi3.SchemaRefs{openapi3.NewSchemaRef("", withProperties("branch"))}}

			assert.Equal(t, []string{"root.oneOf[0]"}, collect(t, root))
		})

		t.Run("anyOf の枝を辿る", func(t *testing.T) {
			t.Parallel()

			root := &openapi3.Schema{AnyOf: openapi3.SchemaRefs{openapi3.NewSchemaRef("", withProperties("branch"))}}

			assert.Equal(t, []string{"root.anyOf[0]"}, collect(t, root))
		})

		t.Run("additionalProperties のスキーマ形を辿る", func(t *testing.T) {
			t.Parallel()

			root := withProperties("known")
			root.AdditionalProperties.Schema = openapi3.NewSchemaRef("", withProperties("valueField"))

			assert.Equal(t, []string{"root", "root.additionalProperties"}, collect(t, root))
		})

		t.Run("properties も allOf も持たないスキーマは subject にしない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, collect(t, openapi3.NewStringSchema()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("not の中は辿らない", func(t *testing.T) {
			t.Parallel()

			root := withProperties("accepted")
			root.Not = openapi3.NewSchemaRef("", withProperties("rejectedShape"))

			assert.Equal(t, []string{"root"}, collect(t, root),
				"not は拒否したい形であって受理できるフィールドの集合ではないため、宣言を要求してはならない")
		})

		t.Run("$ref が循環しても再帰が止まる", func(t *testing.T) {
			t.Parallel()

			root := withProperties("self")
			root.Properties["self"] = openapi3.NewSchemaRef("", root)

			assert.Equal(t, []string{"root"}, collect(t, root))
		})
	})
}
