package main

import (
	"flag"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

func main() {
	var walkDir string
	flag.StringVar(&walkDir, "dir", ".", "specify directory want to inject custom traversal")
	flag.Parse()

	var files []string
	_ = filepath.Walk(walkDir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.Contains(info.Name(), ".pb.go") {
			files = append(files, path)
		}
		return nil
	})

	areas := make(map[string][]*textArea)
	for _, path := range files {
		result, err := parseFile(path)
		if err != nil {
			log.Fatal(err)
		}
		if len(result) == 0 {
			continue
		}
		areas[path] = result
	}

	for path, area := range areas {
		err := writeFile(path, area)
		if err != nil {
			log.Fatal(err)
		}
	}
}
