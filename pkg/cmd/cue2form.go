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

type encodeCmd struct {
	flag *flag.FlagSet
	out  *string
	to   *string
}

func newEncoderCmd() *encodeCmd {
	flagSet := flag.NewFlagSet("encoder", flag.ExitOnError)
	outPutDir := flagSet.String("out", "", "output directory")
	to := flagSet.String("to", "xlsform", `expected output format`)
	return &encodeCmd{
		flag: flagSet,
		out:  outPutDir,
		to:   to,
	}
}

func (cmd *encodeCmd) runEncodeCmd(ctx context.Context, args []string) error {
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

type decodeCmd struct {
	flag *flag.FlagSet
	out  *string
	pkg  *string
}

func newDecodeCmd() *decodeCmd {
	flagSet := flag.NewFlagSet("decoder", flag.ExitOnError)
	outPutDir := flagSet.String("out", "", "output directory, defaults to current dir")
	pkg := flagSet.String("pkg", "", `package that has the schema definitions`)
	return &decodeCmd{
		flag: flagSet,
		out:  outPutDir,
		pkg:  pkg,
	}
}

func (cmd *decodeCmd) runDecodeCmd(ctx context.Context, args []string) error {
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

type cue2formCmd struct {
	flag *flag.FlagSet
}

func newCue2FormCmd() *cue2formCmd {
	flagSet := flag.NewFlagSet("cue2form", flag.ExitOnError)
	flagSet.String("out", "", "output directory, defaults to current dir")
	flagSet.String("pkg", "", `package that has the schema definitions`)
	flagSet.String("to", "xlsform", `expected output format`)
	return &cue2formCmd{flag: flagSet}
}

func (cmd *cue2formCmd) runCue2Form(ctx context.Context, args []string) {
	cmd.flag.Usage = func() {}
	err := cmd.flag.Parse(args)
	if err != nil {
		log.Fatal(err)
	}
	if len(cmd.flag.Args()) <= 0 {
		log.Fatal("no file args")
	}
	file := cmd.flag.Arg(0)
	if strings.HasSuffix(file, ".cue") {
		cmd := newEncoderCmd()
		err := cmd.runEncodeCmd(ctx, args)
		if err != nil {
			log.Println(err)
			cmd.flag.Usage()
		}
	} else if strings.HasSuffix(file, ".xlsx") {
		cmd := newDecodeCmd()
		err := cmd.runDecodeCmd(ctx, args)
		if err != nil {
			log.Println(err)
			cmd.flag.Usage()
		}
	} else {
		log.Fatal("expecting xlsx or cue file")
	}
}

func ExecCue2FormCmd(ctx context.Context) {
	cmd := newCue2FormCmd()
	cmd.runCue2Form(ctx, os.Args[1:])
}
