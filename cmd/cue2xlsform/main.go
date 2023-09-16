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
	module    string
	schemaPkg string
)

func writeFile(parentDir, file string, b []byte) error {
	return ioutil.WriteFile(filepath.Join(parentDir, file), b, fs.ModePerm)
}

func main() {
	flag.StringVar(&dir, "dir", "", "path to form directory with cue files")
	flag.StringVar(&file, "file", "", "path to xls form")
	flag.StringVar(&outputDir, "out", "", "output directory")
	flag.StringVar(&module, "module", "", "module directory")
	flag.StringVar(&schemaPkg, "pkg", "", "path of the schema relative to the module, only useful when decoding xls forms to cue")

	flag.Parse()
	if dir != "" {
		encoder := xlsform.NewEncoder(dir)
		encoder.UseModule(module)
		f, err := encoder.Encode()
		if err != nil {
			log.Fatal(err)
		}
		if outputDir == "stdout" {
			fmt.Printf("%s", f.Bytes())
		} else {
			outPutPath := fmt.Sprintf("%s.xlsx", filepath.Base(dir))
			if err = writeFile(outputDir, outPutPath, f.Bytes()); err != nil {
				log.Fatalf("err writing %s: %s", outPutPath, err)
			}
			fmt.Println(outPutPath)
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
		if module != "" {
			err = decoder.UseSchema(module, schemaPkg)
			if err != nil {
				log.Fatalf("err during schema init: %s", err)
			}
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
			fmt.Println(outputDir)
		}
		return
	}
	if dir == "" && file == "" {
		flag.Usage()
	}
}
