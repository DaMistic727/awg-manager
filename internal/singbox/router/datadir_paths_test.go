package router

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// F263: инвентарь DataDirPaths берётся из объявлений, а не пишется руками.
// Пакетная переменная, инициализированная путём в боевом каталоге данных, —
// путь, который SetDataDir обязан перенацелить; забытый уводил бы запись
// песочницы в боевой каталог (F168).
func TestDataDirPaths_CoversEveryDataDirDecl(t *testing.T) {
	const prodDataDir = "/opt/etc/awg-manager/"
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	declared := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, v := range vs.Values {
					lit, ok := v.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					if s, err := strconv.Unquote(lit.Value); err == nil && strings.HasPrefix(s, prodDataDir) {
						declared[vs.Names[i].Name] = name
					}
				}
			}
		}
	}
	if len(declared) == 0 {
		t.Fatal("ни одного объявления пути в каталоге данных — разбор сломан")
	}
	inv := DataDirPaths()
	for v, file := range declared {
		if _, ok := inv[v]; !ok {
			t.Errorf("%s (%s) объявлен в каталоге данных, но не в DataDirPaths — SetDataDir его не перенацелит", v, file)
		}
	}
}
