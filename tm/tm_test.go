package tm

import(
    "testing"
)

func TestMatchTokens(t *testing.T) {
    delT, insT := LineToTokens("{abc{def}ghi}"), LineToTokens("{def}")
    matA, matB := MatchTokens(delT, insT)
    
    if matA[0] != 0 {
        t.Errorf("matA[0] should be 0")
    }
    if matB[0] != 0 {
        t.Errorf("matB[0] should be 0")
    }
    
    if matA[len(matA) - 1] != len(insT) - 1 {
        t.Errorf("matA[len-1] should be %d", len(insT)-1)
    }
    if matB[len(matB) - 1] != len(delT) -1 {
        t.Errorf("matB[len-1] should be %d", len(delT)-1)
    }
}
