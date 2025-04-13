# shouchangen
This is a code generation tool to generate following methods for the specified type, based on the constants of the specified type in a Go source file.

- String() string
- MarshalText() (text []byte, err error)
- UnmarshalText(text []byte) error


The target use case when you have large amount of constants of a given type, you don't need to write these methods manually. 

The generated unmarshal function could optionally include fuzzy match so that it could accommodate input that includes typo, it uses the `github.com/agext/levenshtein` for the fuzzy match.

By default, the marshall/unmarshal text is the constant name, it could be overridden by having a line comment with format: `to:"<new_marshal_text>" from:"<new_unmarhsal_text>"`.

Optionally a external transformer could be specified for custom constant name transform logic. 


## Example

1. input golang source code (input.go):
```
package color

type Color int

const (
	ColorRed  Color = iota //to:"red" from:"hongse"
	ColorBlue              //to:"lanse"
	ColorYellow
)
```
2. generate code via command, include fuzzy: `shouchangen -s testdata/example.go -t Color --fuzzy`
```
package color

import (
	"fmt"

	"github.com/agext/levenshtein"

)

func (val Color) String() string {
	r,err:=val.MarshalText()
	if err!=nil {
		return fmt.Sprint(err)
	}
	return string(r)
}

func (val Color) MarshalText() (text []byte, err error) {
	switch val {
	 
	case ColorBlue:
		return []byte("lanse"),nil
	 
	case ColorRed:
		return []byte("red"),nil
	 
	case ColorYellow:
		return []byte("coloryellow"),nil
	
	}
	return nil, fmt.Errorf("unknown value %#v", val)
}

var unmarshalMap = map[string]Color {
	 
	"ColorBlue": ColorBlue,
	 
	"ColorYellow": ColorYellow,
	 
	"hongse": ColorRed,
	

}

func (val *Color) UnmarshalText(text []byte) error {
	input := string(text)
	if r,ok:= unmarshalMap[input];ok {
	    *val = r
		return nil
	}
	
	var maxSimilarity float64 = 0.6
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
	
	
	return fmt.Errorf("%s is not a valid Color",input)
}
	
		
```
## External Transformer
In case of custom transform logic is need in marshalling or unmarshalling, shouchangen could run an external transformer that does following:
- for each constant name, shouchangen send a line of "marshal/unmarhsal <constant_name>" to its stdin
	- in case alias is used, then `<constant_name>` is the alias
- the transformer return the transformed text in its stdout, which will be used by shouchangen for marshalling or unmarshalling accordingly

the external transformer could be specified by using `-tran <transformer_command_with_args>`

see [exampletransformer](./exampletransformer/) folder for a example implementation. 

