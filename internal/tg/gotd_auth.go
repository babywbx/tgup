package tg

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	tdtg "github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// SendCode sends a verification code to the given phone number.
func (c *GotdClient) SendCode(ctx context.Context, phone string) error {
	api, err := c.getAPI()
	if err != nil {
		return err
	}

	sent, err := api.AuthSendCode(ctx, &tdtg.AuthSendCodeRequest{
		PhoneNumber: phone,
		APIID:       c.cfg.AppID,
		APIHash:     c.cfg.AppHash,
		Settings:    tdtg.CodeSettings{},
	})
	if err != nil {
		return mapGotdError(err)
	}

	switch s := sent.(type) {
	case *tdtg.AuthSentCode:
		c.phoneCodeHash = s.PhoneCodeHash
		return nil
	case *tdtg.AuthSentCodeSuccess:
		// Already authenticated (e.g. via Firebase).
		_ = s
		return nil
	default:
		return fmt.Errorf("unexpected send code response: %T", sent)
	}
}

// SignInWithCode signs in using a phone number and verification code.
func (c *GotdClient) SignInWithCode(ctx context.Context, phone, code string) error {
	if c.phoneCodeHash == "" {
		return fmt.Errorf("must call SendCode before SignInWithCode")
	}

	api, err := c.getAPI()
	if err != nil {
		return err
	}

	_, err = api.AuthSignIn(ctx, &tdtg.AuthSignInRequest{
		PhoneNumber:   phone,
		PhoneCodeHash: c.phoneCodeHash,
		PhoneCode:     code,
	})
	if err != nil {
		if tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
			return &PasswordRequiredError{}
		}
		return mapGotdError(err)
	}
	c.phoneCodeHash = "" // clear stale auth state
	return nil
}

// PasswordRequiredError signals that 2FA password is needed.
type PasswordRequiredError struct{}

func (e *PasswordRequiredError) Error() string {
	return "2FA password required"
}

// SignInWithPassword completes sign-in with 2FA password.
func (c *GotdClient) SignInWithPassword(ctx context.Context, password string) error {
	api, err := c.getAPI()
	if err != nil {
		return err
	}

	pwd, err := api.AccountGetPassword(ctx)
	if err != nil {
		return mapGotdError(err)
	}

	inputCheck, err := computeSRPAnswer(password, pwd)
	if err != nil {
		return fmt.Errorf("compute SRP: %w", err)
	}

	_, err = api.AuthCheckPassword(ctx, inputCheck)
	if err != nil {
		return mapGotdError(err)
	}
	return nil
}

// StartQRLogin begins the QR login flow and returns a login URL.
func (c *GotdClient) StartQRLogin(ctx context.Context) (QRLogin, error) {
	api, err := c.getAPI()
	if err != nil {
		return QRLogin{}, err
	}

	result, err := api.AuthExportLoginToken(ctx, &tdtg.AuthExportLoginTokenRequest{
		APIID:   c.cfg.AppID,
		APIHash: c.cfg.AppHash,
	})
	if err != nil {
		return QRLogin{}, mapGotdError(err)
	}

	switch r := result.(type) {
	case *tdtg.AuthLoginToken:
		url := "tg://login?token=" + base64.URLEncoding.EncodeToString(r.Token)
		return QRLogin{URL: url}, nil
	case *tdtg.AuthLoginTokenSuccess:
		return QRLogin{}, nil
	default:
		return QRLogin{}, fmt.Errorf("unexpected login token type: %T", result)
	}
}

// WaitQRLogin polls until the QR code is scanned and login succeeds.
func (c *GotdClient) WaitQRLogin(ctx context.Context, _ QRLogin) error {
	api, err := c.getAPI()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		result, err := api.AuthExportLoginToken(ctx, &tdtg.AuthExportLoginTokenRequest{
			APIID:   c.cfg.AppID,
			APIHash: c.cfg.AppHash,
		})
		if err != nil {
			if tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
				return &PasswordRequiredError{}
			}
			return mapGotdError(err)
		}

		switch r := result.(type) {
		case *tdtg.AuthLoginTokenSuccess:
			_ = r
			return nil
		case *tdtg.AuthLoginTokenMigrateTo:
			return fmt.Errorf("DC migration required (DC %d) during QR login", r.DCID)
		}
	}
}
