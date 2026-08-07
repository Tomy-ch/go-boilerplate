package main

import (
	"flag"
	"log"
	"os"

	"go-boilerplate/pkg/xerrors"
)

var (
	// errNameRequired は、-name が指定されなかったことを表す。
	errNameRequired = xerrors.New("name is required")
	// errTypeRequired は、-type が指定されなかったことを表す。
	errTypeRequired = xerrors.New("type is required")
)

// main はエラーを終了コードへ変換するだけに留め、判断は run が持ちます。
// main は 1:1 の対象外でテストを書けないため、ここに分岐を置くと検査されない
// コードがそのぶん増える。
func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

// run は、フラグを解釈して context key の生成を実行します。
func run(args []string) error {
	fs := flag.NewFlagSet("genctxkey", flag.ContinueOnError)
	name := fs.String("name", "", "context key name")
	typ := fs.String("type", "", "value type")
	importPath := fs.String("import", "", "import path (optional)")
	importAlias := fs.String("alias", "", "import alias (optional)")
	out := fs.String("out", ".", "output directory")
	testValue := fs.String("test-value", "", "test success value expression for non-primitive types (optional)")

	if err := fs.Parse(args); err != nil {
		// ヘルプ要求は失敗ではないので 0 で終える。usage は flag が既に出力している。
		if xerrors.Is(err, flag.ErrHelp) {
			return nil
		}

		return xerrors.Wrap(err, "failed to parse flags")
	}

	// 生成器も未指定を拒むが、そちらは name / type をまとめた 1 つのエラーになる。
	// どちらのフラグが足りないかを呼び出し側へ返すため、ここで個別に見る。
	if *name == "" {
		return errNameRequired
	}

	if *typ == "" {
		return errTypeRequired
	}

	return GenerateCtxKey(*name, *typ, *importPath, *importAlias, *out, *testValue)
}
