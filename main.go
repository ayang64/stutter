package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

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
	return fmt.Sprintf("%s: consider changing %q to %q", s.Position, s.Package+"."+s.Symbol, suggest(s.Package, s.Symbol))
}

type Visit struct {
	Stutter []Stutter
	Package string
	Fset    *token.FileSet
}

func (v *Visit) Append(symb string, pkg string, pos token.Position) {
	v.Stutter = append(v.Stutter, Stutter{
		Symbol:   symb,
		Package:  pkg,
		Position: pos,
	})
}

func (s *Visit) Visit(node ast.Node) ast.Visitor {
	// case insensitive string contains function.
	contains := func(a, b string) bool {
		return strings.Contains(strings.ToLower(a), strings.ToLower(b))
	}
	switch v := node.(type) {
	case *ast.FuncDecl:
		if contains(v.Name.String(), s.Package) {
			s.Append(v.Name.String(), s.Package, s.Fset.PositionFor(v.Pos(), true))
		}
	case *ast.GenDecl:
		for _, spec := range v.Specs {
			switch d := spec.(type) {
			case *ast.TypeSpec:
				if contains(d.Name.String(), s.Package) {
					s.Append(d.Name.String(), s.Package, s.Fset.PositionFor(d.Pos(), true))
				}
			case *ast.ValueSpec:
				for _, name := range d.Names {
					if contains(name.String(), s.Package) {
						s.Append(name.String(), s.Package, s.Fset.PositionFor(d.Pos(), true))
					}
				}
			}
		}
	}
	return s
}

func main() {
	sem := make(chan struct{}, runtime.NumCPU()*4)
	for _, p := range os.Args[1:] {
		sem <- struct{}{}
		p := p
		go func() {
			filepath.WalkDir(p, func(path string, d fs.DirEntry, e error) error {
				if !d.IsDir() {
					return nil
				}

				switch d.Name() {
				case "testdata", "vendor":
					return fmt.Errorf("skippig %q directory", d.Name())
				}

				fset := token.NewFileSet()
				pkgs, err := parser.ParseDir(fset, path, nil, parser.SkipObjectResolution)
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
				return nil
			})
			<-sem
		}()
	}

	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
}
