package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/babywbx/tgup/internal/app"
	"github.com/babywbx/tgup/internal/config"
)

func runLogin(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string
	var useCode bool
	var useQR bool
	var phone string

	fs.StringVar(&configPath, "config", "", "path to config file")
	fs.BoolVar(&useCode, "code", false, "use phone+code login")
	fs.BoolVar(&useQR, "qr", false, "use QR code login")
	fs.StringVar(&phone, "phone", "", "phone number for --code login")

	apiID := &intValue{}
	apiHash := &stringValue{}
	sessionPath := &stringValue{}
	sessionPathAlias := &stringValue{}

	fs.Var(apiID, "api-id", "telegram api id")
	fs.Var(apiHash, "api-hash", "telegram api hash")
	fs.Var(sessionPath, "session", "session file path")
	fs.Var(sessionPathAlias, "session-path", "session file path (alias)")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "unexpected login args: %s\n", strings.Join(fs.Args(), " "))
		return 2
	}

	// --code and --qr are mutually exclusive and one is required.
	if useCode == useQR {
		if useCode {
			fmt.Fprintln(stderr, "conflicting flags: --code and --qr are mutually exclusive")
		} else {
			fmt.Fprintln(stderr, "one of --code or --qr is required")
		}
		return 2
	}

	if sessionPath.set && sessionPathAlias.set {
		fmt.Fprintln(stderr, "conflicting flags: --session and --session-path")
		return 2
	}
	if sessionPathAlias.set && !sessionPath.set {
		sessionPath = sessionPathAlias
	}

	cli := config.Overlay{}
	if apiID.set {
		cli.Telegram.APIID = apiID.ptr()
	}
	if apiHash.set {
		cli.Telegram.APIHash = apiHash.ptr()
	}
	if sessionPath.set {
		cli.Telegram.SessionPath = sessionPath.ptr()
	}

	method := app.LoginMethodCode
	if useQR {
		method = app.LoginMethodQR
	}

	if err := app.Login(configPath, cli, app.LoginOptions{
		Method: method,
		Phone:  phone,
		Stdout: stdout,
		Stderr: stderr,
	}); err != nil {
		fmt.Fprintf(stderr, "login failed: %v\n", err)
		return 1
	}
	return 0
}
