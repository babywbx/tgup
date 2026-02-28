package tg

import "context"

// AuthService contains Telegram login operations.
type AuthService interface {
	SendCode(ctx context.Context, phone string) error
	SignInWithCode(ctx context.Context, phone, code string) error
	SignInWithPassword(ctx context.Context, password string) error
	StartQRLogin(ctx context.Context) (QRLogin, error)
	WaitQRLogin(ctx context.Context, qr QRLogin) error
}

// QRLogin is an app-owned QR login payload.
type QRLogin struct {
	URL string
}
