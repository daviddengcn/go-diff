/*
	go-diff is a tool checking semantic difference between source files.

	Currently supported language:
		Go fully

	If the language is not supported or parsing is failed for either file,
	a line-to-line comparing is imposed.
*/
package main

import (
	"flag"
	"fmt"

	"github.com/daviddengcn/go-diff/cmd"
)

func main() {
	var options godiff.Options

	flag.BoolVar(&options.NoColor, "no-color", false, "turn off the colors")

	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Println("Please specify the new/original files.")
		return
	} // if
	orgFn := flag.Arg(0)
	newFn := flag.Arg(1)

	godiff.Exec(orgFn, newFn, options)
}
