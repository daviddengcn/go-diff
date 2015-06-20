package tm

import (
	"github.com/daviddengcn/go-algs/ed"
	"github.com/daviddengcn/go-villa"
	"github.com/golangplus/math"
)

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
		}
		tokens[len(tokens)-1] = tokens[len(tokens)-1] + string(c)

		lastTp = tp
	}

	return tokens
}

func diffAt(a []string, iA int, b []string, iB int) int {
	if iA < 0 {
		if iB < 0 {
			return 0
		} else {
			return 1
		}
	}
	if iB < 0 {
		return 1
	}

	if iA >= len(a) {
		if iB >= len(b) {
			return 0
		} else {
			return 1
		}
	}
	if iB >= len(b) {
		return 1
	}

	if a[iA] == b[iB] {
		return 0
	}

	return 2
}

func nearChecks(a []string) (l, r []bool) {
	l, r = make([]bool, len(a)), make([]bool, len(a))

	const (
		normal = iota
		doubleQuoted
		singleQuoted
	)

	status := normal
	escaped := false
	for i, el := range a {
		l[i], r[i] = true, true

		switch status {
		case normal:
			switch el {
			case `"`:
				status = doubleQuoted
				l[i] = false
			case "'":
				status = singleQuoted
				l[i] = false
			case ",", ")":
				r[i] = false
			}
		case doubleQuoted:
			switch el {
			case `\`:
				if !escaped {
					escaped = true
					continue
				}
			case `"`:
				if !escaped {
					status = normal
					r[i] = false
				}
			}
			escaped = false
		case singleQuoted:
			switch el {
			case `\`:
				if !escaped {
					escaped = true
					continue
				}
			case "'":
				if !escaped {
					status = normal
					r[i] = false
				}
			}
			escaped = false
		} // switch status
	}

	return l, r
}

// if tks[i] and tks[j] are pairs, pairs[i], pairs[j] = j, i
func findPairs(tks []string) (pairs []int) {
	pairs = make([]int, len(tks))
	var s0, s1, s2 villa.IntSlice
	for i, tk := range tks {
		pairs[i] = -1

		switch tk {
		case "(":
			s0.Add(i)
		case ")":
			if len(s0) > 0 {
				j := s0.Pop()
				pairs[i], pairs[j] = j, i
			}

		case "[":
			s1.Add(i)
		case "]":
			if len(s1) > 0 {
				j := s1.Pop()
				pairs[i], pairs[j] = j, i
			} // if

		case "{":
			s2.Add(i)
		case "}":
			if len(s2) > 0 {
				j := s2.Pop()
				pairs[i], pairs[j] = j, i
			} // if
		}
	}

	return pairs
}

func noMatchBetween(mat []int, p1, p2 int) bool {
	if p2 < p1 {
		p1, p2 = p2, p1
	}
	for i := p1 + 1; i < p2; i++ {
		if mat[i] >= 0 {
			return false
		}
	}

	return true
}

/*
if one of the pairs is match, and the other is not. Some adjustment can be performed.
*/
func alignPairs(matA, matB, pairA, pairB []int) {
	for {
		changed := false
		/*
			A   i <---> j   m
			    ^          7
			    |         /
			    v        L
			B   k <---> l
		*/
		for i := range matA {
			j, k := pairA[i], matA[i]
			if j >= 0 && k >= 0 && matA[j] < 0 {
				l := pairB[k]
				if l >= 0 && matB[l] >= 0 {
					m := matB[l]
					if noMatchBetween(matA, j, m) {
						matA[j], matA[m], matB[l] = l, -1, j
						changed = true
					}
				}
			}
		}

		/*
			B   i <---> j   m
			    ^          7
			    |         /
			    v        L
			A   k <---> l
		*/
		for i := range matB {
			j, k := pairB[i], matB[i]
			if j >= 0 && k >= 0 && matB[j] < 0 {
				l := pairA[k]
				if l >= 0 && matA[l] >= 0 {
					m := matA[l]
					if noMatchBetween(matB, j, m) {
						matB[j], matB[m], matA[l] = l, -1, j
						changed = true
					}
				}
			}
		}

		if !changed {
			break
		}
	}
}

func MatchTokens(delT, insT []string) (matA, matB []int) {
	delL, delR := nearChecks(delT)

	_, matA, matB = ed.EditDistanceFFull(len(delT), len(insT), func(iA, iB int) int {
		if delT[iA] == insT[iB] {
			c := 0
			if delL[iA] {
				c += diffAt(delT, iA-1, insT, iB-1)
			}
			if delR[iA] {
				c += diffAt(delT, iA+1, insT, iB+1)
			}
			return c
		} // if
		return len(delT[iA]) + len(insT[iB]) + 5
	}, func(iA int) int {
		if delT[iA] == " " {
			return 0
		} // if

		return len(delT[iA]) + 2
	}, func(iB int) int {
		if insT[iB] == " " {
			return 0
		} // if

		return len(insT[iB]) + 2
	})

	delP, insP := findPairs(delT), findPairs(insT)
	alignPairs(matA, matB, delP, insP)

	return matA, matB
}

func DiffOfStrings(a, b string, mx int) int {
	if a == b {
		return 0
	} // if
	return ed.String(a, b) * mx / mathp.MaxI(len(a), len(b))
}

var key_WORDS villa.StrSet = villa.NewStrSet(
	"if", "for", "return", "switch", "case", "select", "go")

func isKeywords(a []string) (res []bool) {
	res = make([]bool, len(a))
	for i, w := range a {
		res[i] = key_WORDS.In(w)
	}

	return res
}

func CalcDiffOfSourceLine(a, b string, mx int) int {
	if a == b {
		return 0
	} // if

	delT, insT := LineToTokens(a), LineToTokens(b)
	delK, insK := isKeywords(delT), isKeywords(insT)

	diff := ed.EditDistanceF(len(delT), len(insT), func(iA, iB int) int {
		if delT[iA] == insT[iB] {
			return 0
		} // if
		return 50
	}, func(iA int) int {
		if delK[iA] {
			return 2
		}
		return 1
	}, func(iB int) int {
		if insK[iB] {
			return 2
		}

		return 1
	})

	return diff * mx / (len(delT) + len(insT))
}
