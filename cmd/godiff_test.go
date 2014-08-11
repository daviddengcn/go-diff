package godiff

import (
	"strings"
	"testing"
	
	"github.com/daviddengcn/go-assert"
)

func TestNodeToLines_Literal(t *testing.T) {
	info, err := Parse("", `
package main

func main() {
	a := Da {
		A: 10,
		B: 20,
	}
}

	`,)
	if !assert.NoError(t, err) {
		return
	}
	
	lines := info.funcs.sourceLines("")
	assert.LinesEqual(t, "lines", lines, strings.Split(
`func main() {
    a := Da{
        A: 10,
        B: 20,
    }
}`, "\n"))
}