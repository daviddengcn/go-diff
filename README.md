go-diff
=======

A diff tool for go languange. It can show the semantic difference between two go source files.

The following differences will be ignored:
 1. Order of <code>import</code> statements
 2. Order of definitions of global <code>type</code>/<code>const</code>/<code>var</code>/<code>func</code>
 3. Whether more than one parameters or global variables are declared in one line. e.g. <code>var a, b int = 1, 2</code> is equivalent to <code>var a int = 1; var  b int = 2</code>. (NOTE parallel assignments are not normalized)
 4. All comments
 5. Code formats. e.g. some useless new lines.

Installation
------------
```bash
$ go get -u github.com/daviddengcn/go-diff
$ go install github.com/daviddengcn/go-diff
$ go-diff <new-file> <org-file>
```

Using as git diff
```bash
$ set GIT_EXTERNAL_DIFF="go-diff"
$ git diff
```
