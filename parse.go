package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"slices"
	"strings"
)

var (
	rComment = regexp.MustCompile(`^//.*?@(?i:inject_field):\s*(.*)$`)
)

type textArea struct {
	Start        int
	End          int
	CommentStart int
	CommentEnd   int
	InjectField  string
}

func parseFile(inputPath string) (areas []*textArea, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, g := range f.Comments {
		for _, comment := range g.List {
			field := fieldFromComment(comment.Text)
			if field == "" {
				continue
			}

			area := &textArea{
				CommentStart: int(comment.Pos()),
				CommentEnd:   int(comment.End()),
				InjectField:  field,
			}
			areas = append(areas, area)
		}
	}

	if len(areas) == 0 {
		return
	}

	var lastTypePosEnd int
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		var typeSpec *ast.TypeSpec
		for _, spec := range genDecl.Specs {
			if ts, tsOK := spec.(*ast.TypeSpec); tsOK {
				typeSpec = ts
				break
			}
		}

		// skip if can't get type spec
		if typeSpec == nil {
			continue
		}

		structDecl, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		insertPos := int(structDecl.Fields.End() - 1)
		endPos := int(typeSpec.End())

		findMember := func(inject string) bool {
			fieldIndex := strings.Index(inject, " ")
			fieldName := inject[:fieldIndex]
			for _, v := range structDecl.Fields.List {
				for _, name := range v.Names {
					if fieldName == name.Name {
						return true
					}
				}
			}
			return false
		}

		for _, area := range areas {
			if lastTypePosEnd < area.CommentStart && area.CommentEnd < insertPos {
				// prevent duplicate inject
				if findMember(area.InjectField) {
					continue
				}
				area.Start = insertPos - 1
				area.End = endPos
			}
		}
		lastTypePosEnd = endPos
	}
	areas = slices.DeleteFunc(areas, func(area *textArea) bool {
		return area.Start == 0
	})
	//if len(areas) > 0 {
	//	log.Printf("parsed file %q, number of fields to inject is %d", filepath.Base(inputPath), len(areas))
	//}
	return
}

func writeFile(inputPath string, areas []*textArea) (err error) {
	contents, err := os.ReadFile(inputPath)
	if err != nil {
		return
	}

	for i := range areas {
		area := areas[len(areas)-i-1]
		contents = injectField(contents, area)
	}
	if err = os.WriteFile(inputPath, contents, 0644); err != nil {
		return
	}
	return
}

func fieldFromComment(comment string) string {
	match := rComment.FindStringSubmatch(comment)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func injectField(contents []byte, area *textArea) (injected []byte) {
	injected = contents[:area.Start]
	old := make([]byte, len(contents)-area.Start)
	copy(old, contents[area.Start:])
	wrap := "\t" + area.InjectField + "\n"
	injected = append(injected, wrap...)
	injected = append(injected, old...)
	return
}
