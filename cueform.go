package main

import (
	"log"

	"github.com/freddieptf/cueform/encoding/xlsform"
)

func main() {
	err := xlsform.Encode("sample/wash")
	if err != nil {
		log.Fatal(err)
	}
}
