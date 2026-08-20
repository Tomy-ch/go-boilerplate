// Package main は Graphify の node-link JSON を決定的な分割 JSON へ変換する。
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	schemaVersion    = "1.0.0"
	generatorName    = "go-boilerplate/graphify-export"
	generatorVersion = "1.0.0"
)

type sourceGraph struct {
	Directed     bool             `json:"directed"`
	Multigraph   bool             `json:"multigraph"`
	Graph        map[string]any   `json:"graph"`
	Nodes        []map[string]any `json:"nodes"`
	Links        []map[string]any `json:"links"`
	Edges        []map[string]any `json:"edges"`
	Hyperedges   []map[string]any `json:"hyperedges"`
	InputTokens  json.Number      `json:"input_tokens"`
	OutputTokens json.Number      `json:"output_tokens"`
}

type semanticModel struct {
	Nodes    []node
	Edges    []edge
	Metadata metadata
}

type node struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Name       string         `json:"name"`
	Location   *location      `json:"source_location,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type edge struct {
	Source     string         `json:"source"`
	Target     string         `json:"target"`
	Relation   string         `json:"relation"`
	Location   *location      `json:"source_location,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
	sortKey    string
}

type location struct {
	File     string `json:"file,omitempty"`
	Location string `json:"location,omitempty"`
}

type metadata struct {
	SchemaVersion    string         `json:"schema_version"`
	Generator        generator      `json:"generator"`
	AnalysisMode     string         `json:"analysis_mode"`
	Source           sourceMetadata `json:"source"`
	Counts           counts         `json:"counts"`
	SourceAttributes map[string]any `json:"source_attributes,omitempty"`
}

type generator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type sourceMetadata struct {
	Format     string `json:"format"`
	Directed   bool   `json:"directed"`
	Multigraph bool   `json:"multigraph"`
}

type counts struct {
	Nodes int `json:"nodes"`
	Edges int `json:"edges"`
}

type nodeDocument struct {
	SchemaVersion string `json:"schema_version"`
	Nodes         []node `json:"nodes"`
}

type edgeDocument struct {
	SchemaVersion string `json:"schema_version"`
	Edges         []edge `json:"edges"`
}

type documents struct {
	nodes    []byte
	edges    []byte
	metadata []byte
}

func main() {
	log.SetFlags(0)
	inputPath := filepath.Join("graphify-out", "graph.json")
	outputDir := "graphify-out"
	if len(os.Args) > 1 {
		inputPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		outputDir = os.Args[2]
	}

	if err := run(inputPath, outputDir); err != nil {
		log.Fatal(err)
	}
}

func run(inputPath, outputDir string) error {
	source, err := loadGraph(inputPath)
	if err != nil {
		return fmt.Errorf("Graphify graph の読み込み: %w", err)
	}

	model, err := normalize(source)
	if err != nil {
		return fmt.Errorf("semantic model の正規化: %w", err)
	}

	docs, err := marshalDocuments(model)
	if err != nil {
		return fmt.Errorf("JSON の生成: %w", err)
	}
	if err := writeDocuments(outputDir, docs); err != nil {
		return fmt.Errorf("graphify-out への書き出し: %w", err)
	}

	log.Printf("Graphify export complete: %d nodes, %d edges -> %s", len(model.Nodes), len(model.Edges), outputDir)

	return nil
}

func loadGraph(path string) (sourceGraph, error) {
	file, err := os.Open(path) //nolint:gosec // 開発者が指定したローカル生成物を読む CLI。
	if err != nil {
		return sourceGraph{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	decoder := json.NewDecoder(file)
	decoder.UseNumber()
	var graph sourceGraph
	if err := decoder.Decode(&graph); err != nil {
		return sourceGraph{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return sourceGraph{}, fmt.Errorf("decode %s: trailing JSON value", path)
	}

	return graph, nil
}

func normalize(source sourceGraph) (semanticModel, error) {
	if len(source.Links) > 0 && len(source.Edges) > 0 {
		return semanticModel{}, errors.New("links と edges が同時に存在します")
	}
	if len(source.Nodes) == 0 {
		return semanticModel{}, errors.New("node が1件もありません")
	}

	model := semanticModel{Nodes: make([]node, 0, len(source.Nodes))}
	nodeIDs := make(map[string]struct{}, len(source.Nodes))
	for _, raw := range source.Nodes {
		normalized, err := normalizeNode(raw)
		if err != nil {
			return semanticModel{}, err
		}
		if _, exists := nodeIDs[normalized.ID]; exists {
			return semanticModel{}, fmt.Errorf("node ID %q が重複しています", normalized.ID)
		}
		nodeIDs[normalized.ID] = struct{}{}
		model.Nodes = append(model.Nodes, normalized)
	}

	rawEdges := source.Links
	if len(rawEdges) == 0 {
		rawEdges = source.Edges
	}
	model.Edges = make([]edge, 0, len(rawEdges))
	for _, raw := range rawEdges {
		normalized, err := normalizeEdge(raw)
		if err != nil {
			return semanticModel{}, err
		}
		if _, exists := nodeIDs[normalized.Source]; !exists {
			return semanticModel{}, fmt.Errorf("edge source %q に対応する node がありません", normalized.Source)
		}
		if _, exists := nodeIDs[normalized.Target]; !exists {
			return semanticModel{}, fmt.Errorf("edge target %q に対応する node がありません", normalized.Target)
		}
		model.Edges = append(model.Edges, normalized)
	}

	sort.Slice(model.Nodes, func(i, j int) bool { return model.Nodes[i].ID < model.Nodes[j].ID })
	sort.Slice(model.Edges, func(i, j int) bool { return model.Edges[i].sortKey < model.Edges[j].sortKey })
	model.Metadata = buildMetadata(source, len(model.Nodes), len(model.Edges))

	return model, nil
}

func normalizeNode(raw map[string]any) (node, error) {
	id, err := preferredString(raw, "id")
	if err != nil {
		return node{}, fmt.Errorf("node id が不正です: %w", err)
	}
	if id == "" {
		return node{}, errors.New("node id が不正です: 空文字です")
	}
	name, err := preferredString(raw, "label", "name")
	if err != nil {
		return node{}, fmt.Errorf("node %q の name が不正です: %w", id, err)
	}
	if name == "" {
		name = id
	}
	kind, err := preferredString(raw, "file_type", "kind", "type")
	if err != nil {
		return node{}, fmt.Errorf("node %q の kind が不正です: %w", id, err)
	}
	if kind == "" {
		kind = "unknown"
	}
	file, err := preferredString(raw, "source_file")
	if err != nil {
		return node{}, fmt.Errorf("node %q の source_file が不正です: %w", id, err)
	}
	position, err := preferredString(raw, "source_location")
	if err != nil {
		return node{}, fmt.Errorf("node %q の source_location が不正です: %w", id, err)
	}

	return node{
		ID:       id,
		Kind:     kind,
		Name:     name,
		Location: newLocation(file, position),
		Attributes: cloneWithoutKeys(
			raw, "id", "label", "name", "file_type", "kind", "type", "source_file", "source_location",
		),
	}, nil
}

func normalizeEdge(raw map[string]any) (edge, error) {
	source, err := preferredString(raw, "source")
	if err != nil {
		return edge{}, fmt.Errorf("edge source が不正です: %w", err)
	}
	if source == "" {
		return edge{}, errors.New("edge source が不正です: 空文字です")
	}
	target, err := preferredString(raw, "target")
	if err != nil {
		return edge{}, fmt.Errorf("edge target が不正です: %w", err)
	}
	if target == "" {
		return edge{}, errors.New("edge target が不正です: 空文字です")
	}
	relation, err := preferredString(raw, "relation", "type", "kind")
	if err != nil {
		return edge{}, fmt.Errorf("edge %q -> %q の relation が不正です: %w", source, target, err)
	}
	if relation == "" {
		return edge{}, fmt.Errorf("edge %q -> %q の relation が不正です: 空文字です", source, target)
	}
	file, err := preferredString(raw, "source_file")
	if err != nil {
		return edge{}, fmt.Errorf("edge %q -> %q の source_file が不正です: %w", source, target, err)
	}
	position, err := preferredString(raw, "source_location")
	if err != nil {
		return edge{}, fmt.Errorf("edge %q -> %q の source_location が不正です: %w", source, target, err)
	}

	normalized := edge{
		Source:     source,
		Target:     target,
		Relation:   relation,
		Location:   newLocation(file, position),
		Attributes: cloneWithoutKeys(raw, "source", "target", "relation", "type", "kind", "source_file", "source_location"),
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return edge{}, fmt.Errorf("edge %q -> %q の sort key: %w", source, target, err)
	}
	normalized.sortKey = string(encoded)

	return normalized, nil
}

func preferredString(attributes map[string]any, keys ...string) (string, error) {
	for _, key := range keys {
		value, exists := attributes[key]
		if !exists || value == nil {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("%s は string ではありません", key)
		}

		return text, nil
	}

	return "", nil
}

func cloneWithoutKeys(source map[string]any, keys ...string) map[string]any {
	omit := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		omit[key] = struct{}{}
	}
	cloned := make(map[string]any)
	for key, value := range source {
		if _, skip := omit[key]; !skip {
			cloned[key] = value
		}
	}
	if len(cloned) == 0 {
		return nil
	}

	return cloned
}

func newLocation(file, position string) *location {
	if file == "" && position == "" {
		return nil
	}

	return &location{File: file, Location: position}
}

func buildMetadata(source sourceGraph, nodeCount, edgeCount int) metadata {
	attributes := make(map[string]any)
	if len(source.Graph) > 0 {
		attributes["graph"] = source.Graph
	}
	if len(source.Hyperedges) > 0 {
		attributes["hyperedges"] = source.Hyperedges
	}
	if source.InputTokens != "" {
		attributes["input_tokens"] = source.InputTokens
	}
	if source.OutputTokens != "" {
		attributes["output_tokens"] = source.OutputTokens
	}
	if len(attributes) == 0 {
		attributes = nil
	}

	return metadata{
		SchemaVersion: schemaVersion,
		Generator: generator{
			Name:    generatorName,
			Version: generatorVersion,
		},
		AnalysisMode: "full",
		Source: sourceMetadata{
			Format:     "networkx-node-link",
			Directed:   source.Directed,
			Multigraph: source.Multigraph,
		},
		Counts:           counts{Nodes: nodeCount, Edges: edgeCount},
		SourceAttributes: attributes,
	}
}

func marshalDocuments(model semanticModel) (documents, error) {
	marshal := func(value any) ([]byte, error) {
		body, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, err
		}

		return append(body, '\n'), nil
	}

	nodes, err := marshal(nodeDocument{SchemaVersion: schemaVersion, Nodes: model.Nodes})
	if err != nil {
		return documents{}, err
	}
	edges, err := marshal(edgeDocument{SchemaVersion: schemaVersion, Edges: model.Edges})
	if err != nil {
		return documents{}, err
	}
	meta, err := marshal(model.Metadata)
	if err != nil {
		return documents{}, err
	}

	return documents{nodes: nodes, edges: edges, metadata: meta}, nil
}

func writeDocuments(outputDir string, docs documents) error {
	if strings.TrimSpace(outputDir) == "" {
		return errors.New("output directory が空です")
	}
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return err
	}

	writeAtomic := func(name string, body []byte) error {
		temporary, err := os.CreateTemp(outputDir, "."+name+"-*")
		if err != nil {
			return err
		}
		temporaryPath := temporary.Name()
		defer func() { _ = os.Remove(temporaryPath) }()
		if err := temporary.Chmod(0o644); err != nil {
			_ = temporary.Close()
			return err
		}
		if _, err := temporary.Write(body); err != nil {
			_ = temporary.Close()
			return err
		}
		if err := temporary.Close(); err != nil {
			return err
		}

		return os.Rename(temporaryPath, filepath.Join(outputDir, name))
	}

	if err := writeAtomic("nodes.json", docs.nodes); err != nil {
		return err
	}
	if err := writeAtomic("edges.json", docs.edges); err != nil {
		return err
	}

	return writeAtomic("metadata.json", docs.metadata)
}
