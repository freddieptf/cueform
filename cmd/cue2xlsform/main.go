package main

import (
	"flag"
	"log"

	"github.com/freddieptf/cueform/encoding/xlsform"
)

var (
	dir       string
	file      string
	outputDir string
)

func main() {
	flag.StringVar(&dir, "dir", "", "path to form directory with cue files")
	flag.StringVar(&file, "file", "", "path to xls form")
	flag.StringVar(&outputDir, "out", "", "output directory")
	flag.Parse()
	if dir != "" {
		err := xlsform.Encode(outputDir, dir)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	if file != "" {
		err := xlsform.Decode(outputDir, file)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	if dir == "" && file == "" {
		flag.Usage()
	}
}
