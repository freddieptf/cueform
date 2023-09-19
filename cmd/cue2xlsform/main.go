package main

import (
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
	file      string
	outputDir string
	module    string
	schemaPkg string
)

func writeFile(parentDir, file string, b []byte) (string, error) {
	out := filepath.Join(parentDir, file)
	err := ioutil.WriteFile(out, b, fs.ModePerm)
	return out, err
}

func main() {
	flag.StringVar(&file, "file", "", "path to xls or cue file")
	flag.StringVar(&outputDir, "out", "", "output directory")
	flag.StringVar(&module, "module", "", "module directory")
	flag.StringVar(&schemaPkg, "pkg", "", "path of the schema relative to the module, only useful when decoding xls forms to cue")
	flag.Parse()

	if file == "" {
		flag.Usage()
		return
	}

	fileName := filepath.Base(file)

	if strings.HasSuffix(fileName, ".cue") {
		encoder := xlsform.NewEncoder(file)
		encoder.UseModule(module)
		f, err := encoder.Encode()
		if err != nil {
			log.Fatal(err)
		}
		if outputDir == "stdout" {
			fmt.Printf("%s", f.Bytes())
		} else {
			fileName := fmt.Sprintf("%s.xlsx", strings.TrimSuffix(fileName, ".cue"))
			if outputPath, err := writeFile(outputDir, fileName, f.Bytes()); err != nil {
				log.Fatalf("err writing %s: %s", outputPath, err)
			} else {
				fmt.Println(outputPath)
			}
		}
	} else if strings.HasSuffix(fileName, ".xlsx") {
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
		surveyBytes, err := decoder.Decode()
		if err != nil {
			log.Fatal(err)
		}
		if outputDir == "stdout" {
			fmt.Printf("%s\n", surveyBytes)
		} else {
			fileName := fmt.Sprintf("%s.cue", strings.TrimSuffix(fileName, ".xlsx"))
			if outputPath, err := writeFile(outputDir, fileName, surveyBytes); err != nil {
				log.Fatalf("err writing %s: %s", fileName, err)
			} else {
				fmt.Println(outputPath)
			}
		}
		return
	} else {
		fmt.Println("was expecting xls or cue file")
		flag.Usage()
	}
}
