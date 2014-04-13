package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/daviddengcn/go-algs/ed"
	"github.com/daviddengcn/go-assert"
	"github.com/daviddengcn/go-diff/tm"
	"github.com/daviddengcn/go-villa"
)

func TestDiffLine(t *testing.T) {
	delT := tm.LineToTokens("func g(src string, dst string) error {")
	insT := tm.LineToTokens("func (m *monitor) g(gsp string) error {")
	_, matA, matB := ed.EditDistanceFFull(len(delT), len(insT), func(iA, iB int) int {
		if delT[iA] == insT[iB] {
			return 0
		} // if
		return len(delT[iA]) + len(insT[iB]) + 1
	}, func(iA int) int {
		if delT[iA] == " " {
			return 0
		} // if
		return len(delT[iA])
	}, func(iB int) int {
		if insT[iB] == " " {
			return 0
		} // if
		return len(insT[iB])
	})
	ShowDelTokens(delT, matA, insT)
	ShowInsTokens(insT, matB, delT)
}

func TestGreedyMatch(t *testing.T) {
	_, _, matA, matB := GreedyMatch(2, 2, func(iA, iB int) int {
		if iA == 1 && iB == 0 {
			return 1
		} // if

		return 40
	}, ed.ConstCost(10), ed.ConstCost(10))

	if !villa.IntSlice(matA).Equals([]int{-1, 0}) {
		t.Errorf("matA should be [-1, 0]")
	} // if
	if !villa.IntSlice(matB).Equals([]int{1, -1}) {
		t.Errorf("matA should be [1, -1]")
	} // if
}

func TestExp(t *testing.T) {
	fmt.Println("OLD")
	orgInfo, err := Parse("", `
package main
func main() {
	a := (1 + 2) / 3
	fmt.Println("Access token (" + ac_FILENAME + ") not found, try authorize...")
	c = f(1, 2, 3)
}
`)
	if err != nil {
		t.Errorf("org Parse failed: %v", err)
	}

	fmt.Println("NEW")
	newInfo, err := Parse("", `
package main
func main() {
	a := (1 +
		2) / 3
	fmt.Println("Access token (" + ac_FILENAME +
		") not found, try authorize...")
	c = f(1, 2,
		3)
}
`)
	if err != nil {
		t.Errorf("new Parse failed: %v", err)
	}

	Diff(orgInfo, newInfo)
}

func TestDiffLines(t *testing.T) {
	orgLines := strings.Split(
		`
`, "\n")
	newLines := strings.Split(
		`
`, "\n")

	DiffLines(orgLines, newLines, "%s")
}

func TestBug_func_params(t *testing.T) {
	fmt.Println("====")
	p, err := Parse("", `
package main
var i, j   interface { }
func foo(i, j interface{}) {
}
`)
	if !assert.NoError(t, err) {
		return
	}

	assert.LinesEqual(t, "func foo", p.funcs.Parts[0].sourceLines(""), []string{
		"func foo(i interface{}, j interface{}) {",
		"}",
	})
	assert.LinesEqual(t, "var", p.vars.Parts[0].sourceLines(""), []string{
		"var i, j interface{}",
	})
}

func TestBug_Match(t *testing.T) {
	fmt.Println("====")
	orgLines := strings.Split(
`import org.apache.hadoop.hbase.regionserver.HRegion;
import org.apache.hadoop.hbase.util.Bytes;
import org.junit.After;`, "\n")
	newLines := strings.Split(
`import org.apache.hadoop.hbase.regionserver.HRegion;
import org.apache.hadoop.hbase.util.Bytes;
import org.apache.hadoop.hbase.util.StringBytes;
import org.junit.After;`, "\n")

	cnt := DiffLines(orgLines, newLines, "%s")
	assert.Equals(t, "Number of different lines", cnt , 1)
}

type OutputSaver struct {
	deleted []string
	inserted []string
}

func (os *OutputSaver) outputIns(line string) {
	os.inserted = append(os.inserted, line)
	fmt.Println("INSERT", line)
}

func (os *OutputSaver) outputDel(line string) {
	os.deleted = append(os.deleted, line)
	fmt.Println("DELETE", line)
}

func (os *OutputSaver) outputSame(line string) {
}

func (os *OutputSaver) outputChange(del, ins string) {
	os.outputDel(del)
	os.outputIns(ins)
	fmt.Println(del, "->", ins)
}

func (os *OutputSaver) end() {
}

func TestTrimDiff(t *testing.T) {
	orgLines := strings.Split(
` {
}`, "\n")
	newLines := strings.Split(
` {
	}
}`, "\n")
	var os OutputSaver

	cnt := DiffLinesTo(orgLines, newLines, "%s", &os)
	assert.Equals(t, "Number of different lines", cnt , 2)
	
	assert.LinesEqual(t, "inserted", os.inserted, []string{"	}"})
	assert.LinesEqual(t, "deleted", os.deleted, []string{})
}
