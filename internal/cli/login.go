package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/wbx/tgup/internal/app"
)

func runLogin(args []string, _ io.Writer, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintf(stderr, "unexpected login args: %s\n", strings.Join(args, " "))
		return 2
	}
	if err := app.Login(context.Background()); err != nil {
		fmt.Fprintf(stderr, "login failed: %v\n", err)
		return 1
	}
	return 0
}
