package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/freddieptf/cueform/encoding/xlsform"
)

var (
	dir       string
	file      string
	outputDir string
)

func writeFile(parentDir, file string, b []byte) error {
	return ioutil.WriteFile(filepath.Join(parentDir, file), b, fs.ModePerm)
}

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
		fReader, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}
		decoder, err := xlsform.NewDecoder(fReader)
		if err != nil {
			log.Fatalf("err initializing decoder: %s", err)
		}
		result := decoder.Decode()
		if result.Err != nil {
			log.Fatal(err)
		}
		if outputDir == "stdout" {
			fmt.Printf("%s\n\n===============================\n\n%s", result.Choices, result.Survey)
		} else {
			if outputDir == "" {
				outputDir = filepath.Join("./", strings.TrimSuffix(filepath.Base(file), ".xlsx"))
				if err = os.Mkdir(outputDir, fs.ModePerm); err != nil && !errors.Is(err, os.ErrExist) {
					log.Fatalf("could not create output dir: %s", err)
				}
			}
			if err = writeFile(outputDir, "choices.cue", result.Choices); err != nil {
				log.Fatalf("err writing choices.cue: %s", err)
			}
			if err = writeFile(outputDir, "survey.cue", result.Survey); err != nil {
				log.Fatalf("err writing survey.cue: %s", err)
			}
			fmt.Println("Done, files written to", outputDir)
		}
		return
	}
	if dir == "" && file == "" {
		flag.Usage()
	}
}
