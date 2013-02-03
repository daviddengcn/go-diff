package tm

import (
	"fmt"
	"testing"
)

func TestMatchTokens1(t *testing.T) {
	delT, insT := LineToTokens("{abc{def}ghi}"), LineToTokens("{def}")
	matA, matB := MatchTokens(delT, insT)

	if matA[0] != 0 {
		t.Errorf("matA[0] should be 0")
	}
	if matB[0] != 0 {
		t.Errorf("matB[0] should be 0")
	}

	if matA[len(matA)-1] != len(insT)-1 {
		t.Errorf("matA[len-1] should be %d", len(insT)-1)
	}
	if matB[len(matB)-1] != len(delT)-1 {
		t.Errorf("matB[len-1] should be %d", len(delT)-1)
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
	if matB[3] != 4 {
		t.Errorf("matB[3] should be 4")
	}

	if matA[5] != 4 {
		t.Errorf("matA[5] should be 4")
	}
	if matB[4] != 5 {
		t.Errorf("matB[4] should be 5")
	}

	delT, insT = LineToTokens("(a.b())"), LineToTokens("(a.b)")
	matA, matB = MatchTokens(delT, insT)
	fmt.Println(delT, matA)
	fmt.Println(insT, matB)

	if matA[0] != 0 {
		t.Errorf("matA[0] should be 0")
	}
	if matB[0] != 0 {
		t.Errorf("matB[0] should be 0")
	}

	if matA[6] != 4 {
		t.Errorf("matA[5] should be 4 but got %d", matA[6])
	}
	if matB[4] != 6 {
		t.Errorf("matB[4] should be 6 but got %d", matA[4])
	}

	delT, insT = LineToTokens("[a.b[]]"), LineToTokens("[a.b]")
	matA, matB = MatchTokens(delT, insT)
	fmt.Println(delT, matA)
	fmt.Println(insT, matB)

	if matA[0] != 0 {
		t.Errorf("matA[0] should be 0")
	}
	if matB[0] != 0 {
		t.Errorf("matB[0] should be 0")
	}

	if matA[6] != 4 {
		t.Errorf("matA[5] should be 4 but got %d", matA[6])
	}
	if matB[4] != 6 {
		t.Errorf("matB[4] should be 6 but got %d", matA[4])
	}
}

func TestMatchTokens3(t *testing.T) {
	pairOrder(LineToTokens(")]}"))
}