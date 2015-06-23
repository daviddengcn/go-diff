package godiff

import (
	"strings"
	"testing"

	"github.com/golangplus/bytes"
	"github.com/golangplus/testing/assert"

	goassert "github.com/daviddengcn/go-assert"
)

func TestNodeToLines_Literal(t *testing.T) {
	info, err := parse("", `
package main

func main() {
	a := Da {
		A: 10,
		B: 20,
	}
}

	`)
	if !assert.NoError(t, err) {
		return
	}

	lines := info.funcs.sourceLines("")
	goassert.LinesEqual(t, "lines", lines, strings.Split(
		`func main() {
    a := Da{
        A: 10,
        B: 20,
    }
}`, "\n"))
}

func TestDiffLines_1(t *testing.T) {
	var buf bytesp.ByteSlice
	gOut = &buf

	src := strings.Split(`This a line with the word abc different only
This a line with the word def different only
This a line with the word ghi different only
This a line with the word jkl different only`, "\n")
	dst := strings.Split(`This a line with the word abc different only
This a line with the word ghi different only
This a line with the word jkl different only
This a line with the word def different only`, "\n")
	diffLines(src, dst, "%s")

	assert.Equal(t, "diff", string(buf), `    This a line with the word abc different only
--- This a line with the word def different only
    This a line with the word ghi different only
    This a line with the word jkl different only
+++ This a line with the word def different only
`)
}

func TestDiffLines_2(t *testing.T) {
	var buf bytesp.ByteSlice
	gOut = &buf

	src := strings.Split(`abc
{
  hello
}
defg`, "\n")
	dst := strings.Split(`abc
{
  gogogo
}
{
  hello
}
defg`, "\n")
	diffLines(src, dst, "%s")

	t.Logf("Diff: %s", string(buf))

	goassert.LinesEqual(t, "diff", strings.Split(string(buf), "\n"),
		strings.Split(`    abc
+++ {
+++   gogogo
+++ }
    {
        ... (2 lines)
    defg
`, "\n"))
}
