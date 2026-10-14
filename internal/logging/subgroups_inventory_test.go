package logging

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// F664: запись с подгруппой вне KnownSubgroups своей группы не выбирается
// чип-фильтром журнала. Страж сканирует все вызовы NewScopedLogger в дереве
// и сверяет пару (группа, подгруппа) с каталогом. Аргументы — константы
// logging.Group*/Sub* или строковые литералы; иное (переменная) — отказ:
// сверять нечем.
func TestScopedLoggerSubgroupsInCatalog(t *testing.T) {
	consts := map[string]string{}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "types.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, sp := range gd.Specs {
			vs := sp.(*ast.ValueSpec)
			for i, v := range vs.Values {
				if lit, ok := v.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					consts[vs.Names[i].Name], _ = strconv.Unquote(lit.Value)
				}
			}
		}
	}
	resolve := func(e ast.Expr) (string, bool) {
		switch x := e.(type) {
		case *ast.BasicLit:
			s, err := strconv.Unquote(x.Value)
			return s, err == nil
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok && id.Name == "logging" {
				v, ok := consts[x.Sel.Name]
				return v, ok
			}
		case *ast.Ident:
			v, ok := consts[x.Name]
			return v, ok
		}
		return "", false
	}
	calls := 0
	for _, root := range []string{"../../internal", "../../cmd"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) != 3 {
					return true
				}
				name := ""
				switch fn := call.Fun.(type) {
				case *ast.SelectorExpr:
					name = fn.Sel.Name
				case *ast.Ident:
					name = fn.Name
				}
				if name != "NewScopedLogger" {
					return true
				}
				calls++
				g, okG := resolve(call.Args[1])
				sg, okS := resolve(call.Args[2])
				pos := fset.Position(call.Pos())
				if !okG || !okS {
					t.Errorf("%s: группа/подгруппа не константа и не литерал — страж не может сверить", pos)
					return true
				}
				if !slices.Contains(KnownSubgroups[g], sg) {
					t.Errorf("%s: подгруппа %q вне KnownSubgroups[%q]", pos, sg, g)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls < 50 {
		t.Fatalf("вызовов NewScopedLogger найдено %d — разбор сломан", calls)
	}
}
