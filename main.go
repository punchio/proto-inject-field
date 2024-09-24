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
	flag.StringVar(&walkDir, "input", ".", "specify directory want to inject custom traversal")
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

	var matches []string
	for _, path := range files {
		areas, err := parseFile(path)
		if err != nil {
			log.Fatal(err)
		}
		if len(areas) == 0 {
			continue
		}

		if err = writeFile(path, areas); err != nil {
			log.Fatal(err)
		}

		matches = append(matches, path)
	}

	if len(matches) > 0 {
		log.Printf("match files:%v", matches)
	}
}
