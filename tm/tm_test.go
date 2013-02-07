package tm

import (
	"fmt"
	"testing"
)

func TestMatchTokens1(t *testing.T) {
	delT, insT := LineToTokens("{abc{def}ghi}"), LineToTokens("{def}")
	matA, matB := MatchTokens(delT, insT)
	fmt.Println(delT, matA)
	fmt.Println(insT, matB)

	if matA[2] != 0 {
		t.Errorf("matA[2] should be 0")
	}
	if matB[0] != 2 {
		t.Errorf("matB[0] should be 2")
	}

	if matA[4] != 2 {
		t.Errorf("matA[4] should be 2")
	}
	if matB[2] != 4 {
		t.Errorf("matB[2] should be 4")
	}
}

func TestMatchTokens2(t *testing.T) {
	delT, insT := LineToTokens("(a.b())"), LineToTokens("a.b()")
	matA, matB := MatchTokens(delT, insT)
	fmt.Println(delT, matA)
	fmt.Println(insT, matB)

	if matA[4] != 3 {
		t.Errorf("matA[4] should be 3")
	}

	if matA[5] != 4 {
		t.Errorf("matA[5] should be 4")
	}

	delT, insT = LineToTokens("(a.b(), c)"), LineToTokens("(a.b, c)")
	matA, matB = MatchTokens(delT, insT)
	fmt.Println(delT, matA)
	fmt.Println(insT, matB)

	if matA[0] != 0 {
		t.Errorf("matA[0] should be 0")
	}

	if matA[9] != 7 {
		t.Errorf("matA[9] should be 7 but got %d, %v, %v", matA[9], delT, insT)
	}

	delT, insT = LineToTokens("[a.b[]]"), LineToTokens("[a.b]")
	matA, matB = MatchTokens(delT, insT)
	fmt.Println(delT, matA)
	fmt.Println(insT, matB)

	if matA[0] != 0 {
		t.Errorf("matA[0] should be 0")
	}

	if matA[6] != 4 {
		t.Errorf("matA[5] should be 4 but got %d", matA[6])
	}
	
	delT, insT = LineToTokens("(), (abc)"), LineToTokens("(abc)")
	matA, matB = MatchTokens(delT, insT)
	fmt.Println(delT, matA)
	fmt.Println(insT, matB)

	if matA[4] != 0 {
		t.Errorf("matA[4] should be 0 but got %d", matA[4])
	}

	if matA[6] != 2 {
		t.Errorf("matA[6] should be 2 but got %d", matA[6])
	}

	delT, insT = LineToTokens(`"", "abc"`), LineToTokens(`"abc"`)
	matA, matB = MatchTokens(delT, insT)
	fmt.Println(delT, matA)
	fmt.Println(insT, matB)

	if matA[4] != 0 {
		t.Errorf("matA[4] should be 0 but got %d", matA[4])
	}

	if matA[6] != 2 {
		t.Errorf("matA[6] should be 2 but got %d", matA[6])
	}
	
	delT, insT = LineToTokens(`cmd:=exec.Command("go", "abc")`), LineToTokens(`cmd:=villa.Path("go").Command("abc")`)
	matA, matB = MatchTokens(delT, insT)
	fmt.Println(delT, matA)
	fmt.Println(insT, matB)

	if matA[7] != -1 {
		t.Errorf("matA[7] should be -1 but got %d", matA[7])
	}

	if matA[12] != 14 {
		t.Errorf("matA[12] should be 14 but got %d", matA[12])
	}
}

func TestCalcDiffOfSourceLine(t *testing.T) {
	diff := CalcDiffOfSourceLine("return &monitor{", "m := &monitor{", 300)
	fmt.Println("diff", diff)
	
	diff = CalcDiffOfSourceLine("if (delO[iA] == 0) == (insO[iB] == 0) {", "if delL[iA] {", 1000)
	fmt.Println("diff", diff)
	diff = CalcDiffOfSourceLine("if (delO[iA] == 0) == (insO[iB] == 0) {", "c += diffAt(delT, iA - 1, insT, iB - 1)", 1000)
	fmt.Println("diff", diff)
}
