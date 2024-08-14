package main

import (
	"bytes"
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
	InsertPos    int
	CommentStart int
	CommentEnd   int
	InjectField  string
}

func parseFile(inputPath string) ([]*textArea, error) {
	f, err := parser.ParseFile(token.NewFileSet(), inputPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var areas []*textArea
	for _, g := range f.Comments {
		for _, comment := range g.List {
			tag := parseInjectField(comment.Text)
			if tag == "" {
				continue
			}

			area := &textArea{
				CommentStart: int(comment.Pos()),
				CommentEnd:   int(comment.End()),
				InjectField:  tag,
			}
			areas = append(areas, area)
		}
	}

	if len(areas) == 0 {
		return areas, nil
	}

	var lastTypePosEnd int
	for _, decl := range f.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		var s *ast.TypeSpec
		var st *ast.StructType
		for _, spec := range d.Specs {
			s, ok = spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok = s.Type.(*ast.StructType)
			if !ok {
				continue
			}

			findField := func(inject string) bool {
				for _, v := range st.Fields.List {
					for _, name := range v.Names {
						if strings.Contains(inject, name.Name) {
							return true
						}
					}
				}
				return false
			}

			insertPos := int(st.Fields.End() - 1)
			for _, area := range areas {
				if lastTypePosEnd < area.CommentStart && area.CommentEnd < insertPos {
					// prevent duplicate inject
					if findField(area.InjectField) {
						continue
					}
					area.InsertPos = insertPos
				}
			}
			lastTypePosEnd = insertPos
		}
	}

	areas = slices.DeleteFunc(areas, func(area *textArea) bool {
		return area.InsertPos == 0
	})
	return areas, nil
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

func parseInjectField(comment string) string {
	match := rComment.FindStringSubmatch(comment)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

func injectField(contents []byte, area *textArea) (injected []byte) {
	injected = bytes.Clone(contents[:area.InsertPos-1])
	tail := bytes.Clone(contents[area.InsertPos-1:])
	wrap := "\t" + area.InjectField + "\n"
	injected = append(injected, wrap...)
	injected = append(injected, tail...)
	return
}
