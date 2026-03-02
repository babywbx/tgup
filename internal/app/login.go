package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	tdtg "github.com/gotd/td/tg"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/babywbx/tgup/internal/config"
	"github.com/babywbx/tgup/internal/tg"
	"golang.org/x/term"
)

// LoginMethod specifies which authentication method to use.
type LoginMethod int

const (
	LoginMethodCode LoginMethod = iota
	LoginMethodQR
)

// LoginOptions holds parameters for the login command.
type LoginOptions struct {
	Method LoginMethod
	Phone  string
	Stdout io.Writer
	Stderr io.Writer
}

// Login authenticates with Telegram and saves the session.
func Login(configPath string, cli config.Overlay, opts LoginOptions) error {
	cfg, err := config.ResolveTelegramOnly(configPath, cli)
	if err != nil {
		return fmt.Errorf("resolve config: %w", err)
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	tgCfg := tg.GotdConfig{
		AppID:       cfg.Telegram.APIID,
		AppHash:     cfg.Telegram.APIHash,
		SessionPath: cfg.Telegram.SessionPath,
	}

	ctx := context.Background()
	authorized, err := tg.IsSessionAuthorized(ctx, tgCfg)
	if err != nil {
		return fmt.Errorf("check auth: %w", err)
	}
	if authorized {
		fmt.Fprintf(stdout, "already authorized: session=%s\n", cfg.Telegram.SessionPath)
		return nil
	}

	sessionPath := cfg.Telegram.SessionPath

	switch opts.Method {
	case LoginMethodCode:
		return loginWithCode(ctx, tgCfg, opts, stdout, sessionPath)
	case LoginMethodQR:
		return loginWithQR(ctx, tgCfg, stdout, sessionPath)
	default:
		return fmt.Errorf("unknown login method: %d", opts.Method)
	}
}

func loginWithCode(ctx context.Context, tgCfg tg.GotdConfig, opts LoginOptions, stdout io.Writer, sessionPath string) error {
	phone := opts.Phone
	if phone == "" {
		var err error
		phone, err = promptLine("phone (+123...): ")
		if err != nil {
			return fmt.Errorf("read phone: %w", err)
		}
	}
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return fmt.Errorf("phone number is required")
	}

	if err := tg.LoginWithCode(ctx, tgCfg, tg.CodeLoginOptions{
		Phone: phone,
		Code: func(_ context.Context, _ *tdtg.AuthSentCode) (string, error) {
			code, err := promptLine("code: ")
			if err != nil {
				return "", fmt.Errorf("read code: %w", err)
			}
			return strings.TrimSpace(code), nil
		},
		Password: func(context.Context) (string, error) {
			pw, err := promptPassword("2FA password (input hidden): ")
			if err != nil {
				return "", fmt.Errorf("read password: %w", err)
			}
			return pw, nil
		},
	}); err != nil {
		return fmt.Errorf("sign in: %w", err)
	}

	fmt.Fprintf(stdout, "ok: session=%s\n", sessionPath)
	return nil
}

func loginWithQR(ctx context.Context, tgCfg tg.GotdConfig, stdout io.Writer, sessionPath string) error {
	printed := false
	if err := tg.LoginWithQR(ctx, tgCfg, tg.QRLoginOptions{
		Show: func(_ context.Context, url string) error {
			if printed {
				fmt.Fprintln(stdout, "\nQR code expired, refreshing...")
			}
			printed = true
			printQR(stdout, url)
			fmt.Fprintln(stdout, "waiting for QR login...")
			return nil
		},
		Password: func(context.Context) (string, error) {
			pw, err := promptPassword("2FA password (input hidden): ")
			if err != nil {
				return "", fmt.Errorf("read password: %w", err)
			}
			return pw, nil
		},
	}); err != nil {
		return fmt.Errorf("QR login: %w", err)
	}

	fmt.Fprintf(stdout, "ok: session=%s\n", sessionPath)
	return nil
}

func printQR(w io.Writer, url string) {
	q, err := qrcode.New(url, qrcode.Medium)
	if err == nil {
		fmt.Fprintln(w, q.ToSmallString(false))
		return
	}
	// Fallback: redact the token before printing.
	redacted := redactQRURL(url)
	fmt.Fprintf(w, "QR login URL: %s\n", redacted)
	fmt.Fprintln(w, "(scan this URL with a QR code generator app)")
}

func redactQRURL(url string) string {
	const prefix = "tg://login?token="
	if strings.HasPrefix(url, prefix) {
		token := url[len(prefix):]
		if len(token) > 8 {
			return prefix + token[:4] + "..." + token[len(token)-4:]
		}
		return prefix + "***"
	}
	return "<redacted>"
}

func promptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		pw, err := term.ReadPassword(fd)
		fmt.Println() // newline after hidden input
		return string(pw), err
	}
	// Non-terminal fallback.
	return promptLine("")
}
