package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/freddieptf/cueform/encoding/xlsform"
)

type decoderCmd struct {
	flag *flag.FlagSet
	out  *string
	pkg  *string
}

func newDecoderCmd() *decoderCmd {
	flagSet := flag.NewFlagSet("decoder", flag.ExitOnError)
	outPutDir := flagSet.String("out", "", "output directory, defaults to current dir")
	pkg := flagSet.String("pkg", "", `package that has the schema definitions`)
	return &decoderCmd{
		flag: flagSet,
		out:  outPutDir,
		pkg:  pkg,
	}
}

func (cmd *decoderCmd) runDecodeCmd(ctx context.Context, args []string) error {
	err := cmd.flag.Parse(args)
	if err != nil {
		return err
	}
	file := cmd.flag.Arg(0)
	fReader, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	if *cmd.pkg == "" {
		return errors.New("missing pkg")
	}
	decoder := xlsform.NewDecoder(*cmd.pkg)
	surveyBytes, err := decoder.Decode(fReader)
	if err != nil {
		log.Fatal(err)
	}
	if *cmd.out == "stdout" {
		fmt.Printf("%s\n", surveyBytes)
	} else {
		fileName := strings.TrimSuffix(filepath.Base(file), ".xlsx")
		if outputPath, err := writeFile(*cmd.out, fmt.Sprintf("%s.cue", fileName), surveyBytes); err != nil {
			log.Fatalf("err writing %s: %s", fileName, err)
		} else {
			fmt.Println(outputPath)
		}
	}
	return nil
}
