package cmd

import (
	"context"
	"log"
	"os"
)

func ExecCueform(ctx context.Context) {
	encoderCmd := newEncoderCmd()
	decoderCmd := newDecodeCmd()
	printUsage := func() {
		encoderCmd.flag.Usage()
		decoderCmd.flag.Usage()
	}
	if len(os.Args) <= 1 {
		printUsage()
		return
	}
	switch os.Args[1] {
	case "encode":
		err := encoderCmd.runEncodeCmd(ctx, os.Args[2:])
		if err != nil {
			log.Println(err)
			encoderCmd.flag.Usage()
		}
	case "decode":
		err := decoderCmd.runDecodeCmd(ctx, os.Args[2:])
		if err != nil {
			log.Println(err)
			decoderCmd.flag.Usage()
		}
	default:
		printUsage()
	}
}
