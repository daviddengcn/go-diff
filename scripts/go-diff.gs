#!/usr/bin/env gosl
# gosl: https://github.com/daviddengcn/gosl

import godiff "github.com/daviddengcn/go-diff/cmd"

if len(Args) < 7 {
	Fatalf("go-diff.gs is supposed to be called from Git")
}

godiff.Exec(Args[2], Args[5], godiff.Options{})
