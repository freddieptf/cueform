package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/freddieptf/cueform/encoding/xlsform"
)

type encoderCmd struct {
	flag *flag.FlagSet
	out  *string
	to   *string
}

func newEncoderCmd() *encoderCmd {
	flagSet := flag.NewFlagSet("encoder", flag.ExitOnError)
	outPutDir := flagSet.String("out", "", "output directory")
	to := flagSet.String("to", "xlsform", `expected output format`)
	return &encoderCmd{
		flag: flagSet,
		out:  outPutDir,
		to:   to,
	}
}

func (cmd *encoderCmd) runEncodeCmd(ctx context.Context, args []string) error {
	err := cmd.flag.Parse(args)
	if err != nil {
		return err
	}
	file := cmd.flag.Arg(0)
	switch *cmd.to {
	case "xlsform":
		encoder := xlsform.NewEncoder()
		f, err := encoder.Encode(file)
		if err != nil {
			log.Fatal(err)
		}
		if *cmd.out == "stdout" {
			fmt.Printf("%s", f.Bytes())
		} else {
			fileName := strings.TrimSuffix(filepath.Base(file), ".cue")
			if outputPath, err := writeFile(*cmd.out, fmt.Sprintf("%s.xlsx", fileName), f.Bytes()); err != nil {
				log.Printf("err writing %s: %s", outputPath, err)
			} else {
				fmt.Println(outputPath)
			}
		}
	default:
		return fmt.Errorf("output format not supported: %s", *cmd.to)
	}
	return nil
}
