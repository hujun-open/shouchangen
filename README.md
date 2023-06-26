# shouchangen
This is a code generation tool to generate following methods for the specified type, based on the constants of the specified type in a Go source file.

- String() string
- MarshalText() (text []byte, err error)
- UnmarshalText(text []byte) error


The target use case when you have large amount of constants of a given type, you don't need to write these methods manually. 

By default, the marshalled text is the lower case of constant name, it could be overridden by having a line comment with format: `//alias:"<new_name>"`.

Optionally a external transformer could be specified for custom constant name transform logic. 


## Example

1. input golang source code (input.go):
```
package color

type Color int

const (
	ColorRed Color = iota //alias:"red"
	ColorBlue
	ColorYellow
)
```

2. run `shouchangen -s input.go -t Color -o output.go`

3. it generates output.go:
```
package color

import "fmt"

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
		return []byte("colorblue"),nil
	 
	case ColorRed:
		return []byte("red"),nil
	 
	case ColorYellow:
		return []byte("coloryellow"),nil
	
	}
	return nil, fmt.Errorf("unknown value %#v", val)
}

func (val *Color) UnmarshalText(text []byte) error {
	input := string(text)
	switch input {
	 
	case "colorblue":
		*val=ColorBlue
	 
	case "red":
		*val=ColorRed
	 
	case "coloryellow":
		*val=ColorYellow
	
	default:
		return fmt.Errorf("failed to parse %v into Color", input)
	}
	return nil
}
		
```
## External Transformer
In case of custom transform logic is need in marshalling or unmarshalling, shouchangen could run an external transformer that does following:
- for each constant name, shouchangen send a line of "marshal/unmarhsal <constant_name>" to its stdin
	- in case alias is used, then `<constant_name>` is the alias
- the transformer return the transformed text in its stdout, which will be used by shouchangen for marshalling or unmarshalling accordingly

the external transformer could be specified by using `-tran <transformer_command_with_args>`

see [exampletransformer](./exampletransformer/) folder for a example implementation. 

