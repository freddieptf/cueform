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
	pkg       string

	moduleHelp = `path to module with the required schema definitions
example, assuming you cloned github.com/freddieptf/cueform to the current directory
	cue2xlsform --module ./cueform --file form.xlsx`
	pkgHelp = `package in module that has the schema definitions
example, if you're using schema/xlsform from github.com/freddieptf/cueform
	cue2xlsform --module ./cueform --pkg schema/xlsform --file form.xlsx
`
)

func writeFile(parentDir, file string, b []byte) (string, error) {
	out := filepath.Join(parentDir, file)
	err := ioutil.WriteFile(out, b, fs.ModePerm)
	return out, err
}

func main() {
	flag.StringVar(&file, "file", "", "path to xls or cue file")
	flag.StringVar(&outputDir, "out", "", "output directory")
	flag.StringVar(&module, "module", "", moduleHelp)
	flag.StringVar(&pkg, "pkg", "", pkgHelp)
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
		err = decoder.UseSchema(module, pkg)
		if err != nil {
			log.Fatalf("err during schema init: %s", err)
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
