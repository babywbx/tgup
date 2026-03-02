package tg

import (
	"context"

	tdtg "github.com/gotd/td/tg"
)

// CodePrompt asks the user for the Telegram verification code.
type CodePrompt func(ctx context.Context, sentCode *tdtg.AuthSentCode) (string, error)

// PasswordPrompt asks the user for the 2FA password.
type PasswordPrompt func(ctx context.Context) (string, error)

// QRShowFunc renders a QR login URL for the user.
// Called each time a new token is exported (initial + refreshes on expiry).
type QRShowFunc func(ctx context.Context, url string) error

// CodeLoginOptions configures phone+code login.
type CodeLoginOptions struct {
	Phone    string
	Code     CodePrompt
	Password PasswordPrompt // optional; used only when 2FA is required
}

// QRLoginOptions configures QR login.
type QRLoginOptions struct {
	Show     QRShowFunc
	Password PasswordPrompt // optional; used only when 2FA is required
}
