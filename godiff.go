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
	"os"

	"github.com/daviddengcn/go-diff/cmd"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: go-diff [options] org-filename new-filename\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	var options godiff.Options

	flag.BoolVar(&options.NoColor, "no-color", false, "turn off the colors")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 2 {
		usage()
		return
	} // if
	orgFn := flag.Arg(0)
	newFn := flag.Arg(1)

	godiff.Exec(orgFn, newFn, options)
}
