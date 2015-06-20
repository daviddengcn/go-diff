package tm

import (
	"fmt"
	"testing"

	"github.com/golangplus/testing/assert"
)

func TestMatchTokens1(t *testing.T) {
	delT, insT := LineToTokens("{abc{def}ghi}"), LineToTokens("{def}")
	assert.StringEqual(t, "delT", delT, "[{ abc { def } ghi }]")
	assert.StringEqual(t, "insT", insT, "[{ def }]")

	matA, matB := MatchTokens(delT, insT)
	assert.StringEqual(t, "matA", matA, "[-1 -1 0 1 2 -1 -1]")
	assert.StringEqual(t, "matB", matB, "[2 3 4]")
}

func TestMatchTokens2(t *testing.T) {
	delT, insT := LineToTokens("(a.b())"), LineToTokens("a.b()")
	assert.StringEqual(t, "delT", delT, "[( a . b ( ) )]")
	assert.StringEqual(t, "insT", insT, "[a . b ( )]")
	matA, matB := MatchTokens(delT, insT)
	assert.StringEqual(t, "matA", matA, "[-1 0 1 2 3 4 -1]")
	assert.StringEqual(t, "matB", matB, "[1 2 3 4 5]")

	delT, insT = LineToTokens("(a.b(), c)"), LineToTokens("(a.b, c)")
	assert.StringEqual(t, "delT", delT, "[( a . b ( ) ,   c )]")
	assert.StringEqual(t, "insT", insT, "[( a . b ,   c )]")
	matA, matB = MatchTokens(delT, insT)
	assert.StringEqual(t, "matA", matA, "[0 1 2 3 -1 -1 4 -1 6 7]")
	assert.StringEqual(t, "matB", matB, "[0 1 2 3 6 -1 8 9]")

	delT, insT = LineToTokens("[a.b[]]"), LineToTokens("[a.b]")
	assert.StringEqual(t, "delT", delT, "[[ a . b [ ] ]]")
	assert.StringEqual(t, "insT", insT, "[[ a . b ]]")
	matA, matB = MatchTokens(delT, insT)
	assert.StringEqual(t, "matA", matA, "[0 1 2 3 -1 -1 4]")
	assert.StringEqual(t, "matB", matB, "[0 1 2 3 6]")

	delT, insT = LineToTokens("(), (abc)"), LineToTokens("(abc)")
	assert.StringEqual(t, "delT", delT, "[( ) ,   ( abc )]")
	assert.StringEqual(t, "insT", insT, "[( abc )]")
	matA, matB = MatchTokens(delT, insT)
	assert.StringEqual(t, "matA", matA, "[-1 -1 -1 -1 0 1 2]")
	assert.StringEqual(t, "matB", matB, "[4 5 6]")

	delT, insT = LineToTokens(`"", "abc"`), LineToTokens(`"abc"`)
	assert.StringEqual(t, "delT", delT, `[" " ,   " abc "]`)
	assert.StringEqual(t, "insT", insT, `[" abc "]`)
	matA, matB = MatchTokens(delT, insT)
	assert.StringEqual(t, "matA", matA, "[-1 -1 -1 -1 0 1 2]")
	assert.StringEqual(t, "matB", matB, "[4 5 6]")

	delT, insT = LineToTokens(`cmd:=exec.Command("go", "abc")`), LineToTokens(`cmd:=villa.Path("go").Command("abc")`)
	assert.StringEqual(t, "delT", delT, `[cmd : = exec . Command ( " go " ,   " abc " )]`)
	assert.StringEqual(t, "insT", insT, `[cmd : = villa . Path ( " go " ) . Command ( " abc " )]`)
	matA, matB = MatchTokens(delT, insT)
	assert.StringEqual(t, "matA", matA, "[0 1 2 -1 11 12 13 -1 -1 -1 -1 -1 14 15 16 17]")
	assert.StringEqual(t, "matB", matB, "[0 1 2 -1 -1 -1 -1 -1 -1 -1 -1 4 5 6 12 13 14 15]")

	delT, insT = LineToTokens("Rename(m.exeFile(gsp))"), LineToTokens("Rename(exeFile)")
	assert.StringEqual(t, "delT", delT, `[Rename ( m . exe File ( gsp ) )]`)
	assert.StringEqual(t, "insT", insT, `[Rename ( exe File )]`)
	matA, matB = MatchTokens(delT, insT)
	assert.StringEqual(t, "matA", matA, "[0 1 -1 -1 2 3 -1 -1 -1 4]")
	assert.StringEqual(t, "matB", matB, "[0 1 4 5 9]")

	delT, insT = LineToTokens("Rename(exeFile)"), LineToTokens("Rename(m.exeFile(gsp))")
	assert.StringEqual(t, "delT", delT, `[Rename ( exe File )]`)
	assert.StringEqual(t, "insT", insT, `[Rename ( m . exe File ( gsp ) )]`)
	matA, matB = MatchTokens(delT, insT)
	assert.StringEqual(t, "matA", matA, "[0 1 4 5 9]")
	assert.StringEqual(t, "matB", matB, "[0 1 -1 -1 2 3 -1 -1 -1 4]")

	delT, insT = LineToTokens("Rename[m.exeFile[gsp]]"), LineToTokens("Rename[exeFile]")
	assert.StringEqual(t, "delT", delT, `[Rename [ m . exe File [ gsp ] ]]`)
	assert.StringEqual(t, "insT", insT, `[Rename [ exe File ]]`)
	matA, matB = MatchTokens(delT, insT)
	assert.StringEqual(t, "matA", matA, "[0 1 -1 -1 2 3 -1 -1 -1 4]")
	assert.StringEqual(t, "matB", matB, "[0 1 4 5 9]")
}

func TestMatchTokens3(t *testing.T) {
	MatchTokens(LineToTokens("[({gsp}}))]]"), LineToTokens("])}"))
}

func TestCalcDiffOfSourceLine(t *testing.T) {
	diff := CalcDiffOfSourceLine("return &monitor{", "m := &monitor{", 300)
	fmt.Println("diff", diff)

	diff = CalcDiffOfSourceLine("if (delO[iA] == 0) == (insO[iB] == 0) {", "if delL[iA] {", 1000)
	fmt.Println("diff", diff)
	diff = CalcDiffOfSourceLine("if (delO[iA] == 0) == (insO[iB] == 0) {", "c += diffAt(delT, iA - 1, insT, iB - 1)", 1000)
	fmt.Println("diff", diff)
}
