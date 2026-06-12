package main

import (
	"flag"
	"log"
)

func main() {
	var name string
	var typ string
	var importPath string
	var importAlias string
	var out string
	var testValue string

	flag.StringVar(&name, "name", "", "context key name")
	flag.StringVar(&typ, "type", "", "value type")
	flag.StringVar(&importPath, "import", "", "import path (optional)")
	flag.StringVar(&importAlias, "alias", "", "import alias (optional)")
	flag.StringVar(&out, "out", ".", "output directory")
	flag.StringVar(&testValue, "test-value", "", "test success value expression for non-primitive types (optional)")
	flag.Parse()

	if name == "" {
		log.Fatal("name is required")
	}

	if typ == "" {
		log.Fatal("type is required")
	}

	if err := GenerateCtxKey(name, typ, importPath, importAlias, out, testValue); err != nil {
		log.Fatal(err)
	}
}
