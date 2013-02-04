package tm

import (
//	"fmt"
	"github.com/daviddengcn/go-algs/ed"
	"github.com/daviddengcn/go-villa"
)

func max(a, b int) int {
	if a > b {
		return a
	} // if

	return b
}

const (
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
		tokens[len(tokens)-1] = tokens[len(tokens)-1] + string(c)

		lastTp = tp
	} // for c

	return tokens
}

func pairOrder(tks []string) (order []int) {
	order = make([]int, len(tks))
	var s0, s1, s2 villa.IntSlice
	q0 := -1
	escaped := false
	for i := range tks {
		switch tks[i] {
		case "(":
			s0.Add(i)
		case ")":
			if len(s0) > 0 {
				order[s0.Remove(len(s0) - 1)] = i
				order[i] = -1
			}

		case "{":
			s1.Add(i)
		case "}":
			if len(s1) > 0 {
				order[s1.Remove(len(s1) - 1)] = i
				order[i] = -1
			}
		
		case "[":
			s2.Add(i)
		case "]":
			if len(s2) > 0 {
				order[s2.Remove(len(s2) - 1)] = i
				order[i] = -1
			}
			
		case `\`:
			if q0 >= 0 && !escaped {
				escaped = true
				continue
			}
			
		case `"`:
			if !escaped {
				if q0 < 0 {
					q0 = i
				} else {
					order[q0] = i
					order[i] = -1
					q0 = -1
				}
			}
		}
		
		if escaped {
			escaped = false
		}
	} // for i
//fmt.Println("order", order)
	return order
}

func MatchTokens(delT, insT []string) (matA, matB []int) {
	delO, insO := pairOrder(delT), pairOrder(insT)
	_, matA, matB = ed.EditDistanceFFull(len(delT), len(insT), func(iA, iB int) int {
		if delT[iA] == insT[iB] {
			if (delO[iA] == 0) == (insO[iB] == 0) {
				return 0
			} else {
				return 1
			}
		} // if
		return len(delT[iA]) + len(insT[iB]) + 1
	}, func(iA int) int {
		if delT[iA] == " " || delO[iA] < 0 {
			return 0
		} // if
		return len(delT[iA])
	}, func(iB int) int {
		if insT[iB] == " " || insO[iB] < 0 {
			return 0
		} // if
		return len(insT[iB])
	})

	for i := range matA {
		j := delO[i]
		if j > 0 {
			k := matA[i]
			if k < 0 {
				matA[j] = -1
			} else {
				l := insO[k]
				if l > 0 {
					matA[j] = l
				} else {
					matA[j] = -1
				}
			}
		}
	}

	for k := range matB {
		l := insO[k]
		if l > 0 {
			i := matB[k]
			if i < 0 {
				matB[l] = -1
			} else {
				j := delO[i]
				if j > 0 {
					matB[l] = j
				} else {
					matB[j] = -1
				}
			}
		}
	}

	return matA, matB
}

func DiffOfStrings(a, b string, mx int) int {
	if a == b {
		return 0
	} // if
	return ed.String(a, b) * mx / max(len(a), len(b))
}

func CalcDiffOfSourceLine(a, b string, mx int) int {
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
