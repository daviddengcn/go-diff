/*
	go-diff is a tool checking semantic difference between source files.

	Currently supported language:
		Go fully

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

func GreedyMatch(lenA, lenB int, diffF func(iA, iB int) int, delCost, insCost func(int) int) (diffMat villa.IntMatrix, cost int, matA, matB []int) {
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
	DF_NONE = iota
	DF_TYPE
	DF_CONST
	DF_VAR
	DF_STRUCT
	DF_INTERFACE
	DF_FUNC
	DF_STAR
	DF_VAR_LINE
	DF_PAIR
	DF_NAMES
	DF_VALUES
	DF_BLOCK
	DF_RESULTS
)

var TYPE_NAMES []string = []string{
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

type DiffFragment interface {
	Type() int
	Weight() int

	// Max diff = this.Weight() + that.Weight()
	calcDiff(that DiffFragment) int

	showDiff(that DiffFragment)
	// indent is the leading chars from the second line
	sourceLines(indent string) []string
	oneLine() string
}

type Fragment struct {
	tp    int
	Parts []DiffFragment
}

func (f *Fragment) Type() int {
	return f.tp
}

func (f *Fragment) Weight() (w int) {
	if f == nil {
		return 10
	} // if

	switch f.Type() {
	case DF_FUNC:
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
	case DF_STAR:
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

func (f *Fragment) oneLine() string {
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

func (f *Fragment) sourceLines(indent string) (lines []string) {
	if f == nil {
		return nil
	} // if

	switch f.tp {
	case DF_TYPE:
		lines = append(lines, TYPE_NAMES[f.tp])
		lines = catLines(lines, " ", f.Parts[0].sourceLines(indent))
		lines = catLines(lines, " ", f.Parts[1].sourceLines(indent))
	case DF_CONST:
		if len(f.Parts) == 1 {
			lines = append(lines, TYPE_NAMES[f.tp])
			lines = catLines(lines, " ", f.Parts[0].sourceLines(indent))
		} else {
			lines = append(lines, TYPE_NAMES[f.tp]+"(")
			for _, p := range f.Parts {
				lines = append(lines, catLines([]string{indent + "    "}, "", p.sourceLines(indent+"    "))...)
			} // p
			lines = append(lines, indent+")")
		} // else
	case DF_VAR:
		lines = append(lines, TYPE_NAMES[f.tp])
		lines = catLines(lines, " ", f.Parts[0].sourceLines(indent+"    "))
	case DF_VAR_LINE:
		lines = f.Parts[0].sourceLines(indent)
		lines = catLines(lines, " ", f.Parts[1].sourceLines(indent))
		lines = catLines(lines, " = ", f.Parts[2].sourceLines(indent))
	case DF_FUNC:
		lines = append(lines, TYPE_NAMES[f.tp])
		if f.Parts[0].(*Fragment) != nil {
			lines = catLines(catLines(lines, " (", f.Parts[0].sourceLines(indent+"    ")), "", []string{")"}) // recv
		} // if
		lines = catLines(lines, " ", f.Parts[1].sourceLines(indent+"    ")) // name
		lines = catLines(catLines(catLines(lines, "", []string{"("}), "",
			f.Parts[2].sourceLines(indent+"    ")), "", []string{")"}) // params
		lines = catLines(lines, " ", f.Parts[3].sourceLines(indent+"    ")) // returns
		lines = catLines(lines, " ", f.Parts[4].sourceLines(indent))        // body
	case DF_RESULTS:
		if len(f.Parts) > 0 {
			if len(f.Parts) > 1 || len(f.Parts[0].(*Fragment).Parts[0].(*StringFrag).source) > 0 {
				lines = append(lines, "(")
			} // if
			for i, p := range f.Parts {
				if i > 0 {
					lines = catLines(lines, "", []string{", "})
				} // if
				lines = catLines(lines, "", p.sourceLines(indent+"    "))
			} // for i, p
			if len(f.Parts) > 1 || len(f.Parts[0].(*Fragment).Parts[0].(*StringFrag).source) > 0 {
				lines = catLines(lines, "", []string{")"})
			} // if
		} // if
	case DF_BLOCK:
		lines = append(lines, "{")
		for _, p := range f.Parts {
			lines = append(lines, catLines([]string{indent + "    "}, "", p.sourceLines(indent+"    "))...)
		} // for p
		lines = append(lines, indent+"}")
	case DF_STRUCT, DF_INTERFACE:
		if len(f.Parts) == 0 {
			lines = append(lines, TYPE_NAMES[f.tp]+"{}")
		} else {
			lines = append(lines, TYPE_NAMES[f.tp]+" {")
			for _, p := range f.Parts {
				lns := p.sourceLines(indent + "    ")
				if len(lns) > 0 {
					lns[0] = indent + "    " + lns[0]
					lines = append(lines, lns...)
				} // if
			} // for p
			lines = append(lines, indent+"}")
		}
	case DF_STAR:
		lines = append(lines, TYPE_NAMES[f.tp])
		lines = catLines(lines, "", f.Parts[0].sourceLines(indent))
	case DF_PAIR:
		lines = catLines(f.Parts[0].sourceLines(indent), " ", f.Parts[1].sourceLines(indent))
	case DF_NAMES:
		s := ""
		for _, p := range f.Parts {
			s = cat(s, ", ", p.sourceLines(indent + "    ")[0])
		} // for p
		lines = append(lines, s)
	case DF_VALUES:
		for _, p := range f.Parts {
			lines = catLines(lines, ", ", p.sourceLines(indent+"    "))
		} // for p
	case DF_NONE:
		for _, p := range f.Parts {
			lines = append(lines, p.sourceLines(indent)...)
		} // for p
	default:
		lines = []string{"TYPE: " + TYPE_NAMES[f.Type()]}
		for _, p := range f.Parts {
			lines = append(lines, p.sourceLines(indent+"    ")...)
		} // for p
	}

	//f.lines = lines
	return lines
}

func (f *Fragment) calcDiff(that DiffFragment) int {
	switch g := that.(type) {
	case *Fragment:
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
		case DF_STAR:
			if g.Type() == DF_STAR {
				return f.Parts[0].calcDiff(g.Parts[0])
			} // if

			return f.Parts[0].calcDiff(g) + 50
		}

		if g.Type() == DF_STAR {
			return f.calcDiff(g.Parts[0]) + 50
		} // if

		if f.Type() != g.Type() {
			return f.Weight() + g.Weight()
		} // if

		switch f.Type() {
		case DF_FUNC:
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

func (f *Fragment) showDiff(that DiffFragment) {
	DiffLines(f.sourceLines(""), that.sourceLines(""), `%s`)
}

type StringFrag struct {
	weight int
	source string
}

func newStringFrag(source string, weight int) *StringFrag {
	return &StringFrag{weight: weight, source: source}
}

func (sf *StringFrag) Type() int {
	return DF_NONE
}

func (sf *StringFrag) Weight() int {
	return sf.weight
}

func (sf *StringFrag) calcDiff(that DiffFragment) int {
	switch g := that.(type) {
	case *StringFrag:
		s1, s2 := strings.TrimSpace(sf.source), strings.TrimSpace(g.source)
		if len(s1)+len(s2) == 0 {
			return 0
		} // if
		wt := sf.weight + g.weight
		return ed.String(s1, s2) * wt / max(len(s1), len(s2))
	} // switch

	return sf.Weight() + that.Weight()
}

func (sf *StringFrag) showDiff(that DiffFragment) {
	DiffLines(sf.sourceLines("    "), that.sourceLines("    "), `%s`)
}

func (sf *StringFrag) oneLine() string {
	if sf == nil {
		return ""
	} // if

	return sf.source
}

func (sf *StringFrag) sourceLines(indent string) []string {
	lines := strings.Split(sf.source, "\n")
	for i := range lines {
		if i > 0 {
			lines[i] = indent + lines[i]
		} // if
	} // for i

	return lines
}

const (
	TD_STRUCT = iota
	TD_INTERFACE
	TD_POINTER
	TD_ONELINE
)

func newNameTypes(fs *token.FileSet, fl *ast.FieldList) (dfs []DiffFragment) {
	for _, f := range fl.List {
		if len(f.Names) > 0 {
			for _, name := range f.Names {
				dfs = append(dfs, &Fragment{tp: DF_PAIR,
					Parts: []DiffFragment{newStringFrag(name.String(), 100),
						newTypeDef(fs, f.Type)}})
			} // for name
		} else {
			// embedding
			dfs = append(dfs, &Fragment{tp: DF_PAIR,
				Parts: []DiffFragment{newStringFrag("", 50),
					newTypeDef(fs, f.Type)}})
		} // else
	} // for f

	return dfs
}

func newTypeDef(fs *token.FileSet, def ast.Expr) DiffFragment {
	switch d := def.(type) {
	case *ast.StructType:
		return &Fragment{tp: DF_STRUCT, Parts: newNameTypes(fs, d.Fields)}

	case *ast.InterfaceType:
		return &Fragment{tp: DF_INTERFACE, Parts: newNameTypes(fs, d.Methods)}

	case *ast.StarExpr:
		return &Fragment{tp: DF_STAR, Parts: []DiffFragment{newTypeDef(fs, d.X)}}
	} // switch

	var src bytes.Buffer
	(&printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}).Fprint(&src, fs, def)
	return &StringFrag{weight: 50, source: src.String()}
}

func newTypeStmtInfo(fs *token.FileSet, name string, def ast.Expr) *Fragment {
	var f Fragment

	f.tp = DF_TYPE
	f.Parts = []DiffFragment{
		newStringFrag(name, 100),
		newTypeDef(fs, def)}

	return &f
}

func newExpDef(fs *token.FileSet, def ast.Expr) DiffFragment {
	//ast.Print(fs, def)
	var src bytes.Buffer
	(&printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}).Fprint(&src, fs, def)
	return &StringFrag{weight: 100, source: src.String()}
}

func newVarSpecs(fs *token.FileSet, specs []ast.Spec) (dfs []DiffFragment) {
	for _, spec := range specs {
		f := &Fragment{tp: DF_VAR_LINE}

		names := &Fragment{tp: DF_NAMES}
		sp := spec.(*ast.ValueSpec)
		for _, name := range sp.Names {
			names.Parts = append(names.Parts, &StringFrag{weight: 100,
				source: fmt.Sprint(name)})
		}
		f.Parts = append(f.Parts, names)

		if sp.Type != nil {
			f.Parts = append(f.Parts, newTypeDef(fs, sp.Type))
		} else {
			f.Parts = append(f.Parts, (*Fragment)(nil))
		} // else

		values := &Fragment{tp: DF_VALUES}
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

func newBlockDecl(fs *token.FileSet, blk *ast.BlockStmt) (f *Fragment) {
	f = &Fragment{tp: DF_BLOCK}
	lines := blockToLines(fs, blk)
	for _, line := range lines {
		f.Parts = append(f.Parts, &StringFrag{weight: 100, source: line})
	} // for line

	return f
}

func newFuncDecl(fs *token.FileSet, d *ast.FuncDecl) (f *Fragment) {
	f = &Fragment{tp: DF_FUNC}

	// recv
	if d.Recv != nil {
		f.Parts = append(f.Parts, newNameTypes(fs, d.Recv)...)
	} else {
		f.Parts = append(f.Parts, (*Fragment)(nil))
	} // else

	// name
	f.Parts = append(f.Parts, &StringFrag{weight: 200, source: fmt.Sprint(d.Name)})

	//  params
	if d.Type.Params != nil {
		f.Parts = append(f.Parts, &Fragment{tp: DF_VALUES,
			Parts: newNameTypes(fs, d.Type.Params)})
	} else {
		f.Parts = append(f.Parts, (*Fragment)(nil))
	} // else

	// Results
	if d.Type.Results != nil {
		f.Parts = append(f.Parts, &Fragment{tp: DF_RESULTS, Parts: newNameTypes(fs, d.Type.Results)})
	} else {
		f.Parts = append(f.Parts, (*Fragment)(nil))
	} // else

	// body
	if d.Body != nil {
		f.Parts = append(f.Parts, newBlockDecl(fs, d.Body))
	} else {
		f.Parts = append(f.Parts, (*Fragment)(nil))
	} // else
	return f
}

type FileInfo struct {
	f     *ast.File
	fs    *token.FileSet
	types *Fragment
	vars  *Fragment
	funcs *Fragment
}

func (info *FileInfo) collect() {
	info.types = &Fragment{}
	info.vars = &Fragment{}
	info.funcs = &Fragment{}

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
				v := &Fragment{tp: DF_CONST, Parts: newVarSpecs(info.fs, d.Specs)}
				info.vars.Parts = append(info.vars.Parts, v)
			case token.VAR:
				//ast.Print(info.fs, d)
				vss := newVarSpecs(info.fs, d.Specs)
				for _, vs := range vss {
					info.vars.Parts = append(info.vars.Parts, &Fragment{tp: DF_VAR, Parts: []DiffFragment{vs}})
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

func Parse(fn string, src interface{}) (*FileInfo, error) {
	if fn == "/dev/null" {
		return &FileInfo{
			f:     &ast.File{},
			types: &Fragment{},
			vars:  &Fragment{},
			funcs: &Fragment{},
		}, nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fn, src, 0)
	if err != nil {
		return nil, err
	} // if

	info := &FileInfo{f: f, fs: fset}
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

func ShowDelWholeLine(line string) {
	changeColor(del_COLOR, false, ct.None, false)
	fmt.Fprintln(out, "===", line)
	resetColor()
}
func ShowDelLine(line string) {
	changeColor(del_COLOR, false, ct.None, false)
	fmt.Fprintln(out, "---", line)
	resetColor()
}
func ShowColorDelLine(line, lcs string) {
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

func ShowDelLines(lines []string, gapLines int) {
	if len(lines) <= gapLines*2+1 {
		for _, line := range lines {
			ShowDelLine(line)
		} // for line
		return
	} // if

	for i, line := range lines {
		if i < gapLines || i >= len(lines)-gapLines {
			ShowDelLine(line)
		} // if
		if i == gapLines {
			ShowDelWholeLine(fmt.Sprintf("    ... (%d lines)", len(lines)-gapLines*2))
		} // if
	} // for i
}

func ShowInsLine(line string) {
	changeColor(ins_COLOR, false, ct.None, false)
	fmt.Fprintln(out, "+++", line)
	resetColor()
}
func ShowColorInsLine(line, lcs string) {
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

func ShowInsWholeLine(line string) {
	changeColor(ins_COLOR, false, ct.None, false)
	fmt.Fprintln(out, "###", line)
	resetColor()
}

func ShowInsLines(lines []string, gapLines int) {
	if len(lines) <= gapLines*2+1 {
		for _, line := range lines {
			ShowInsLine(line)
		} // for line
		return
	} // if

	for i, line := range lines {
		if i < gapLines || i >= len(lines)-gapLines {
			ShowInsLine(line)
		} // if
		if i == gapLines {
			ShowInsWholeLine(fmt.Sprintf("    ... (%d lines)", len(lines)-gapLines*2))
		} // if
	} // for i
}

func ShowDelTokens(del []string, mat []int, ins []string) {
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

func ShowInsTokens(ins []string, mat []int, del []string) {
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

func ShowDiffLine(del, ins string) {
	delT, insT := tm.LineToTokens(del), tm.LineToTokens(ins)
	matA, matB := tm.MatchTokens(delT, insT)

	ShowDelTokens(delT, matA, insT)
	ShowInsTokens(insT, matB, delT)
}

func DiffLineSet(orgLines, newLines []string, format string) {
	sort.Strings(orgLines)
	sort.Strings(newLines)

	_, matA, matB := ed.EditDistanceFFull(len(orgLines), len(newLines), func(iA, iB int) int {
		return tm.DiffOfStrings(orgLines[iA], newLines[iB], 4000)
	}, ed.ConstCost(1000), ed.ConstCost(1000))

	for i, j := 0, 0; i < len(orgLines) || j < len(newLines); {
		switch {
		case j >= len(newLines) || i < len(orgLines) && matA[i] < 0:
			ShowDelLine(fmt.Sprintf(format, orgLines[i]))
			i++
		case i >= len(orgLines) || j < len(newLines) && matB[j] < 0:
			ShowInsLine(fmt.Sprintf(format, newLines[j]))
			j++
		default:
			if strings.TrimSpace(orgLines[i]) != strings.TrimSpace(newLines[j]) {
				ShowDiffLine(fmt.Sprintf(format, orgLines[i]), fmt.Sprintf(format, newLines[j]))
			} // if
			i++
			j++
		}
	} // for i, j
}

type LineOutputer interface {
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
	ShowInsLine(line)
}

func (lo *lineOutput) outputDel(line string) {
	lo.end()
	ShowDelLine(line)
}

func (lo *lineOutput) outputChange(del, ins string) {
	lo.end()
	ShowDiffLine(del, ins)
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

func DiffLinesTo(orgLines, newLines []string, format string, lo LineOutputer) int {
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
func DiffLines(orgLines, newLines []string, format string) int {
	return DiffLinesTo(orgLines, newLines, format, &lineOutput{})
}

/*
   Diff Package
*/
func DiffPackage(orgInfo, newInfo *FileInfo) {
	orgName := orgInfo.f.Name.String()
	newName := newInfo.f.Name.String()
	if orgName != newName {
		ShowDiffLine("package "+orgName, "package "+newName)
	} //  if
}

/*
   Diff Imports
*/
func extractImports(info *FileInfo) []string {
	imports := make([]string, 0, len(info.f.Imports))
	for _, imp := range info.f.Imports {
		imports = append(imports, imp.Path.Value)
	} // for imp

	return imports
}

func DiffImports(orgInfo, newInfo *FileInfo) {
	orgImports := extractImports(orgInfo)
	newImports := extractImports(newInfo)

	DiffLineSet(orgImports, newImports, `import %s`)
}

/*
   Diff Types
*/
func DiffTypes(orgInfo, newInfo *FileInfo) {
	mat, _, matA, matB := GreedyMatch(len(orgInfo.types.Parts), len(newInfo.types.Parts), func(iA, iB int) int {
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
			ShowDelWholeLine(orgInfo.types.Parts[i].oneLine())
		} else {
			for ; j0 < j; j0++ {
				if matB[j0] < 0 {
					ShowInsWholeLine(newInfo.types.Parts[j0].oneLine())
				}
			}

			if mat[i][j] > 0 {
				orgInfo.types.Parts[i].showDiff(newInfo.types.Parts[j])
			} //  if
		} // else
	} // for i

	for ; j0 < len(matB); j0++ {
		if matB[j0] < 0 {
			ShowInsWholeLine(newInfo.types.Parts[j0].oneLine())
		}
	}
}

func DiffVars(orgInfo, newInfo *FileInfo) {
	mat, _, matA, matB := GreedyMatch(len(orgInfo.vars.Parts), len(newInfo.vars.Parts), func(iA, iB int) int {
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
			ShowDelLines(orgInfo.vars.Parts[i].sourceLines(""), 2)
			// fmt.Println()
		} else {
			for ; j0 < j; j0++ {
				if matB[j0] < 0 {
					ShowInsLines(newInfo.vars.Parts[j0].sourceLines(""), 2)
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
			ShowInsLines(newInfo.vars.Parts[j0].sourceLines(""), 2)
		} // if
	}
}

func DiffFuncs(orgInfo, newInfo *FileInfo) {
	mat, _, matA, matB := GreedyMatch(len(orgInfo.funcs.Parts), len(newInfo.funcs.Parts), func(iA, iB int) int {
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
			ShowDelWholeLine(orgInfo.funcs.Parts[i].oneLine())
		} else {
			for ; j0 < j; j0++ {
				if matB[j0] < 0 {
					ShowInsWholeLine(newInfo.funcs.Parts[j0].oneLine())
				}
			}
			if mat[i][j] > 0 {
				orgInfo.funcs.Parts[i].showDiff(newInfo.funcs.Parts[j])
			} //  if
		} // else
	} // for i

	for ; j0 < len(matB); j0++ {
		if matB[j0] < 0 {
			ShowInsWholeLine(newInfo.funcs.Parts[j0].oneLine())
		}
	}
}

func Diff(orgInfo, newInfo *FileInfo) {
	DiffPackage(orgInfo, newInfo)
	DiffImports(orgInfo, newInfo)
	DiffTypes(orgInfo, newInfo)
	DiffVars(orgInfo, newInfo)
	DiffFuncs(orgInfo, newInfo)
}

func readLines(fn villa.Path) []string {
	bts, err := fn.ReadFile()

	if err != nil {
		return nil
	}

	return strings.Split(string(bts), "\n")
}

type Options struct {
	NoColor bool
}

var (
	out      io.Writer = os.Stdout
	gOptions Options
)

// ExecWriter prints the difference between two Go files to stdout.
func Exec(orgFn, newFn string, options Options) {
	out = os.Stdout
	gOptions = options

	fmt.Printf("Difference between %s and %s ...\n", orgFn, newFn)

	orgInfo, err1 := Parse(orgFn, nil)
	newInfo, err2 := Parse(newFn, nil)

	if err1 != nil || err2 != nil {
		orgLines := readLines(villa.Path(orgFn))
		newLines := readLines(villa.Path(newFn))

		DiffLines(orgLines, newLines, "%s")
		return
	}

	Diff(orgInfo, newInfo)
}

// ExecWriter prints the difference between two parsed Go files into w.
func ExecWriter(w io.Writer, fset0 *token.FileSet, file0 *ast.File, fset1 *token.FileSet, file1 *ast.File) {
	out = w
	gOptions = Options{NoColor: true}

	orgInfo := &FileInfo{f: file0, fs: fset0}
	orgInfo.collect()
	newInfo := &FileInfo{f: file1, fs: fset1}
	newInfo.collect()

	Diff(orgInfo, newInfo)
}
