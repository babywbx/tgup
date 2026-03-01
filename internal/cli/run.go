package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/babywbx/tgup/internal/app"
)

func runRun(args []string, _ io.Writer, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintf(stderr, "unexpected run args: %s\n", strings.Join(args, " "))
		return 2
	}
	if err := app.Run(context.Background()); err != nil {
		fmt.Fprintf(stderr, "run failed: %v\n", err)
		return 1
	}
	return 0
}
