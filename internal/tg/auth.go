package tg

import "context"

// AuthService contains Telegram login operations.
type AuthService interface {
	SendCode(ctx context.Context, phone string) error
	SignInWithCode(ctx context.Context, phone, code string) error
	SignInWithPassword(ctx context.Context, password string) error
	StartQRLogin(ctx context.Context) (QRLogin, error)
	// WaitQRLogin polls until the QR code is scanned. When the server issues
	// a fresh token (previous one expired), onRefresh is called so the caller
	// can re-render the QR code.
	WaitQRLogin(ctx context.Context, initial QRLogin, onRefresh func(QRLogin)) error
}

// QRLogin is an app-owned QR login payload.
type QRLogin struct {
	URL   string
	token []byte // raw token bytes for change detection
}
