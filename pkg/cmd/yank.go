package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"slices"

	"github.com/freddieptf/cueform/pkg/labels"
)

var (
	yankResources = []string{"labels"}
)

type yankCmd struct {
	flag   *flag.FlagSet
	dryRun *bool
}

func newYankCmd() *yankCmd {
	flagSet := flag.NewFlagSet("yank", flag.ExitOnError)
	dryRun := flagSet.Bool("dry", false, "dry run mode, only print out the changes")
	defaultUsage := flagSet.Usage
	flagSet.Usage = func() {
		defaultUsage()
		fmt.Println(`supported yankable resources: labels`)
	}
	return &yankCmd{flag: flagSet, dryRun: dryRun}
}

func (cmd *yankCmd) runYankCmd(ctx context.Context, args []string) error {
	err := cmd.flag.Parse(args)
	if err != nil {
		return err
	}
	if len(cmd.flag.Args()) < 2 {
		return fmt.Errorf("not enough arguments")
	}
	resource := cmd.flag.Arg(0)
	if slices.Index(yankResources, resource) == -1 {
		return fmt.Errorf("unsupported action: %s", resource)
	}
	switch resource {
	case "labels":
		result, err := labels.ExtractLabels(cmd.flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		if *cmd.dryRun {
			fmt.Println(string(result.Labels))
			fmt.Println(string(result.Form))
			return nil
		}
		parentPath := filepath.Dir(cmd.flag.Arg(1))
		_, err = writeFile(parentPath, "labels.cue", result.Labels)
		if err != nil {
			log.Fatal(err)
		}
		_, err = writeFile(parentPath, filepath.Base(cmd.flag.Arg(1)), result.Form)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}
