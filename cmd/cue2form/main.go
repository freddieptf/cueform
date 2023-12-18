package main

import (
	"context"

	"github.com/freddieptf/cueform/pkg/cmd"
)

func main() {
	ctx := context.Background()
	cmd.ExecCue2FormCmd(ctx)
}
