package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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

	client := tg.NewGotdClient(tg.GotdConfig{
		AppID:       cfg.Telegram.APIID,
		AppHash:     cfg.Telegram.APIHash,
		SessionPath: cfg.Telegram.SessionPath,
	})

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Close(ctx)

	if client.IsAuthenticated(ctx) {
		fmt.Fprintf(stdout, "already authorized: session=%s\n", cfg.Telegram.SessionPath)
		return nil
	}

	switch opts.Method {
	case LoginMethodCode:
		return loginWithCode(ctx, client, opts, stdout)
	case LoginMethodQR:
		return loginWithQR(ctx, client, stdout)
	default:
		return fmt.Errorf("unknown login method: %d", opts.Method)
	}
}

func loginWithCode(ctx context.Context, client *tg.GotdClient, opts LoginOptions, stdout io.Writer) error {
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

	if err := client.SendCode(ctx, phone); err != nil {
		return fmt.Errorf("send code: %w", err)
	}

	code, err := promptLine("code: ")
	if err != nil {
		return fmt.Errorf("read code: %w", err)
	}
	code = strings.TrimSpace(code)

	err = client.SignInWithCode(ctx, phone, code)
	if err != nil {
		var pwdErr *tg.PasswordRequiredError
		if !errors.As(err, &pwdErr) {
			return fmt.Errorf("sign in: %w", err)
		}
		// 2FA password required.
		password, err := promptPassword("2FA password (input hidden): ")
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		if err := client.SignInWithPassword(ctx, password); err != nil {
			return fmt.Errorf("sign in with password: %w", err)
		}
	}

	fmt.Fprintln(stdout, "ok")
	return nil
}

func loginWithQR(ctx context.Context, client *tg.GotdClient, stdout io.Writer) error {
	qr, err := client.StartQRLogin(ctx)
	if err != nil {
		return fmt.Errorf("start QR login: %w", err)
	}

	printQR(stdout, qr.URL)
	fmt.Fprintln(stdout, "waiting for QR login...")

	err = client.WaitQRLogin(ctx, qr)
	if err != nil {
		var pwdErr *tg.PasswordRequiredError
		if !errors.As(err, &pwdErr) {
			return fmt.Errorf("QR login: %w", err)
		}
		password, err := promptPassword("2FA password (input hidden): ")
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		if err := client.SignInWithPassword(ctx, password); err != nil {
			return fmt.Errorf("sign in with password: %w", err)
		}
	}

	fmt.Fprintln(stdout, "ok")
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
	// Redact the token parameter value in tg://login?token=...
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
