go-diff [![GoSearch](http://go-search.org/badge?id=github.com%2Fdaviddengcn%2Fgo-diff)](http://go-search.org/view?id=github.com%2Fdaviddengcn%2Fgo-diff)
=======

A diff tool for [Go languange](http://golang.org/). It shows the semantic differences between two Go source files.

Ignored Difference
------------------
 1. Order of <code>import</code> statements
 1. Order of definitions of global <code>type</code>/<code>const</code>/<code>var</code>/<code>func</code>
 1. Whether more than one parameters or global variables are declared in one line. e.g. <code>var a, b int = 1, 2</code> is equivalent to <code>var a int = 1; var  b int = 2</code>. (NOTE parallel assignments are not normalized)
 1. All comments.
 1. Code formats. e.g. some useless new lines.

Other Features
--------------
 1. Smart matching algorithm on go ast tree.
 1. If a function is deleted or added as a whole, only one-line message is shown (starting by <code>===</code> or <code>###</code>)
 1. Easily see which function or type, etc. the difference is in.
 1. Import/const/var/func diffrences are shown in order, independent of the lines' order in the source.
 2. Token based line-line difference presentation.

Installation
------------
```bash
$ go get -u github.com/daviddengcn/go-diff
$ go install github.com/daviddengcn/go-diff
$ go-diff <old-file> <new-file>
```
(Make sure <code>$GO_PATH/bin</code> is in system's <code>$PATH</code>)

<b>Used as git diff</b>

1. Link `scripts/go-diff.gs` (you need install [gosl](http://github.com/daviddengcn/gosl)) or `scripts/go-diff.sh` to a folder in `PATH`

1. Set Git external diff driver (change `go-diff.gs` to `go-diff.sh` accordingly if necessary)

```bash
$ git config [--global] diff.external go-diff.gs
$ git diff
```

License
-------
BSD license
