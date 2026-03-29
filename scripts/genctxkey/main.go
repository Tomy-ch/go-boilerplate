package main

import (
	"flag"
	"log"
	"strings"
)

func main() {
	var name string
	var typ string
	var out string

	flag.StringVar(&name, "name", "", "context key name")
	flag.StringVar(&typ, "type", "", "value type")
	flag.StringVar(&out, "out", ".", "output directory")
	flag.Parse()

	if name == "" {
		log.Fatal("name is required")
	}

	if typ == "" {
		log.Fatal("type is required")
	}

	// enforce fully-qualified type if package path is included
	if strings.Contains(typ, "/") && !strings.Contains(typ, ".") {
		log.Fatal("type must be in format <import-path>.<Type>, e.g. github.com/foo/bar.Authn")
	}

	if err := GenerateCtxKey(name, typ, out); err != nil {
		log.Fatal(err)
	}
}
