package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFixture(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	return path
}

func validSourceGraph() sourceGraph {
	return sourceGraph{
		Directed: true,
		Nodes: []map[string]any{
			{"id": "b", "label": "B", "file_type": "code", "source_file": "b.go", "source_location": "L2", "z": "last"},
			{
				"id": "a", "label": "A", "file_type": "document", "source_file": "README.md",
				"source_location": "L1", "community": json.Number("2"),
			},
		},
		Links: []map[string]any{
			{
				"source": "b", "target": "a", "relation": "calls", "source_file": "b.go",
				"source_location": "L3", "confidence": "EXTRACTED",
			},
			{
				"source": "a", "target": "b", "relation": "references", "source_file": "README.md",
				"source_location": "L4", "weight": json.Number("0.8"),
			},
		},
	}
}

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Graphifyの全ノードと全エッジを3ファイルへ出力する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			input := writeFixture(t, root, "graph.json", `{
  "directed": true,
  "multigraph": false,
  "graph": {"hyperedges": [{"id": "flow"}]},
  "nodes": [
    {"id": "z", "label": "Z", "file_type": "code", "source_file": "z.go", "source_location": "L9"},
    {"id": "a", "label": "A", "file_type": "document", "source_file": "README.md", "source_location": "L1"}
  ],
  "links": [
    {"source": "z", "target": "a", "relation": "references", "confidence": "EXTRACTED"}
  ]
}`)
			output := filepath.Join(root, "graphify-out")

			require.NoError(t, run(input, output))

			nodes, err := os.ReadFile(filepath.Join(output, "nodes.json")) //nolint:gosec // テスト内で組み立てた一時パス
			require.NoError(t, err)
			edges, err := os.ReadFile(filepath.Join(output, "edges.json")) //nolint:gosec // テスト内で組み立てた一時パス
			require.NoError(t, err)
			metadataBody, err := os.ReadFile(filepath.Join(output, "metadata.json")) //nolint:gosec // テスト内で組み立てた一時パス
			require.NoError(t, err)
			assert.Contains(t, string(nodes), `"id": "a"`)
			assert.Contains(t, string(edges), `"relation": "references"`)
			assert.Contains(t, string(metadataBody), `"analysis_mode": "full"`)

			firstNodes := append([]byte(nil), nodes...)
			firstEdges := append([]byte(nil), edges...)
			firstMetadata := append([]byte(nil), metadataBody...)
			require.NoError(t, run(input, output))
			secondNodes, err := os.ReadFile(filepath.Join(output, "nodes.json")) //nolint:gosec // テスト内で組み立てた一時パス
			require.NoError(t, err)
			secondEdges, err := os.ReadFile(filepath.Join(output, "edges.json")) //nolint:gosec // テスト内で組み立てた一時パス
			require.NoError(t, err)
			secondMetadata, err := os.ReadFile(filepath.Join(output, "metadata.json")) //nolint:gosec // テスト内で組み立てた一時パス
			require.NoError(t, err)
			assert.Equal(t, firstNodes, secondNodes)
			assert.Equal(t, firstEdges, secondEdges)
			assert.Equal(t, firstMetadata, secondMetadata)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("入力が存在しなければ出力を作らずエラーを返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			output := filepath.Join(root, "graphify-out")

			err := run(filepath.Join(root, "missing.json"), output)

			require.ErrorContains(t, err, "Graphify graph の読み込み")
			_, statErr := os.Stat(output)
			require.ErrorIs(t, statErr, os.ErrNotExist)
		})

		t.Run("出力先がファイルなら書き出し失敗を返す", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			input := writeFixture(t, root, "graph.json", `{"nodes":[{"id":"a"}],"links":[]}`)
			output := writeFixture(t, root, "output", "not a directory")

			err := run(input, output)

			require.ErrorContains(t, err, "graphify-out への書き出し")
		})
	})
}

func Test_loadGraph(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数値精度を保ったままnode-link JSONを読み込む", func(t *testing.T) {
			t.Parallel()
			path := writeFixture(t, t.TempDir(), "graph.json", `{"nodes":[],"links":[],"input_tokens":9007199254740993}`)

			got, err := loadGraph(path)

			require.NoError(t, err)
			assert.Equal(t, json.Number("9007199254740993"), got.InputTokens)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON値が後続していれば単一graphとして受理しない", func(t *testing.T) {
			t.Parallel()
			path := writeFixture(t, t.TempDir(), "graph.json", `{"nodes":[]} {"nodes":[]}`)

			_, err := loadGraph(path)

			require.ErrorContains(t, err, "trailing JSON value")
		})

		t.Run("JSONとして壊れていればdecodeエラーを返す", func(t *testing.T) {
			t.Parallel()
			path := writeFixture(t, t.TempDir(), "graph.json", `{"nodes":`)

			_, err := loadGraph(path)

			require.ErrorContains(t, err, "decode")
		})
	})
}

func Test_normalize(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("node IDとedge内容で順序を安定化する", func(t *testing.T) {
			t.Parallel()

			got, err := normalize(validSourceGraph())

			require.NoError(t, err)
			assert.Equal(t, []string{"a", "b"}, []string{got.Nodes[0].ID, got.Nodes[1].ID})
			assert.Equal(t, "a", got.Edges[0].Source)
			assert.Equal(t, "b", got.Edges[1].Source)
			assert.Equal(t, counts{Nodes: 2, Edges: 2}, got.Metadata.Counts)
		})

		t.Run("将来のNetworkX形式であるedgesキーも受理する", func(t *testing.T) {
			t.Parallel()
			source := validSourceGraph()
			source.Edges = source.Links
			source.Links = nil

			got, err := normalize(source)

			require.NoError(t, err)
			assert.Len(t, got.Edges, 2)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("参照先nodeが無いedgeを拒否する", func(t *testing.T) {
			t.Parallel()
			source := validSourceGraph()
			source.Links[0]["target"] = "missing"

			_, err := normalize(source)

			require.ErrorContains(t, err, `edge target "missing"`)
		})

		t.Run("node IDが重複していれば拒否する", func(t *testing.T) {
			t.Parallel()
			source := validSourceGraph()
			source.Nodes[1]["id"] = "b"

			_, err := normalize(source)

			require.ErrorContains(t, err, `node ID "b" が重複`)
		})

		t.Run("linksとedgesの二重入力を拒否する", func(t *testing.T) {
			t.Parallel()
			source := validSourceGraph()
			source.Edges = []map[string]any{{"source": "a", "target": "b", "relation": "uses"}}

			_, err := normalize(source)

			require.ErrorContains(t, err, "links と edges が同時")
		})

		t.Run("空の解析結果で既存exportを上書きしない", func(t *testing.T) {
			t.Parallel()

			_, err := normalize(sourceGraph{})

			require.ErrorContains(t, err, "node が1件もありません")
		})
	})
}

func Test_normalizeNode(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Graphify固有属性をsemantic attributesとして保持する", func(t *testing.T) {
			t.Parallel()
			raw := map[string]any{
				"id": "user_service", "label": "UserService", "file_type": "code",
				"source_file": "user.go", "source_location": "L10", "community": json.Number("3"),
			}

			got, err := normalizeNode(raw)

			require.NoError(t, err)
			assert.Equal(t, "user_service", got.ID)
			assert.Equal(t, "code", got.Kind)
			assert.Equal(t, "UserService", got.Name)
			assert.Equal(t, &location{File: "user.go", Location: "L10"}, got.Location)
			assert.Equal(t, json.Number("3"), got.Attributes["community"])
		})

		t.Run("labelとkindが無ければIDとunknownで補完する", func(t *testing.T) {
			t.Parallel()

			got, err := normalizeNode(map[string]any{"id": "external"})

			require.NoError(t, err)
			assert.Equal(t, "external", got.Name)
			assert.Equal(t, "unknown", got.Kind)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDが空なら拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeNode(map[string]any{"id": ""})

			require.ErrorContains(t, err, "node id が不正")
		})

		t.Run("IDが文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeNode(map[string]any{"id": json.Number("1")})

			require.ErrorContains(t, err, "node id が不正")
		})

		t.Run("labelが文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeNode(map[string]any{"id": "a", "label": json.Number("1")})

			require.ErrorContains(t, err, "name が不正")
		})

		t.Run("kindが文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeNode(map[string]any{"id": "a", "file_type": json.Number("1")})

			require.ErrorContains(t, err, "kind が不正")
		})

		t.Run("source fileが文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeNode(map[string]any{"id": "a", "source_file": json.Number("1")})

			require.ErrorContains(t, err, "source_file が不正")
		})

		t.Run("source locationが文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeNode(map[string]any{"id": "a", "source_location": json.Number("1")})

			require.ErrorContains(t, err, "source_location が不正")
		})
	})
}

func Test_normalizeEdge(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("confidenceとweightをsemantic attributesとして保持する", func(t *testing.T) {
			t.Parallel()
			raw := map[string]any{
				"source": "a", "target": "b", "relation": "calls",
				"source_file": "a.go", "source_location": "L12",
				"confidence": "INFERRED", "weight": json.Number("0.8"),
			}

			got, err := normalizeEdge(raw)

			require.NoError(t, err)
			assert.Equal(t, "a", got.Source)
			assert.Equal(t, "b", got.Target)
			assert.Equal(t, "calls", got.Relation)
			assert.Equal(t, "INFERRED", got.Attributes["confidence"])
			assert.NotEmpty(t, got.sortKey)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("relationが無ければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeEdge(map[string]any{"source": "a", "target": "b"})

			require.ErrorContains(t, err, "relation が不正")
		})

		t.Run("sourceが空なら拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeEdge(map[string]any{"source": "", "target": "b", "relation": "calls"})

			require.ErrorContains(t, err, "edge source が不正")
		})

		t.Run("sourceが文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeEdge(map[string]any{"source": json.Number("1"), "target": "b", "relation": "calls"})

			require.ErrorContains(t, err, "edge source が不正")
		})

		t.Run("targetが空なら拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeEdge(map[string]any{"source": "a", "target": "", "relation": "calls"})

			require.ErrorContains(t, err, "edge target が不正")
		})

		t.Run("targetが文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeEdge(map[string]any{"source": "a", "target": json.Number("1"), "relation": "calls"})

			require.ErrorContains(t, err, "edge target が不正")
		})

		t.Run("relationが文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeEdge(map[string]any{"source": "a", "target": "b", "relation": json.Number("1")})

			require.ErrorContains(t, err, "relation が不正")
		})

		t.Run("source fileが文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeEdge(map[string]any{
				"source": "a", "target": "b", "relation": "calls", "source_file": json.Number("1"),
			})

			require.ErrorContains(t, err, "source_file が不正")
		})

		t.Run("source locationが文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeEdge(map[string]any{
				"source": "a", "target": "b", "relation": "calls", "source_location": json.Number("1"),
			})

			require.ErrorContains(t, err, "source_location が不正")
		})

		t.Run("marshal不能な属性からsort keyを作らない", func(t *testing.T) {
			t.Parallel()

			_, err := normalizeEdge(map[string]any{"source": "a", "target": "b", "relation": "calls", "invalid": func() {}})

			require.ErrorContains(t, err, "sort key")
		})
	})
}

func Test_preferredString(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最初に存在する候補の文字列を返す", func(t *testing.T) {
			t.Parallel()

			got, err := preferredString(map[string]any{"name": "fallback", "label": "primary"}, "label", "name")

			require.NoError(t, err)
			assert.Equal(t, "primary", got)
		})

		t.Run("候補が無ければ空文字を返す", func(t *testing.T) {
			t.Parallel()

			got, err := preferredString(map[string]any{"label": nil}, "label", "name")

			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("候補値が文字列でなければ拒否する", func(t *testing.T) {
			t.Parallel()

			_, err := preferredString(map[string]any{"id": json.Number("1")}, "id")

			require.ErrorContains(t, err, "string ではありません")
		})
	})
}

func Test_cloneWithoutKeys(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既知フィールドだけを除いて元mapを変更しない", func(t *testing.T) {
			t.Parallel()
			source := map[string]any{"id": "a", "confidence": "EXTRACTED"}

			got := cloneWithoutKeys(source, "id")

			assert.Equal(t, map[string]any{"confidence": "EXTRACTED"}, got)
			assert.Equal(t, "a", source["id"])
		})

		t.Run("全フィールドが既知なら空mapではなくnilを返す", func(t *testing.T) {
			t.Parallel()

			got := cloneWithoutKeys(map[string]any{"id": "a"}, "id")

			assert.Nil(t, got)
		})
	})
}

func Test_newLocation(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fileまたは位置があればsource locationを返す", func(t *testing.T) {
			t.Parallel()

			got := newLocation("user.go", "L8")

			assert.Equal(t, &location{File: "user.go", Location: "L8"}, got)
		})

		t.Run("両方が空なら省略可能なnilを返す", func(t *testing.T) {
			t.Parallel()

			got := newLocation("", "")

			assert.Nil(t, got)
		})
	})
}

func Test_buildMetadata(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時刻を含めず解析属性と件数を保持する", func(t *testing.T) {
			t.Parallel()
			source := sourceGraph{
				Directed:    true,
				Graph:       map[string]any{"name": "repository"},
				Hyperedges:  []map[string]any{{"id": "flow"}},
				InputTokens: json.Number("10"),
			}

			got := buildMetadata(source, 4, 3)

			assert.Equal(t, schemaVersion, got.SchemaVersion)
			assert.Equal(t, "full", got.AnalysisMode)
			assert.Equal(t, counts{Nodes: 4, Edges: 3}, got.Counts)
			assert.Equal(t, json.Number("10"), got.SourceAttributes["input_tokens"])
		})

		t.Run("0のtoken数も解析属性として保持する", func(t *testing.T) {
			t.Parallel()

			got := buildMetadata(sourceGraph{Multigraph: true, OutputTokens: json.Number("0")}, 1, 0)

			assert.True(t, got.Source.Multigraph)
			assert.Equal(t, json.Number("0"), got.SourceAttributes["output_tokens"])
		})

		t.Run("解析拡張属性が無ければ空objectを出力しない", func(t *testing.T) {
			t.Parallel()

			got := buildMetadata(sourceGraph{}, 1, 0)

			assert.Nil(t, got.SourceAttributes)
		})
	})
}

func Test_marshalDocuments(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全ドキュメントを有効なJSONとしてmarshalする", func(t *testing.T) {
			t.Parallel()
			model, err := normalize(validSourceGraph())
			require.NoError(t, err)

			got, err := marshalDocuments(model)

			require.NoError(t, err)
			assert.True(t, json.Valid(got.nodes))
			assert.True(t, json.Valid(got.edges))
			assert.True(t, json.Valid(got.metadata))
			assert.Equal(t, byte('\n'), got.nodes[len(got.nodes)-1])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("marshal不能なsemantic attributeを成功扱いしない", func(t *testing.T) {
			t.Parallel()
			model := semanticModel{Nodes: []node{{
				ID: "a", Kind: "code", Name: "A", Attributes: map[string]any{"invalid": func() {}},
			}}}

			_, err := marshalDocuments(model)

			require.Error(t, err)
		})
	})
}

func Test_writeDocuments(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("3ファイルを指定ディレクトリへ書き出す", func(t *testing.T) {
			t.Parallel()
			output := filepath.Join(t.TempDir(), "graphify-out")
			docs := documents{nodes: []byte("nodes"), edges: []byte("edges"), metadata: []byte("metadata")}

			require.NoError(t, writeDocuments(output, docs))

			nodes, err := os.ReadFile(filepath.Join(output, "nodes.json")) //nolint:gosec // テスト内で組み立てた一時パス
			require.NoError(t, err)
			edges, err := os.ReadFile(filepath.Join(output, "edges.json")) //nolint:gosec // テスト内で組み立てた一時パス
			require.NoError(t, err)
			metadataBody, err := os.ReadFile(filepath.Join(output, "metadata.json")) //nolint:gosec // テスト内で組み立てた一時パス
			require.NoError(t, err)
			assert.Equal(t, "nodes", string(nodes))
			assert.Equal(t, "edges", string(edges))
			assert.Equal(t, "metadata", string(metadataBody))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("出力先が空なら現在ディレクトリへ誤出力しない", func(t *testing.T) {
			t.Parallel()

			err := writeDocuments("", documents{})

			require.ErrorContains(t, err, "output directory が空")
		})

		t.Run("出力先が既存ファイルならディレクトリとして扱わない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			output := writeFixture(t, root, "output", "file")

			err := writeDocuments(output, documents{})

			require.Error(t, err)
		})
	})
}
