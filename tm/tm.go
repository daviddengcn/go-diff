package tm

import (
//	"fmt"
	"github.com/daviddengcn/go-algs/ed"
//	"github.com/daviddengcn/go-villa"
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
	
	const(
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

func MatchTokens(delT, insT []string) (matA, matB []int) {
	delL, delR := nearChecks(delT)
	
	_, matA, matB = ed.EditDistanceFFull(len(delT), len(insT), func(iA, iB int) int {
		if delT[iA] == insT[iB] {
			c := 0
			if delL[iA] {
				c += diffAt(delT, iA - 1, insT, iB - 1)
			}
			if delR[iA] {
				c += diffAt(delT, iA + 1, insT, iB + 1)
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

	return matA, matB
}

func DiffOfStrings(a, b string, mx int) int {
	if a == b {
		return 0
	} // if
	return ed.String(a, b) * mx / max(len(a), len(b))
}

var key_WORDS map[string]int = map[string]int {
	"if": 1}

func isKeywords(a []string) (res []bool) {
	res = make([]bool, len(a))
	for i, w := range a {
		_, res[i] = key_WORDS[w]
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
