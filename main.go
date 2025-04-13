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

	"github.com/hujun-open/cobra"
	"github.com/hujun-open/myflags/v2"
)

type pktConstants struct {
	PkgName               string
	Consts                map[string]string //key is constant name, val is the text marshalled value
	Froms                 map[string]string //val is constant name, key is the unmarhal text
	TypeName              string
	IncludeFuzzyUnmarshal bool
	FuzzyMinSimilarity    float64
}

func newPktConstants() *pktConstants {
	return &pktConstants{
		Consts: make(map[string]string),
		Froms:  make(map[string]string),
	}
}

func gen(c *pktConstants, extrenamer string, output io.Writer) error {
	const tpl = `package {{.PkgName}}

import (
	"fmt"
{{if .IncludeFuzzyUnmarshal}}
	"github.com/agext/levenshtein"
{{end}}
)

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

var unmarshalMap = map[string]{{.TypeName}} {
	{{range  $key,$val  := .Froms}} 
	"{{$key}}": {{$val}},
	{{end}}

}

func (val *{{.TypeName}}) UnmarshalText(text []byte) error {
	input := string(text)
	if r,ok:= unmarshalMap[input];ok {
	    *val = r
		return nil
	}
	{{if .IncludeFuzzyUnmarshal}}
	var maxSimilarity float64 = {{.FuzzyMinSimilarity}}
	foundKey:=""
	for key :=range unmarshalMap {
		sim:=levenshtein.Similarity(key, input, nil)
		if sim>=maxSimilarity {
			foundKey = key
			maxSimilarity = sim
		}
	}
	if foundKey!="" {
		*val = unmarshalMap[foundKey]
		return nil
	}
	{{end}}
	
	return fmt.Errorf("%s is not a valid {{.TypeName}}",input)
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

const (
	MarshalTagName   = "to"
	UnmarshalTagName = "from"
)

// parse src go source, return all const names with type types
func parse(src string, conf *Conf) (*pktConstants, error) {
	r := newPktConstants()
	r.IncludeFuzzyUnmarshal = conf.IncludeFuzzyUnmarshal
	r.FuzzyMinSimilarity = conf.FuzzyMinSimilarity
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	r.PkgName = f.Name.Name
	r.TypeName = conf.TypeName
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		lastType := ""
		// L1:
		for _, spec := range genDecl.Specs {
			valueSpec := spec.(*ast.ValueSpec)
			isType := false
			if valueSpec.Type != nil {
				if fmt.Sprint(valueSpec.Type) == conf.TypeName {
					isType = true
				}
				if len(valueSpec.Values) > 0 {
					if ident, ok := valueSpec.Values[0].(*ast.Ident); ok && ident.Name == "iota" {
						lastType = fmt.Sprint(valueSpec.Type)
					}
				}
			} else {
				//type is nil, meaning its type is last iota type
				if lastType == conf.TypeName {
					isType = true
				}

			}
			// fmt.Println(valueSpec, spec)
			if isType {
				useAliasTag := false
				useFromTag := false
				if valueSpec.Comment != nil {

					if len(valueSpec.Comment.List) > 0 {
						//check if line comment contains tags
						comment := strings.TrimSpace(valueSpec.Comment.List[0].Text)
						if len(comment) > 2 {
							tag := reflect.StructTag(comment[2:])
							alias := tag.Get(MarshalTagName)
							if alias != "" {
								r.Consts[valueSpec.Names[0].Name] = alias
								useAliasTag = true
							}
							fromtxt := tag.Get(UnmarshalTagName)
							if fromtxt != "" {
								r.Froms[fromtxt] = valueSpec.Names[0].Name
								useFromTag = true
							}
						}

					}

				}
				if len(valueSpec.Names) > 0 {
					if !useAliasTag {
						r.Consts[valueSpec.Names[0].Name] = valueSpec.Names[0].Name
					}
					if !useFromTag {
						r.Froms[valueSpec.Names[0].Name] = valueSpec.Names[0].Name
					}
				}

			}
		}
	}
	return r, nil
}

type Conf struct {
	Input                 string  `short:"s" usage:"input source file name"`
	TypeName              string  `short:"t" usage:"type name"`
	Output                string  `short:"o" usage:"output file name"`
	Transformer           string  `alias:"tran" usage:"transform executable"`
	IncludeFuzzyUnmarshal bool    `alias:"fuzzy" usage:"include fuzzy unmarshal code"`
	FuzzyMinSimilarity    float64 `alias:"minsim" usage:"minimal fuzzy similarity"`
}

func defConf() *Conf {
	return &Conf{
		FuzzyMinSimilarity: 0.6,
	}
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
	log.Print("include fuzzy", cnf.IncludeFuzzyUnmarshal)
	return nil
}

func (conf *Conf) RootCMD(cmd *cobra.Command, args []string) {
	err := conf.init()
	if err != nil {
		log.Fatal(err)
	}
	buf, err := os.ReadFile(conf.Input)
	if err != nil {
		log.Fatal(err)
	}
	clist, err := parse(string(buf), conf)
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

func main() {
	cnf := defConf()
	filler := myflags.NewFiller("shouchangen", "code generation tool for enum types",
		myflags.WithRootMethod(cnf.RootCMD))
	err := filler.Fill(cnf)
	if err != nil {
		log.Fatal(err)
	}
	err = filler.Execute()
	if err != nil {
		log.Fatal(err)
	}

}
