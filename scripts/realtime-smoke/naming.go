package main

import (
	"encoding/hex"
	"io"
	"strconv"

	"go-boilerplate/pkg/xerrors"
)

// runIDBytes は、実行 ID の乱数長です。16 進 12 桁になり、共有インフラ上で並行する smoke 同士が
// table / topic / queue 名で衝突しない程度に十分です。
const runIDBytes = 6

// names は、1 回の実行が作る resource の名前です。DynamoDB の table 名は小文字（設計正本）、
// SNS / SQS は `-` 区切りにします。
type names struct {
	runID string
}

// newRunID は、実行ごとの一意な ID を返します。
func newRunID(random io.Reader) (string, error) {
	b := make([]byte, runIDBytes)
	if _, err := io.ReadFull(random, b); err != nil {
		return "", xerrors.Wrap(err, "read random")
	}

	return hex.EncodeToString(b), nil
}

func (n names) table() string { return "gobp_smoke_" + n.runID }

func (n names) topic() string { return "gobp-smoke-" + n.runID }

func (n names) queue(i int) string { return "gobp-smoke-" + n.runID + "-" + strconv.Itoa(i) }

func (n names) dlq() string { return "gobp-smoke-" + n.runID + "-dlq" }
