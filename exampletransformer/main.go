package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

// This program is an example shouchangen external transformer, it
// reads lines from stdin, remove the specified prefix if it exists,
// print the result line to stdout.
// This transformer could be run with shouchangen as `shouchangen -s <src_file> -t <typename> -trans "exampletran -p <prefix>"`
func main() {
	prefix := flag.String("p", "", "prefix string")
	flag.Parse()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		line := scanner.Text()
		flist := strings.Fields(line)
		//line format is "marshal/unmarshal <str>"
		if len(flist) > 1 {
			text := flist[1]
			if *prefix != "" {
				if strings.HasPrefix(text, *prefix) {
					text = text[len(*prefix):]
				}
			}
			fmt.Println(text)
		}
	}
}
