package cmd

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
)

type cue2formCmd struct {
	flag *flag.FlagSet
}

func newCue2FormCmd() *cue2formCmd {
	flagSet := flag.NewFlagSet("cue2form", flag.ExitOnError)
	flagSet.String("out", "", "output directory, defaults to current dir")
	flagSet.String("pkg", "", `package that has the schema definitions`)
	flagSet.String("to", "xlsform", `expected output format`)
	flagSet.Usage = func() {}
	return &cue2formCmd{flag: flagSet}
}

func (cmd *cue2formCmd) runCue2Form(ctx context.Context, args []string) {
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
		cmd := newDecoderCmd()
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
