/*
	go-diff is a tool checking semantic difference between source files.

	Currently supported language:

	- Go (fully)

	If the language is not supported or parsing is failed for either file,
	a line-to-line comparing is imposed.
*/
package godiff

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/daviddengcn/go-algs/ed"
	"github.com/daviddengcn/go-colortext"
	"github.com/daviddengcn/go-diff/tm"
	"github.com/daviddengcn/go-villa"
)

func cat(a, sep, b string) string {
	if len(a) > 0 && len(b) > 0 {
		return a + sep + b
	} // if

	return a + b
}

func max(a, b int) int {
	if a > b {
		return a
	} // if

	return b
}

func changeColor(fg ct.Color, fgBright bool, bg ct.Color, bgBright bool) {
	if gOptions.NoColor {
		return
	}

	ct.ChangeColor(fg, fgBright, bg, bgBright)
}

func resetColor() {
	if gOptions.NoColor {
		return
	}

	ct.ResetColor()
}

func greedyMatch(lenA, lenB int, diffF func(iA, iB int) int, delCost, insCost func(int) int) (diffMat villa.IntMatrix, cost int, matA, matB []int) {
	matA, matB = make([]int, lenA), make([]int, lenB)
	villa.IntSlice(matA).Fill(0, lenA, -1)
	villa.IntSlice(matB).Fill(0, lenB, -1)

	diffMat = villa.NewIntMatrix(lenA, lenB)

	for iA := 0; iA < lenA; iA++ {
		if matA[iA] >= 0 {
			continue
		}
		for iB := 0; iB < lenB; iB++ {
			if matB[iB] >= 0 {
				continue
			}

			d := diffF(iA, iB)
			diffMat[iA][iB] = d

			if d == 0 {
				matA[iA], matB[iB] = iB, iA
				break
			}
		}
	}

	mat := diffMat.Clone()
	// mx is a number greater or equal to all mat elements (need not be the exact maximum)
	mx := 0
	for iA := range mat {
		if matA[iA] >= 0 {
			continue
		} // if
		for iB := 0; iB < lenB; iB++ {
			if matB[iB] >= 0 {
				continue
			} // if
			mat[iA][iB] -= delCost(iA) + insCost(iB)
			if mat[iA][iB] > mx {
				mx = mat[iA][iB]
			} // if
		} // for c
	} // for r

	for {
		mn := mx + 1
		selA, selB := -1, -1
		for iA := range mat {
			if matA[iA] >= 0 {
				continue
			} // if
			for iB := 0; iB < lenB; iB++ {
				if matB[iB] >= 0 {
					continue
				} // if

				if mat[iA][iB] < mn {
					mn = mat[iA][iB]
					selA, selB = iA, iB
				} // if
			} // for iB
		} // for iA

		if selA < 0 || mn >= 0 {
			break
		} // if

		matA[selA] = selB
		matB[selB] = selA
	} // for

	for iA := range matA {
		if matA[iA] < 0 {
			cost += delCost(iA)
		} else {
			cost += diffMat[iA][matA[iA]]
		} // else
	} // for iA
	for iB := range matB {
		if matB[iB] < 0 {
			cost += insCost(iB)
		} // if
	} // for iB

	return diffMat, cost, matA, matB
}

const (
	df_NONE = iota
	df_TYPE
	df_CONST
	df_VAR
	df_STRUCT
	df_INTERFACE
	df_FUNC
	df_STAR
	df_VAR_LINE
	df_PAIR
	df_NAMES
	df_VALUES
	df_BLOCK
	df_RESULTS
)

var typeNames []string = []string{
	"",
	"type",
	"const",
	"var",
	"struct",
	"interface",
	"func",
	"*",
	"",
	"",
	"",
	"",
	"",
	""}

type diffFragment interface {
	Type() int
	Weight() int

	// Max diff = this.Weight() + that.Weight()
	calcDiff(that diffFragment) int

	showDiff(that diffFragment)
	// indent is the leading chars from the second line
	sourceLines(indent string) []string
	oneLine() string
}

type fragment struct {
	tp    int
	Parts []diffFragment
}

func (f *fragment) Type() int {
	return f.tp
}

func (f *fragment) Weight() (w int) {
	if f == nil {
		return 10
	} // if

	switch f.Type() {
	case df_FUNC:
		for i := 0; i < 4; i++ {
			w += f.Parts[i].Weight()
		} // for i
		w += int(math.Sqrt(float64(f.Parts[4].Weight())/100.) * 100)
	default:
		for _, p := range f.Parts {
			w += p.Weight()
		} // for p
	}

	switch f.Type() {
	case df_STAR:
		w += 50
	}
	return w
}

func catLines(a []string, sep string, b []string) []string {
	if len(a) > 0 && len(b) > 0 {
		b[0] = cat(a[len(a)-1], sep, b[0])
		a = a[:len(a)-1]
	}

	return append(a, b...)
}

/*
 a[0]
 a[1]
  ...
 cat(a[end], sep, b[0])
 b[1]
  ...
 b[end]
*/
func appendLines(a []string, sep string, b ...string) []string {
	if len(a) > 0 && len(b) > 0 {
		b[0] = cat(a[len(a)-1], sep, b[0])
		a = a[:len(a)-1]
	}

	return append(a, b...)
}

func insertIndent(indent string, lines []string) []string {
	for i := range lines {
		lines[i] = indent + lines[i]
	} // for i

	return lines
}

func insertIndent2(indent string, lines []string) []string {
	for i := range lines {
		if i > 0 {
			lines[i] = indent + lines[i]
		} // if
	} // for i

	return lines
}

func (f *fragment) oneLine() string {
	if f == nil {
		return ""
	}
	switch f.tp {
	}
	lines := f.sourceLines("")
	if len(lines) == 0 {
		return ""
	}

	if len(lines) == 1 {
		return lines[0]
	}

	return lines[0] + " ... " + lines[len(lines)-1] + fmt.Sprintf(" (%d lines)", len(lines))
}

func (f *fragment) sourceLines(indent string) (lines []string) {
	if f == nil {
		return nil
	} // if

	switch f.tp {
	case df_TYPE:
		lines = append(lines, typeNames[f.tp])
		lines = catLines(lines, " ", f.Parts[0].sourceLines(indent))
		lines = catLines(lines, " ", f.Parts[1].sourceLines(indent))
	case df_CONST:
		if len(f.Parts) == 1 {
			lines = append(lines, typeNames[f.tp])
			lines = catLines(lines, " ", f.Parts[0].sourceLines(indent))
		} else {
			lines = append(lines, typeNames[f.tp]+"(")
			for _, p := range f.Parts {
				lines = append(lines, catLines([]string{indent + "    "}, "", p.sourceLines(indent+"    "))...)
			} // p
			lines = append(lines, indent+")")
		} // else
	case df_VAR:
		lines = append(lines, typeNames[f.tp])
		lines = catLines(lines, " ", f.Parts[0].sourceLines(indent+"    "))
	case df_VAR_LINE:
		lines = f.Parts[0].sourceLines(indent)
		lines = catLines(lines, " ", f.Parts[1].sourceLines(indent))
		lines = catLines(lines, " = ", f.Parts[2].sourceLines(indent))
	case df_FUNC:
		lines = append(lines, typeNames[f.tp])
		if f.Parts[0].(*fragment) != nil {
			lines = catLines(catLines(lines, " (", f.Parts[0].sourceLines(indent+"    ")), "", []string{")"}) // recv
		} // if
		lines = catLines(lines, " ", f.Parts[1].sourceLines(indent+"    ")) // name
		lines = catLines(catLines(catLines(lines, "", []string{"("}), "",
			f.Parts[2].sourceLines(indent+"    ")), "", []string{")"}) // params
		lines = catLines(lines, " ", f.Parts[3].sourceLines(indent+"    ")) // returns
		lines = catLines(lines, " ", f.Parts[4].sourceLines(indent))        // body
	case df_RESULTS:
		if len(f.Parts) > 0 {
			if len(f.Parts) > 1 || len(f.Parts[0].(*fragment).Parts[0].(*stringFrag).source) > 0 {
				lines = append(lines, "(")
			} // if
			for i, p := range f.Parts {
				if i > 0 {
					lines = catLines(lines, "", []string{", "})
				} // if
				lines = catLines(lines, "", p.sourceLines(indent+"    "))
			} // for i, p
			if len(f.Parts) > 1 || len(f.Parts[0].(*fragment).Parts[0].(*stringFrag).source) > 0 {
				lines = catLines(lines, "", []string{")"})
			} // if
		} // if
	case df_BLOCK:
		lines = append(lines, "{")
		for _, p := range f.Parts {
			lines = append(lines, catLines([]string{indent + "    "}, "", p.sourceLines(indent+"    "))...)
		} // for p
		lines = append(lines, indent+"}")
	case df_STRUCT, df_INTERFACE:
		if len(f.Parts) == 0 {
			lines = append(lines, typeNames[f.tp]+"{}")
		} else {
			lines = append(lines, typeNames[f.tp]+" {")
			for _, p := range f.Parts {
				lns := p.sourceLines(indent + "    ")
				if len(lns) > 0 {
					lns[0] = indent + "    " + lns[0]
					lines = append(lines, lns...)
				} // if
			} // for p
			lines = append(lines, indent+"}")
		}
	case df_STAR:
		lines = append(lines, typeNames[f.tp])
		lines = catLines(lines, "", f.Parts[0].sourceLines(indent))
	case df_PAIR:
		lines = catLines(f.Parts[0].sourceLines(indent), " ", f.Parts[1].sourceLines(indent))
	case df_NAMES:
		s := ""
		for _, p := range f.Parts {
			s = cat(s, ", ", p.sourceLines(indent + "    ")[0])
		} // for p
		lines = append(lines, s)
	case df_VALUES:
		for _, p := range f.Parts {
			lines = catLines(lines, ", ", p.sourceLines(indent+"    "))
		} // for p
	case df_NONE:
		for _, p := range f.Parts {
			lines = append(lines, p.sourceLines(indent)...)
		} // for p
	default:
		lines = []string{"TYPE: " + typeNames[f.Type()]}
		for _, p := range f.Parts {
			lines = append(lines, p.sourceLines(indent+"    ")...)
		} // for p
	}

	//f.lines = lines
	return lines
}

func (f *fragment) calcDiff(that diffFragment) int {
	switch g := that.(type) {
	case *fragment:
		if f == nil {
			if g == nil {
				return 0
			} else {
				return f.Weight() + g.Weight()
			} // else
		} // if
		if g == nil {
			return f.Weight() + g.Weight()
		} // if

		switch f.Type() {
		case df_STAR:
			if g.Type() == df_STAR {
				return f.Parts[0].calcDiff(g.Parts[0])
			} // if

			return f.Parts[0].calcDiff(g) + 50
		}

		if g.Type() == df_STAR {
			return f.calcDiff(g.Parts[0]) + 50
		} // if

		if f.Type() != g.Type() {
			return f.Weight() + g.Weight()
		} // if

		switch f.Type() {
		case df_FUNC:
			res := int(0)
			for i := 0; i < 4; i++ {
				res += f.Parts[i].calcDiff(g.Parts[i])
			} // for i

			res += int(math.Sqrt(float64(f.Parts[4].calcDiff(g.Parts[4]))/100.) * 100)

			return res
		}

		return ed.EditDistanceF(len(f.Parts), len(g.Parts), func(iA, iB int) int {
			return f.Parts[iA].calcDiff(g.Parts[iB]) * 3 / 2
		}, func(iA int) int {
			return f.Parts[iA].Weight()
		}, func(iB int) int {
			return g.Parts[iB].Weight()
		})
	}
	return f.Weight() + that.Weight()
}

func (f *fragment) showDiff(that diffFragment) {
	diffLines(f.sourceLines(""), that.sourceLines(""), `%s`)
}

type stringFrag struct {
	weight int
	source string
}

func newStringFrag(source string, weight int) *stringFrag {
	return &stringFrag{weight: weight, source: source}
}

func (sf *stringFrag) Type() int {
	return df_NONE
}

func (sf *stringFrag) Weight() int {
	return sf.weight
}

func (sf *stringFrag) calcDiff(that diffFragment) int {
	switch g := that.(type) {
	case *stringFrag:
		s1, s2 := strings.TrimSpace(sf.source), strings.TrimSpace(g.source)
		if len(s1)+len(s2) == 0 {
			return 0
		} // if
		wt := sf.weight + g.weight
		return ed.String(s1, s2) * wt / max(len(s1), len(s2))
	} // switch

	return sf.Weight() + that.Weight()
}

func (sf *stringFrag) showDiff(that diffFragment) {
	diffLines(sf.sourceLines("    "), that.sourceLines("    "), `%s`)
}

func (sf *stringFrag) oneLine() string {
	if sf == nil {
		return ""
	} // if

	return sf.source
}

func (sf *stringFrag) sourceLines(indent string) []string {
	lines := strings.Split(sf.source, "\n")
	for i := range lines {
		if i > 0 {
			lines[i] = indent + lines[i]
		} // if
	} // for i

	return lines
}

const (
	td_STRUCT = iota
	td_INTERFACE
	td_POINTER
	td_ONELINE
)

func newNameTypes(fs *token.FileSet, fl *ast.FieldList) (dfs []diffFragment) {
	for _, f := range fl.List {
		if len(f.Names) > 0 {
			for _, name := range f.Names {
				dfs = append(dfs, &fragment{tp: df_PAIR,
					Parts: []diffFragment{newStringFrag(name.String(), 100),
						newTypeDef(fs, f.Type)}})
			} // for name
		} else {
			// embedding
			dfs = append(dfs, &fragment{tp: df_PAIR,
				Parts: []diffFragment{newStringFrag("", 50),
					newTypeDef(fs, f.Type)}})
		} // else
	} // for f

	return dfs
}

func newTypeDef(fs *token.FileSet, def ast.Expr) diffFragment {
	switch d := def.(type) {
	case *ast.StructType:
		return &fragment{tp: df_STRUCT, Parts: newNameTypes(fs, d.Fields)}

	case *ast.InterfaceType:
		return &fragment{tp: df_INTERFACE, Parts: newNameTypes(fs, d.Methods)}

	case *ast.StarExpr:
		return &fragment{tp: df_STAR, Parts: []diffFragment{newTypeDef(fs, d.X)}}
	} // switch

	var src bytes.Buffer
	(&printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}).Fprint(&src, fs, def)
	return &stringFrag{weight: 50, source: src.String()}
}

func newTypeStmtInfo(fs *token.FileSet, name string, def ast.Expr) *fragment {
	var f fragment

	f.tp = df_TYPE
	f.Parts = []diffFragment{
		newStringFrag(name, 100),
		newTypeDef(fs, def)}

	return &f
}

func newExpDef(fs *token.FileSet, def ast.Expr) diffFragment {
	//ast.Print(fs, def)
	var src bytes.Buffer
	(&printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}).Fprint(&src, fs, def)
	return &stringFrag{weight: 100, source: src.String()}
}

func newVarSpecs(fs *token.FileSet, specs []ast.Spec) (dfs []diffFragment) {
	for _, spec := range specs {
		f := &fragment{tp: df_VAR_LINE}

		names := &fragment{tp: df_NAMES}
		sp := spec.(*ast.ValueSpec)
		for _, name := range sp.Names {
			names.Parts = append(names.Parts, &stringFrag{weight: 100,
				source: fmt.Sprint(name)})
		}
		f.Parts = append(f.Parts, names)

		if sp.Type != nil {
			f.Parts = append(f.Parts, newTypeDef(fs, sp.Type))
		} else {
			f.Parts = append(f.Parts, (*fragment)(nil))
		} // else

		values := &fragment{tp: df_VALUES}
		for _, v := range sp.Values {
			values.Parts = append(values.Parts, newExpDef(fs, v))
		} // for v
		f.Parts = append(f.Parts, values)

		dfs = append(dfs, f)
	}

	return dfs
}

func printToLines(fs *token.FileSet, node interface{}) []string {
	var src bytes.Buffer
	(&printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}).Fprint(&src, fs, node)
	return strings.Split(src.String(), "\n")
}

func nodeToLines(fs *token.FileSet, node interface{}) (lines []string) {
	switch nd := node.(type) {
	case *ast.IfStmt:
		lines = append(lines, "if")
		if nd.Init != nil {
			lines = appendLines(lines, " ", nodeToLines(fs, nd.Init)...)
			lines = appendLines(lines, "", ";")
		} // if

		lines = catLines(lines, " ", nodeToLines(fs, nd.Cond))
		lines = catLines(lines, " ", []string{"{"})
		lines = append(lines, insertIndent("    ", blockToLines(fs, nd.Body))...)
		lines = append(lines, "}")
		if nd.Else != nil {
			//ast.Print(fs, st.Else)
			lines = catLines(lines, "", []string{" else "})
			lines = catLines(lines, "", nodeToLines(fs, nd.Else))
		} // if
	case *ast.AssignStmt:
		for _, exp := range nd.Lhs {
			lines = catLines(lines, ", ", nodeToLines(fs, exp))
		} // for i

		lines = catLines(lines, "", []string{" " + nd.Tok.String() + " "})

		for i, exp := range nd.Rhs {
			if i > 0 {
				lines = catLines(lines, "", []string{", "})
			} // if
			lines = catLines(lines, "", nodeToLines(fs, exp))
		} // for i

	case *ast.ForStmt:
		lines = append(lines, "for")
		if nd.Cond != nil {
			lns := []string{}
			if nd.Init != nil {
				lns = catLines(lns, "; ", nodeToLines(fs, nd.Init))
			} // if
			lns = catLines(lns, "; ", nodeToLines(fs, nd.Cond))
			if nd.Post != nil {
				lns = catLines(lns, "; ", nodeToLines(fs, nd.Post))
			} // if

			lines = catLines(lines, " ", lns)
		} // if
		lines = catLines(lines, "", []string{" {"})
		lines = append(lines, insertIndent("    ", blockToLines(fs, nd.Body))...)
		lines = append(lines, "}")
	case *ast.RangeStmt:
		lines = append(lines, "for")
		lines = catLines(lines, " ", nodeToLines(fs, nd.Key))
		if nd.Value != nil {
			lines = catLines(lines, ", ", nodeToLines(fs, nd.Value))
		} // if
		lines = catLines(lines, "", []string{" " + nd.Tok.String() + " "})
		lines = catLines(lines, "", []string{" range"})
		lines = catLines(lines, " ", nodeToLines(fs, nd.X))
		lines = catLines(lines, "", []string{" {"})
		lines = append(lines, insertIndent("    ", blockToLines(fs, nd.Body))...)
		lines = append(lines, "}")

	case *ast.BlockStmt:
		lines = append(lines, "{")
		lines = append(lines, insertIndent("    ", blockToLines(fs, nd))...)
		lines = append(lines, "}")

	case *ast.ReturnStmt:
		lines = append(lines, "return")
		if nd.Results != nil {
			for i, e := range nd.Results {
				if i == 0 {
					lines = appendLines(lines, " ", nodeToLines(fs, e)...)
				} else {
					lines = appendLines(lines, ", ", nodeToLines(fs, e)...)
				} // else
			} // for i, e
		} // if

	case *ast.DeferStmt:
		lines = append(lines, "defer")
		lines = appendLines(lines, " ", nodeToLines(fs, nd.Call)...)

	case *ast.GoStmt:
		lines = append(lines, "go")
		lines = appendLines(lines, " ", nodeToLines(fs, nd.Call)...)

	case *ast.SendStmt:
		lines = append(lines, nodeToLines(fs, nd.Chan)...)
		lines = appendLines(lines, " ", "<-")
		lines = appendLines(lines, " ", nodeToLines(fs, nd.Value)...)

	case *ast.ExprStmt:
		return nodeToLines(fs, nd.X)

	case *ast.EmptyStmt:
		// Do nothing

	case *ast.SwitchStmt:
		lines = append(lines, "switch")
		if nd.Init != nil {
			lines = appendLines(lines, " ", nodeToLines(fs, nd.Init)...)
			lines = appendLines(lines, "", ";")
		} // if
		if nd.Tag != nil {
			lines = appendLines(lines, " ", nodeToLines(fs, nd.Tag)...)
		} // if
		lines = appendLines(lines, " ", nodeToLines(fs, nd.Body)...)

	case *ast.TypeSwitchStmt:
		lines = append(lines, "switch")
		if nd.Init != nil {
			lines = appendLines(lines, " ", nodeToLines(fs, nd.Init)...)
			lines = appendLines(lines, "", ";")
		} // if
		lines = appendLines(lines, " ", nodeToLines(fs, nd.Assign)...)
		lines = appendLines(lines, " ", nodeToLines(fs, nd.Body)...)

	case *ast.CompositeLit:
		if nd.Type != nil {
			lines = append(lines, printToLines(fs, nd.Type)...)
		}
		if len(nd.Elts) == 0 {
			// short form
			lines = appendLines(lines, "", "{}")
		} else {
			lines = appendLines(lines, "", "{")

			for _, el := range nd.Elts {
				lines = append(lines, insertIndent("    ", nodeToLines(fs, el))...)
				lines = appendLines(lines, "", ",")
			}
			// put } in a new line
			lines = append(lines, "")
			lines = appendLines(lines, "", "}")
		}

	case *ast.UnaryExpr:
		lines = append(lines, nd.Op.String())
		lines = appendLines(lines, "", nodeToLines(fs, nd.X)...)

	case *ast.BinaryExpr:
		lines = appendLines(lines, "", nodeToLines(fs, nd.X)...)
		lines = appendLines(lines, " ", nd.Op.String())
		lines = appendLines(lines, " ", nodeToLines(fs, nd.Y)...)

	case *ast.ParenExpr:
		lines = append(lines, "(")
		lines = appendLines(lines, "", nodeToLines(fs, nd.X)...)
		lines = appendLines(lines, "", ")")

	case *ast.CallExpr:
		lines = append(lines, nodeToLines(fs, nd.Fun)...)
		lines = appendLines(lines, "", "(")
		for i, a := range nd.Args {
			if i > 0 {
				lines = appendLines(lines, "", ", ")
			} // if
			lines = appendLines(lines, "", nodeToLines(fs, a)...)
		} // for i, el
		if nd.Ellipsis > 0 {
			lines = appendLines(lines, "", "...")
		} // if
		lines = appendLines(lines, "", ")")
	case *ast.KeyValueExpr:
		lines = append(lines, nodeToLines(fs, nd.Key)...)
		lines = appendLines(lines, ": ", nodeToLines(fs, nd.Value)...)
	case *ast.FuncLit:
		lines = nodeToLines(fs, nd.Type)
		lines = appendLines(lines, " ", nodeToLines(fs, nd.Body)...)

	case *ast.CaseClause:
		if nd.List == nil {
			lines = append(lines, "default:")
		} else {
			lines = append(lines, "case ")
			for i, e := range nd.List {
				if i > 0 {
					lines = appendLines(lines, "", ", ")
				} // if
				lines = appendLines(lines, "", nodeToLines(fs, e)...)
			} // for i
			lines = appendLines(lines, "", ":")
		} // else

		for _, st := range nd.Body {
			lines = append(lines, insertIndent("    ", nodeToLines(fs, st))...)
		} // for

	case *ast.SelectorExpr:
		lines = append(lines, nodeToLines(fs, nd.X)...)
		lines = appendLines(lines, "", ".")
		lines = appendLines(lines, "", nodeToLines(fs, nd.Sel)...)
	case *ast.LabeledStmt:
		lines = append(lines, nodeToLines(fs, nd.Label)...)
		lines = appendLines(lines, "", ":")
		lines = appendLines(lines, "", nodeToLines(fs, nd.Stmt)...)
	case *ast.Ident, *ast.BasicLit, *ast.DeclStmt, *ast.BranchStmt, *ast.IndexExpr, *ast.FuncType, *ast.SliceExpr, *ast.StarExpr, *ast.ArrayType, *ast.TypeAssertExpr:
		return printToLines(fs, nd)
	default:
		//ast.Print(fs, nd)
		//ast.Print(fs, printToLines(fs, nd))

		return printToLines(fs, nd)
	}

	return lines
}

func blockToLines(fs *token.FileSet, blk *ast.BlockStmt) (lines []string) {
	for _, s := range blk.List {
		lines = append(lines, nodeToLines(fs, s)...)
	} // for s

	return lines
}

func newBlockDecl(fs *token.FileSet, blk *ast.BlockStmt) (f *fragment) {
	f = &fragment{tp: df_BLOCK}
	lines := blockToLines(fs, blk)
	for _, line := range lines {
		f.Parts = append(f.Parts, &stringFrag{weight: 100, source: line})
	} // for line

	return f
}

func newFuncDecl(fs *token.FileSet, d *ast.FuncDecl) (f *fragment) {
	f = &fragment{tp: df_FUNC}

	// recv
	if d.Recv != nil {
		f.Parts = append(f.Parts, newNameTypes(fs, d.Recv)...)
	} else {
		f.Parts = append(f.Parts, (*fragment)(nil))
	} // else

	// name
	f.Parts = append(f.Parts, &stringFrag{weight: 200, source: fmt.Sprint(d.Name)})

	//  params
	if d.Type.Params != nil {
		f.Parts = append(f.Parts, &fragment{tp: df_VALUES,
			Parts: newNameTypes(fs, d.Type.Params)})
	} else {
		f.Parts = append(f.Parts, (*fragment)(nil))
	} // else

	// Results
	if d.Type.Results != nil {
		f.Parts = append(f.Parts, &fragment{tp: df_RESULTS, Parts: newNameTypes(fs, d.Type.Results)})
	} else {
		f.Parts = append(f.Parts, (*fragment)(nil))
	} // else

	// body
	if d.Body != nil {
		f.Parts = append(f.Parts, newBlockDecl(fs, d.Body))
	} else {
		f.Parts = append(f.Parts, (*fragment)(nil))
	} // else
	return f
}

type fileInfo struct {
	f     *ast.File
	fs    *token.FileSet
	types *fragment
	vars  *fragment
	funcs *fragment
}

func (info *fileInfo) collect() {
	info.types = &fragment{}
	info.vars = &fragment{}
	info.funcs = &fragment{}

	for _, decl := range info.f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				for i := range d.Specs {
					spec := d.Specs[i].(*ast.TypeSpec)
					//ast.Print(info.fs, spec)
					ti := newTypeStmtInfo(info.fs, spec.Name.String(), spec.Type)
					info.types.Parts = append(info.types.Parts, ti)
				} // for i
			case token.CONST:
				// fmt.Println(d)
				//ast.Print(info.fs, d)
				v := &fragment{tp: df_CONST, Parts: newVarSpecs(info.fs, d.Specs)}
				info.vars.Parts = append(info.vars.Parts, v)
			case token.VAR:
				//ast.Print(info.fs, d)
				vss := newVarSpecs(info.fs, d.Specs)
				for _, vs := range vss {
					info.vars.Parts = append(info.vars.Parts, &fragment{tp: df_VAR, Parts: []diffFragment{vs}})
				} // for spec
			case token.IMPORT:
				// ignore
			default:
				// Unknow
				fmt.Fprintln(out, d)
			} // switch d.tok
		case *ast.FuncDecl:
			//fmt.Printf("%#v\n", d)
			fd := newFuncDecl(info.fs, d)
			info.funcs.Parts = append(info.funcs.Parts, fd)
			//ast.Print(info.fs, d)
		default:
			// fmt.Println(d)
		} // switch decl.(type)
	} // for decl
}

func parse(fn string, src interface{}) (*fileInfo, error) {
	if fn == "/dev/null" {
		return &fileInfo{
			f:     &ast.File{},
			types: &fragment{},
			vars:  &fragment{},
			funcs: &fragment{},
		}, nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fn, src, 0)
	if err != nil {
		return nil, err
	} // if

	info := &fileInfo{f: f, fs: fset}
	info.collect()

	return info, nil
}

const (
	// delete color
	del_COLOR = ct.Red
	// insert color
	ins_COLOR = ct.Green
	// matched color
	mat_COLOR = ct.White
	// folded color
	fld_COLOR = ct.Yellow
)

func showDelWholeLine(line string) {
	changeColor(del_COLOR, false, ct.None, false)
	fmt.Fprintln(out, "===", line)
	resetColor()
}
func showDelLine(line string) {
	changeColor(del_COLOR, false, ct.None, false)
	fmt.Fprintln(out, "---", line)
	resetColor()
}
func showColorDelLine(line, lcs string) {
	changeColor(del_COLOR, false, ct.None, false)
	fmt.Fprint(out, "--- ")
	lcsr := []rune(lcs)
	for _, c := range line {
		if len(lcsr) > 0 && lcsr[0] == c {
			resetColor()
			lcsr = lcsr[1:]
		} else {
			changeColor(del_COLOR, false, ct.None, false)
		} // else
		fmt.Fprintf(out, "%c", c)
	} // for c

	resetColor()
	fmt.Fprintln(out)
}

func showDelLines(lines []string, gapLines int) {
	if len(lines) <= gapLines*2+1 {
		for _, line := range lines {
			showDelLine(line)
		} // for line
		return
	} // if

	for i, line := range lines {
		if i < gapLines || i >= len(lines)-gapLines {
			showDelLine(line)
		} // if
		if i == gapLines {
			showDelWholeLine(fmt.Sprintf("    ... (%d lines)", len(lines)-gapLines*2))
		} // if
	} // for i
}

func showInsLine(line string) {
	changeColor(ins_COLOR, false, ct.None, false)
	fmt.Fprintln(out, "+++", line)
	resetColor()
}
func showColorInsLine(line, lcs string) {
	changeColor(ins_COLOR, false, ct.None, false)
	fmt.Fprint(out, "+++ ")
	lcsr := []rune(lcs)
	for _, c := range line {
		if len(lcsr) > 0 && lcsr[0] == c {
			resetColor()
			lcsr = lcsr[1:]
		} else {
			changeColor(ins_COLOR, false, ct.None, false)
		} // else
		fmt.Fprintf(out, "%c", c)
	} // for c

	resetColor()
	fmt.Fprintln(out)
}

func showInsWholeLine(line string) {
	changeColor(ins_COLOR, false, ct.None, false)
	fmt.Fprintln(out, "###", line)
	resetColor()
}

func showInsLines(lines []string, gapLines int) {
	if len(lines) <= gapLines*2+1 {
		for _, line := range lines {
			showInsLine(line)
		} // for line
		return
	} // if

	for i, line := range lines {
		if i < gapLines || i >= len(lines)-gapLines {
			showInsLine(line)
		} // if
		if i == gapLines {
			showInsWholeLine(fmt.Sprintf("    ... (%d lines)", len(lines)-gapLines*2))
		} // if
	} // for i
}

func showDelTokens(del []string, mat []int, ins []string) {
	changeColor(del_COLOR, false, ct.None, false)
	fmt.Fprint(out, "--- ")

	for i, tk := range del {
		if mat[i] < 0 || tk != ins[mat[i]] {
			changeColor(del_COLOR, false, ct.None, false)
		} else {
			changeColor(mat_COLOR, false, ct.None, false)
		}

		fmt.Fprint(out, tk)
	} // for i

	resetColor()
	fmt.Fprintln(out)
}

func showInsTokens(ins []string, mat []int, del []string) {
	changeColor(ins_COLOR, false, ct.None, false)
	fmt.Fprint(out, "+++ ")

	for i, tk := range ins {
		if mat[i] < 0 || tk != del[mat[i]] {
			changeColor(ins_COLOR, false, ct.None, false)
		} else {
			resetColor()
		} // else

		fmt.Fprint(out, tk)
	} // for i

	resetColor()
	fmt.Fprintln(out)
}

func showDiffLine(del, ins string) {
	delT, insT := tm.LineToTokens(del), tm.LineToTokens(ins)
	matA, matB := tm.MatchTokens(delT, insT)

	showDelTokens(delT, matA, insT)
	showInsTokens(insT, matB, delT)
}

func diffLineSet(orgLines, newLines []string, format string) {
	sort.Strings(orgLines)
	sort.Strings(newLines)

	_, matA, matB := ed.EditDistanceFFull(len(orgLines), len(newLines), func(iA, iB int) int {
		return tm.DiffOfStrings(orgLines[iA], newLines[iB], 4000)
	}, ed.ConstCost(1000), ed.ConstCost(1000))

	for i, j := 0, 0; i < len(orgLines) || j < len(newLines); {
		switch {
		case j >= len(newLines) || i < len(orgLines) && matA[i] < 0:
			showDelLine(fmt.Sprintf(format, orgLines[i]))
			i++
		case i >= len(orgLines) || j < len(newLines) && matB[j] < 0:
			showInsLine(fmt.Sprintf(format, newLines[j]))
			j++
		default:
			if strings.TrimSpace(orgLines[i]) != strings.TrimSpace(newLines[j]) {
				showDiffLine(fmt.Sprintf(format, orgLines[i]), fmt.Sprintf(format, newLines[j]))
			} // if
			i++
			j++
		}
	} // for i, j
}

type lineOutputer interface {
	outputIns(line string)
	outputDel(line string)
	outputSame(line string)
	outputChange(del, ins string)
	end()
}

type lineOutput struct {
	sameLines []string
}

func (lo *lineOutput) outputIns(line string) {
	lo.end()
	showInsLine(line)
}

func (lo *lineOutput) outputDel(line string) {
	lo.end()
	showDelLine(line)
}

func (lo *lineOutput) outputChange(del, ins string) {
	lo.end()
	showDiffLine(del, ins)
}

func (lo *lineOutput) outputSame(line string) {
	lo.sameLines = append(lo.sameLines, line)
}

func (lo *lineOutput) end() {
	if len(lo.sameLines) > 0 {
		fmt.Fprintln(out, "   ", lo.sameLines[0])
		if len(lo.sameLines) == 3 {
			fmt.Fprintln(out, "   ", lo.sameLines[1])
		} // if
		if len(lo.sameLines) > 3 {
			changeColor(fld_COLOR, false, ct.None, false)
			fmt.Fprintf(out, "        ... (%d lines)\n", len(lo.sameLines)-2)
			resetColor()
		} // if
		if len(lo.sameLines) > 1 {
			fmt.Fprintln(out, "   ", lo.sameLines[len(lo.sameLines)-1])
		} // if
	} // if

	lo.sameLines = nil
}

func offsetHeadTails(orgLines, newLines []string) (start, orgEnd, newEnd int) {
	start = 0
	for start < len(orgLines) && start < len(newLines) && orgLines[start] == newLines[start] {
		start++
	}

	orgEnd, newEnd = len(orgLines), len(newLines)
	for orgEnd > start && newEnd > start && orgLines[orgEnd-1] == newLines[newEnd-1] {
		orgEnd--
		newEnd--
	}
	return
}

func diffLinesTo(orgLines, newLines []string, format string, lo lineOutputer) int {
	if len(orgLines)+len(newLines) == 0 {
		return 0
	}

	start, orgEnd, newEnd := 0, len(orgLines), len(newLines)

	if len(orgLines)*len(newLines) > 1024*1024 {
		// Use trivial comparison to offset same head and tail lines.
		start, orgEnd, newEnd = offsetHeadTails(orgLines, newLines)
	}

	fastMode := false
	if len(orgLines)*len(newLines) > 1024*1024 {
		fastMode = true
	}

	_, matA, matB := ed.EditDistanceFFull(orgEnd-start, newEnd-start, func(iA, iB int) int {
		sa, sb := orgLines[iA+start], newLines[iB+start]
		if sa == sb {
			return 0
		}
		sa, sb = strings.TrimSpace(sa), strings.TrimSpace(sb)
		if sa == sb {
			return 1
		}

		mx := (len(sa) + len(sb)) * 150

		var dist int

		if fastMode && len(sa) > 10*len(sb) {
			dist = 100 * (len(sa) - len(sb))
		} else if fastMode && len(sb) > 10*len(sa) {
			dist = 100 * (len(sb) - len(sa))
		} else {
			// When sa and sb has 1/3 in common, convertion const is equal to del+ins const
			dist = tm.CalcDiffOfSourceLine(sa, sb, mx)
		}
		// Even a small change, both lines will be shown, so add a 10% penalty on that.
		return (dist*9+mx)/10 + 1
	}, func(iA int) int {
		return max(1, len(strings.TrimSpace(orgLines[iA+start]))*100)
	}, func(iB int) int {
		return max(1, len(strings.TrimSpace(newLines[iB+start]))*100)
	})

	cnt := 0

	for i, j := 0, 0; i < len(orgLines) || j < len(newLines); {
		switch {
		case i < start || i >= orgEnd && j >= newEnd:
			// cut by offsetHeadTails
			lo.outputSame(fmt.Sprintf(format, newLines[j]))
			i++
			j++
		case j >= newEnd || i < orgEnd && matA[i-start] < 0:
			lo.outputDel(fmt.Sprintf(format, orgLines[i]))
			cnt++
			i++
		case i >= orgEnd || j < newEnd && matB[j-start] < 0:
			lo.outputIns(fmt.Sprintf(format, newLines[j]))
			cnt++
			j++
		default:
			if strings.TrimSpace(orgLines[i]) != strings.TrimSpace(newLines[j]) {
				lo.outputChange(fmt.Sprintf(format, orgLines[i]), fmt.Sprintf(format, newLines[j]))
				cnt += 2
			} else {
				lo.outputSame(fmt.Sprintf(format, newLines[j]))
			} // else
			i++
			j++
		}
	}
	lo.end()
	return cnt
}

/*
  Returns the number of operation lines.
*/
func diffLines(orgLines, newLines []string, format string) int {
	return diffLinesTo(orgLines, newLines, format, &lineOutput{})
}

/*
   Diff Package
*/
func diffPackage(orgInfo, newInfo *fileInfo) {
	orgName := orgInfo.f.Name.String()
	newName := newInfo.f.Name.String()
	if orgName != newName {
		showDiffLine("package "+orgName, "package "+newName)
	} //  if
}

/*
   Diff Imports
*/
func extractImports(info *fileInfo) []string {
	imports := make([]string, 0, len(info.f.Imports))
	for _, imp := range info.f.Imports {
		imports = append(imports, imp.Path.Value)
	} // for imp

	return imports
}

func diffImports(orgInfo, newInfo *fileInfo) {
	orgImports := extractImports(orgInfo)
	newImports := extractImports(newInfo)

	diffLineSet(orgImports, newImports, `import %s`)
}

/*
   Diff Types
*/
func diffTypes(orgInfo, newInfo *fileInfo) {
	mat, _, matA, matB := greedyMatch(len(orgInfo.types.Parts), len(newInfo.types.Parts), func(iA, iB int) int {
		return orgInfo.types.Parts[iA].calcDiff(newInfo.types.Parts[iB]) * 3 / 2
	}, func(iA int) int {
		return orgInfo.types.Parts[iA].Weight()
	}, func(iB int) int {
		return newInfo.types.Parts[iB].Weight()
	})

	j0 := 0
	for i := range matA {
		j := matA[i]
		if j < 0 {
			showDelWholeLine(orgInfo.types.Parts[i].oneLine())
		} else {
			for ; j0 < j; j0++ {
				if matB[j0] < 0 {
					showInsWholeLine(newInfo.types.Parts[j0].oneLine())
				}
			}

			if mat[i][j] > 0 {
				orgInfo.types.Parts[i].showDiff(newInfo.types.Parts[j])
			} //  if
		} // else
	} // for i

	for ; j0 < len(matB); j0++ {
		if matB[j0] < 0 {
			showInsWholeLine(newInfo.types.Parts[j0].oneLine())
		}
	}
}

func diffVars(orgInfo, newInfo *fileInfo) {
	mat, _, matA, matB := greedyMatch(len(orgInfo.vars.Parts), len(newInfo.vars.Parts), func(iA, iB int) int {
		return orgInfo.vars.Parts[iA].calcDiff(newInfo.vars.Parts[iB]) * 3 / 2
	}, func(iA int) int {
		return orgInfo.vars.Parts[iA].Weight()
	}, func(iB int) int {
		return newInfo.vars.Parts[iB].Weight()
	})

	j0 := 0
	for i := range matA {
		j := matA[i]
		if j < 0 {
			showDelLines(orgInfo.vars.Parts[i].sourceLines(""), 2)
			// fmt.Println()
		} else {
			for ; j0 < j; j0++ {
				if matB[j0] < 0 {
					showInsLines(newInfo.vars.Parts[j0].sourceLines(""), 2)
				} // if
			}

			if mat[i][j] > 0 {
				orgInfo.vars.Parts[i].showDiff(newInfo.vars.Parts[j])
				fmt.Fprintln(out)
			} //  if
		} // else
	} // for i

	for ; j0 < len(matB); j0++ {
		if matB[j0] < 0 {
			showInsLines(newInfo.vars.Parts[j0].sourceLines(""), 2)
		} // if
	}
}

func diffFuncs(orgInfo, newInfo *fileInfo) {
	mat, _, matA, matB := greedyMatch(len(orgInfo.funcs.Parts), len(newInfo.funcs.Parts), func(iA, iB int) int {
		return orgInfo.funcs.Parts[iA].calcDiff(newInfo.funcs.Parts[iB]) * 3 / 2
	}, func(iA int) int {
		return orgInfo.funcs.Parts[iA].Weight()
	}, func(iB int) int {
		return newInfo.funcs.Parts[iB].Weight()
	})

	j0 := 0
	for i := range matA {
		j := matA[i]
		if j < 0 {
			showDelWholeLine(orgInfo.funcs.Parts[i].oneLine())
		} else {
			for ; j0 < j; j0++ {
				if matB[j0] < 0 {
					showInsWholeLine(newInfo.funcs.Parts[j0].oneLine())
				}
			}
			if mat[i][j] > 0 {
				orgInfo.funcs.Parts[i].showDiff(newInfo.funcs.Parts[j])
			} //  if
		} // else
	} // for i

	for ; j0 < len(matB); j0++ {
		if matB[j0] < 0 {
			showInsWholeLine(newInfo.funcs.Parts[j0].oneLine())
		}
	}
}

func diff(orgInfo, newInfo *fileInfo) {
	diffPackage(orgInfo, newInfo)
	diffImports(orgInfo, newInfo)
	diffTypes(orgInfo, newInfo)
	diffVars(orgInfo, newInfo)
	diffFuncs(orgInfo, newInfo)
}

func readLines(fn villa.Path) []string {
	bts, err := fn.ReadFile()

	if err != nil {
		return nil
	}

	return strings.Split(string(bts), "\n")
}

// Options specifies options for processing files.
type Options struct {
	NoColor bool // Turn off the colors when printing.
}

var (
	out      io.Writer
	gOptions Options
)

// ExecWriter prints the difference between two Go files to stdout.
func Exec(orgFn, newFn string, options Options) {
	out = os.Stdout
	gOptions = options

	fmt.Printf("Difference between %s and %s ...\n", orgFn, newFn)

	orgInfo, err1 := parse(orgFn, nil)
	newInfo, err2 := parse(newFn, nil)

	if err1 != nil || err2 != nil {
		orgLines := readLines(villa.Path(orgFn))
		newLines := readLines(villa.Path(newFn))

		diffLines(orgLines, newLines, "%s")
		return
	}

	diff(orgInfo, newInfo)
}

// ExecWriter prints the difference between two parsed Go files into w.
func ExecWriter(w io.Writer, fset0 *token.FileSet, file0 *ast.File, fset1 *token.FileSet, file1 *ast.File, options Options) {
	out = w
	gOptions = options

	orgInfo := &fileInfo{f: file0, fs: fset0}
	orgInfo.collect()
	newInfo := &fileInfo{f: file1, fs: fset1}
	newInfo.collect()

	diff(orgInfo, newInfo)
}
