package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func inventory(root string) map[string]string {
	result := map[string]string{}
	fset := token.NewFileSet()
	print := func(node interface{}) string { var b bytes.Buffer; printer.Fprint(&b, fset, node); return b.String() }
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "examples" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			panic(err)
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		key := func(name string) string { return filepath.ToSlash(rel) + "." + name }
		fields := func(name string, list *ast.FieldList) {
			for _, field := range list.List {
				if len(field.Names) == 0 {
					result[key(name+".embedded."+print(field.Type))] = print(field.Type)
				}
				for _, n := range field.Names {
					if n.IsExported() {
						tag := ""
						if field.Tag != nil {
							tag = field.Tag.Value
						}
						result[key(name+"."+n.Name)] = print(field.Type) + " " + tag
					}
				}
			}
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name.IsExported() {
					name := d.Name.Name
					if d.Recv != nil {
						name = print(d.Recv.List[0].Type) + "." + name
					}
					result[key(name)] = print(d.Type)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if !s.Name.IsExported() {
							continue
						}
						switch typ := s.Type.(type) {
						case *ast.StructType:
							fields(s.Name.Name, typ.Fields)
							result[key(s.Name.Name)] = "struct"
						case *ast.InterfaceType:
							fields(s.Name.Name, typ.Methods)
							result[key(s.Name.Name)] = "interface"
						default:
							result[key(s.Name.Name)] = print(s.Type)
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if !name.IsExported() {
								continue
							}
							if st, ok := s.Type.(*ast.StructType); ok {
								fields(name.Name, st.Fields)
								result[key(name.Name)] = "struct variable"
							} else {
								result[key(name.Name)] = print(s.Type)
							}
						}
					}
				}
			}
		}
		return nil
	})
	return result
}
func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: check-public-api BASELINE_DIRECTORY CURRENT_DIRECTORY")
		os.Exit(2)
	}
	old, updated := inventory(os.Args[1]), inventory(os.Args[2])
	missing := map[string]string{}
	changed := map[string][]string{}
	added := 0
	for key, value := range old {
		v, ok := updated[key]
		if !ok {
			missing[key] = value
		} else if v != value {
			changed[key] = []string{value, v}
		}
	}
	for k := range updated {
		if _, ok := old[k]; !ok {
			added++
		}
	}
	reviewed := map[string][]string{}
	for _, key := range []string{"dto.MessageToCreate.MsgType", "dto.RichMediaMessage.SrvSendMsg"} {
		if values, ok := changed[key]; ok && strings.Replace(values[0], ",omitempty", "", 1) == values[1] {
			reviewed[key] = values
			delete(changed, key)
		}
	}
	summary := map[string]interface{}{"baseline": "de69850", "checked_symbols_and_fields": len(old), "added_symbols_and_fields": added, "removed": missing, "incompatible_signatures_or_tags": changed, "reviewed_wire_corrections": reviewed}
	data, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(data))
	if len(missing) > 0 || len(changed) > 0 {
		os.Exit(1)
	}
}
