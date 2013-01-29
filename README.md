go-diff
=======

A diff tool for go languange. It can show the semantic difference between two go source files.

Ignored Difference
------------------
 1. Order of <code>import</code> statements
 1. Order of definitions of global <code>type</code>/<code>const</code>/<code>var</code>/<code>func</code>
 1. Whether more than one parameters or global variables are declared in one line. e.g. <code>var a, b int = 1, 2</code> is equivalent to <code>var a int = 1; var  b int = 2</code>. (NOTE parallel assignments are not normalized)
 1. All comments
 1. Code formats. e.g. some useless new lines.

Other Features
--------------
 1. If a function is deleted or added as a whole, only one-line message is shown (starting by <code>===</code> or <code>###</code>)
 1. Easily see which function or type, etc. the difference is in.
 1. Import/const/var/func diffrences are shown in order, independent of the lines' order in ths source.

Installation
------------
```bash
$ go get -u github.com/daviddengcn/go-diff
$ go install github.com/daviddengcn/go-diff
$ go-diff <new-file> <org-file>
```

<b>Using as git diff</b>
```bash
$ set GIT_EXTERNAL_DIFF="go-diff"
$ git diff
```

License
-------
BSD license
