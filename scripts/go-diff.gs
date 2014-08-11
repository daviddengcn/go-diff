#!/bin/gosl

import godiff "github.com/daviddengcn/go-diff/cmd"

if len(Args) < 7 {
  Fatalf("This is supposed to be called by git")
}

godiff.Exec(Args[2], Args[5], godiff.Options{})
