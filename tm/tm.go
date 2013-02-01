package tm

import(
    "github.com/daviddengcn/go-algs/ed"
)

func max(a, b int) int {
	if a > b {
		return a
	} // if

	return b
}

const(
    rune_SINGLE = iota
    rune_NUM
    rune_CAPITAL
    rune_LOWER
)

func newToken(last, cur int) bool {
    if last == rune_SINGLE || cur == rune_SINGLE {
        return true
    } // if
    
    if last == cur {
        return false
    } // if
    
    if last == rune_NUM || cur == rune_NUM {
        return true
    } // if
    
    if last == rune_LOWER && cur == rune_CAPITAL {
        return true
    } // if
    
    return false
}

func runeType(r rune) int {
    switch {
        case r >= '0' && r <= '9':
            return rune_NUM
            
        case r >= 'A' && r <= 'Z':
            return rune_CAPITAL
            
        case r >= 'a' && r <= 'z':
            return rune_LOWER
    } // if

    return rune_SINGLE
}

func LineToTokens(line string) (tokens []string) {
    lastTp := rune_SINGLE
    for _, c := range line {
        tp := runeType(c)
        if newToken(lastTp, tp) {
            tokens = append(tokens, "")
        } // if
        tokens[len(tokens) - 1] = tokens[len(tokens) - 1] + string(c)
        
        lastTp = tp
    } // for c
    
    return tokens
}

func pairOrder(tks []string) (order []int) {
    order = make([]int, len(tks))
    c0, c1, c2, c3, c4 := 0, 0, 0, 0, 0
    for i := range tks {
        switch tks[i] {
            case "[":
                c0 ++
                order[i] = c0
                
            case "{":
                c1 ++
                order[i] = c1
                
            case "(":
                c2 ++
                order[i] = c2
                
            case `"`:
                order[i] = c3
                c3 = 1-c3
                
            case "'":
                order[i] = c4
                c4 = 1-c4
        }
    } // for i
    
    c0, c1, c2 = 0, 0, 0
    
    for i := len(tks) - 1; i >= 0; i -- {
        switch tks[i] {
            case "]":
                c0 ++
                order[i] = c0
                
            case "}":
                c1 ++
                order[i] = c1
                
            case ")":
                c2 ++
                order[i] = c2
        }
    } // for i
    
    return order
}

func MatchTokens(delT, insT []string) (matA, matB []int) {
    delO, insO := pairOrder(delT), pairOrder(insT)
	_, matA, matB = ed.EditDistanceFFull(len(delT), len(insT), func(iA, iB int) int {
        if delT[iA] == insT[iB] {
            if delO[iA] == insO[iB] {
                return 0
            } else {
                return 1
            }
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
    
    return matA, matB
}

func DiffOfStrings(a, b string, mx int) int {
	if a == b {
		return 0
	} // if
    return ed.String(a, b) * mx / max(len(a), len(b))
}

func DiffOfSourceLine(a, b string, mx int) int {
	if a == b {
		return 0
	} // if
    
    delT, insT := LineToTokens(a), LineToTokens(b)
    
	diff := ed.EditDistanceF(len(delT), len(insT), func(iA, iB int) int {
        if delT[iA] == insT[iB] {
            return 0
        } // if
		return 3
	}, ed.ConstCost(1), ed.ConstCost(1))
    
    return diff * mx / (len(delT) + len(insT))
}

