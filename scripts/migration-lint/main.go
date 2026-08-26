// Package main は database/migrations の連番を検査するツール。
//
//	-check duplicate: 同じ連番を持つマイグレーションが複数ないかを調べる。
//	-check gap:       連番に欠番がないかを調べる。
//
// 検査対象は `<連番>_<名前>.<kind>.sql` の連番部分（最初の `_` より前）で、
// up / down のどちらを見るかは -kind で切り替える。
//
// このツールは lefthook の pre-commit ゲートから呼ばれる。壊れ方が
// 「何も検査しなくなる」方向に出るため、判定ロジックはシェルの中ではなく
// テストの当たる Go 側に置く。
//
// マイグレーションが 1 件も無い場合は正常終了する。サンプル API の削除
// (make setup-remove-sample-api) がマイグレーションごと消し得るため、
// 「ファイルが無い」ことをここでの失敗にはしない。
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go-boilerplate/pkg/xerrors"
)

// duplicateThreshold は、同じ連番を「重複」と数え始める出現回数。
const duplicateThreshold = 2

var (
	// errNoVersionPrefix は、マイグレーションのファイル名に連番の区切りが無いことを表す。
	errNoVersionPrefix = xerrors.New("migration file has no version prefix")
	// errUnknownCheck は、-check に未知の検査内容が渡されたことを表す。
	errUnknownCheck = xerrors.New("unknown -check (duplicate / gap)")
	// errNumberingProblem は、連番の検査で問題が見つかったことを表す。
	errNumberingProblem = xerrors.New("migration numbering check failed")
)

// main は 1:1 テスト規約の対象外で分岐を検査できないため、判断は run に置きます。
func main() {
	log.SetFlags(0)

	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("%v", err)
	}
}

// run は、指定された検査をマイグレーションの連番に対して実行します。
func run(args []string) error {
	fs := flag.NewFlagSet("migration-lint", flag.ContinueOnError)
	kind := fs.String("kind", "up", "検査するマイグレーションの種別 (up / down)")
	check := fs.String("check", "duplicate", "検査内容 (duplicate / gap)")
	dir := fs.String("dir", "database/migrations", "マイグレーションディレクトリ")

	if err := fs.Parse(args); err != nil {
		// ヘルプ要求は失敗ではないので 0 で終える。usage は flag が既に出力している。
		if xerrors.Is(err, flag.ErrHelp) {
			return nil
		}

		return xerrors.Wrap(err, "failed to parse flags")
	}

	versions, err := collectVersions(*dir, *kind)
	if err != nil {
		return err
	}

	var problem string

	switch *check {
	case "duplicate":
		problem = reportDuplicates(*kind, versions)
	case "gap":
		problem = reportGaps(*kind, versions)
	default:
		return xerrors.Wrap(errUnknownCheck, *check)
	}

	if problem == "" {
		return nil
	}

	// 報告文は欠番の一覧を含む複数行なので、エラーのメッセージへ畳み込まずそのまま出す。
	// 畳み込むと sentinel の文言が最終行の末尾に付き、並べた連番が読めなくなる。
	log.Print(problem)

	return errNumberingProblem
}

// collectVersions は、指定種別のマイグレーションファイルから連番部分を昇順で返します。
// 連番は先頭ゼロ埋めの固定幅なので、辞書順の並びは数値順と一致します。
func collectVersions(dir, kind string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*."+kind+".sql"))
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to scan "+dir)
	}

	versions := make([]string, 0, len(matches))

	for _, m := range matches {
		base := filepath.Base(m)

		version, _, found := strings.Cut(base, "_")
		if !found {
			return nil, xerrors.Wrap(errNoVersionPrefix, base)
		}

		versions = append(versions, version)
	}

	sort.Strings(versions)

	return versions, nil
}

// reportDuplicates は、重複する連番があればその報告文を返します（無ければ空文字）。
func reportDuplicates(kind string, versions []string) string {
	seen := make(map[string]int, len(versions))
	duplicates := make([]string, 0)

	for _, v := range versions {
		seen[v]++

		if seen[v] == duplicateThreshold {
			duplicates = append(duplicates, v)
		}
	}

	if len(duplicates) == 0 {
		return ""
	}

	return fmt.Sprintf("Duplicate migration numbers (%s): %s", kind, strings.Join(duplicates, " "))
}

// reportGaps は、連番に欠番があればその報告文を返します（無ければ空文字）。
// 期待値は最小値から最大値までの連番を、最小値と同じ桁数へゼロ埋めしたものです。
func reportGaps(kind string, versions []string) string {
	if len(versions) == 0 {
		return ""
	}

	expected, err := expectedSequence(versions)
	if err != nil {
		return fmt.Sprintf("Migration version is not numeric (%s): %v", kind, err)
	}

	if strings.Join(versions, "\n") == strings.Join(expected, "\n") {
		return ""
	}

	return fmt.Sprintf(
		"Migration version gap detected (%s)\nExisting :\n%s\nExpected :\n%s",
		kind, strings.Join(versions, "\n"), strings.Join(expected, "\n"),
	)
}

// expectedSequence は、昇順の連番列から「欠番の無い状態」を組み立てます。
func expectedSequence(versions []string) ([]string, error) {
	width := len(versions[0])

	first, err := strconv.Atoi(versions[0])
	if err != nil {
		return nil, xerrors.Wrap(err, "invalid version "+versions[0])
	}

	last, err := strconv.Atoi(versions[len(versions)-1])
	if err != nil {
		return nil, xerrors.Wrap(err, "invalid version "+versions[len(versions)-1])
	}

	expected := make([]string, 0, last-first+1)
	for i := first; i <= last; i++ {
		expected = append(expected, fmt.Sprintf("%0*d", width, i))
	}

	return expected, nil
}
