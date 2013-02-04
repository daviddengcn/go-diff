package main

import (
	"github.com/daviddengcn/go-algs/ed"
	"github.com/daviddengcn/go-villa"
	"testing"
	//    "fmt"
	"github.com/daviddengcn/go-diff/tm"
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
