package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
)

func ExecCueform(ctx context.Context) {
	encoderCmd := newEncoderCmd()
	decoderCmd := newDecodeCmd()
	yankCmd := newYankCmd()
	printUsage := func() {
		encoderCmd.flag.Usage()
		fmt.Println()
		decoderCmd.flag.Usage()
		fmt.Println()
		yankCmd.flag.Usage()
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
	case "yank":
		err := yankCmd.runYankCmd(ctx, os.Args[2:])
		if err != nil {
			log.Println(err)
			yankCmd.flag.Usage()
		}

	default:
		printUsage()
	}
}
