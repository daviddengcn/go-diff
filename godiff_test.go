package main

import(
    "testing"
    "github.com/daviddengcn/go-algs/ed"
    "fmt"
)

func TestDiffLine(t *testing.T) {
    delT := lineToTokens("func g(src string, dst string) error {")
    fmt.Println(delT)
    insT := lineToTokens("func (m *monitor) g(gsp string) error {")
    fmt.Println(insT)
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
    ShowDelTokens(delT, matA)
    ShowInsTokens(insT, matB)
}

func TestGreedyMatch(t *testing.T) {
}