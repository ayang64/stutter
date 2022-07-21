package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

type MainMain string

type Stutter struct {
	Symbol   string
	Package  string
	Position token.Position
}

func suggest(p string, s string) string {
	if idx := strings.Index(strings.ToLower(s), strings.ToLower(p)); idx != -1 {
		s = s[:idx] + s[idx+len(p):]
	}
	return p + "." + s
}

func (s Stutter) String() string {
	return fmt.Sprintf("%s: %q in %q: recommend %q", s.Position, s.Package, s.Symbol, suggest(s.Package, s.Symbol))
}

// const block for self test
var Foo string
var MainFoo string

// const block for self testing
const (
	Bar = iota + 200
	MainBar
)

type Visit struct {
	Stutter []Stutter
	Package string
	Fset    *token.FileSet
}

func (s *Visit) Visit(node ast.Node) ast.Visitor {
	contains := func(a, b string) bool {
		return strings.Contains(strings.ToLower(a), strings.ToLower(b))
	}

	switch v := node.(type) {
	case *ast.FuncDecl:
		if contains(v.Name.String(), s.Package) {
			s.Stutter = append(s.Stutter, Stutter{
				Symbol:   v.Name.String(),
				Package:  s.Package,
				Position: s.Fset.PositionFor(v.Pos(), true),
			})
		}

	case *ast.GenDecl:
		for _, spec := range v.Specs {
			switch d := spec.(type) {
			case *ast.TypeSpec:
				if contains(d.Name.String(), s.Package) {
					s.Stutter = append(s.Stutter, Stutter{
						Symbol:   d.Name.String(),
						Package:  s.Package,
						Position: s.Fset.PositionFor(d.Pos(), true),
					})
				}

			case *ast.ValueSpec:
				for _, name := range d.Names {
					if contains(name.String(), s.Package) {
						s.Stutter = append(s.Stutter, Stutter{
							Symbol:   name.String(),
							Package:  s.Package,
							Position: s.Fset.PositionFor(d.Pos(), true),
						})
					}
				}
			}
		}

	}
	return s
}

func main() {
	for _, p := range os.Args[1:] {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			log.Fatal(err)
		}

		visitors := map[string]*Visit{}
		for _, pkg := range pkgs {
			visitors[pkg.Name] = &Visit{Fset: fset, Package: pkg.Name}
			for _, file := range pkg.Files {
				ast.Walk(visitors[pkg.Name], file)
			}
		}

		for _, visitor := range visitors {
			for _, s := range visitor.Stutter {
				fmt.Printf("%s\n", s)
			}
		}
	}
}
