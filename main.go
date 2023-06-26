package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"text/template"

	"github.com/hujun-open/shouchan"
)

type pktConstants struct {
	PkgName  string
	Consts   map[string]string //key is orignal name, val is the text marshalled value
	TypeName string
}

func newPktConstants() *pktConstants {
	return &pktConstants{
		Consts: make(map[string]string),
	}
}

func gen(c *pktConstants, extrenamer string, output io.Writer) error {
	const tpl = `package {{.PkgName}}

import "fmt"

func (val {{.TypeName}}) String() string {
	r,err:=val.MarshalText()
	if err!=nil {
		return fmt.Sprint(err)
	}
	return string(r)
}

func (val {{.TypeName}}) MarshalText() (text []byte, err error) {
	switch val {
	{{range $key,$val := .Consts}} 
	case {{$key}}:
		return []byte("{{marshalrename $val}}"),nil
	{{end}}
	}
	return nil, fmt.Errorf("unknown value %#v", val)
}

func (val *{{.TypeName}}) UnmarshalText(text []byte) error {
	input := string(text)
	switch input {
	{{range  $key,$val  := .Consts}} 
	case "{{unmarshalrename $val}}":
		*val={{$key}}
	{{end}}
	default:
		return fmt.Errorf("failed to parse %v into {{.TypeName}}", input)
	}
	return nil
}
	`
	mrenamer := strings.ToLower
	umrenamer := strings.ToLower
	if extrenamer != "" {
		extr, err := newExtRenamer(extrenamer)
		if err != nil {
			return err
		}
		defer extr.stop()
		mrenamer = extr.marshalRename
		umrenamer = extr.unmarshalRename
	}
	funcMap := template.FuncMap{
		// The name "title" is what the function will be called in the template text.
		"marshalrename":   mrenamer,
		"unmarshalrename": umrenamer,
	}
	t := template.Must(template.New("codegen").Funcs(funcMap).Parse(tpl))
	return t.Execute(output, c)
}

// parse src go source, return all const names with type types
func parse(src string, types string) (*pktConstants, error) {
	r := newPktConstants()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	r.PkgName = f.Name.Name
	r.TypeName = types
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		lastType := ""
	L1:
		for _, spec := range genDecl.Specs {
			valueSpec := spec.(*ast.ValueSpec)
			isType := false
			if valueSpec.Type != nil {
				if fmt.Sprint(valueSpec.Type) == types {
					isType = true
				}
				if len(valueSpec.Values) > 0 {
					if ident, ok := valueSpec.Values[0].(*ast.Ident); ok && ident.Name == "iota" {
						lastType = fmt.Sprint(valueSpec.Type)
					}
				}
			} else {
				//type is nil, meaning its type is last iota type
				if lastType == types {
					isType = true
				}

			}
			// fmt.Println(valueSpec, spec)
			if isType {
				if valueSpec.Comment != nil {
					if len(valueSpec.Comment.List) > 0 {
						//check if line comment contains alias
						comment := strings.TrimSpace(valueSpec.Comment.List[0].Text)
						if len(comment) > 2 {
							tag := reflect.StructTag(comment[2:])
							alias := tag.Get("alias")
							if alias != "" {
								r.Consts[valueSpec.Names[0].Name] = alias
								continue L1
							}
						}

					}

				}
				if len(valueSpec.Names) > 0 {
					r.Consts[valueSpec.Names[0].Name] = valueSpec.Names[0].Name
				}
			}
		}
	}
	return r, nil
}

type Conf struct {
	Input       string `alias:"s" usage:"input source file name"`
	TypeName    string `alias:"t" usage:"type name"`
	Output      string `alias:"o" usage:"output file name"`
	Transformer string `alias:"tran" usage:"transform executable"`
}

func (cnf *Conf) init() error {
	if cnf.Input == "" {
		return fmt.Errorf("input can't be empty")
	}
	if cnf.TypeName == "" {
		return fmt.Errorf("type name can't empty")
	}
	if cnf.Output == "" {
		cnf.Output = cnf.Input + ".out.go"
	}
	return nil
}

func main() {
	cnf, err := shouchan.NewSConfCMDLine(&Conf{}, "")
	if err != nil {
		panic(err)
	}
	cnf.ReadwithCMDLine()
	conf := cnf.GetConf()
	err = conf.init()
	if err != nil {
		log.Fatal(err)
	}
	buf, err := os.ReadFile(conf.Input)
	if err != nil {
		log.Fatal(err)
	}
	clist, err := parse(string(buf), conf.TypeName)
	if err != nil {
		log.Fatal(err)
	}
	ofile, err := os.Create(conf.Output)
	if err != nil {
		log.Fatal(err)
	}
	err = gen(clist, conf.Transformer, ofile)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("output is written in %v", conf.Output)
}
