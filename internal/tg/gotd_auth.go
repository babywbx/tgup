package tg

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
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

	c.phoneCodeHash = "" // clear any stale state

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

	result, err := api.AuthSignIn(ctx, &tdtg.AuthSignInRequest{
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
	c.phoneCodeHash = ""

	switch result.(type) {
	case *tdtg.AuthAuthorization:
		return nil
	case *tdtg.AuthAuthorizationSignUpRequired:
		return fmt.Errorf("sign-up required: this phone number is not registered")
	default:
		return fmt.Errorf("unexpected sign-in result: %T", result)
	}
}

// PasswordRequiredError signals that 2FA password is needed.
type PasswordRequiredError struct{}

func (e *PasswordRequiredError) Error() string {
	return "2FA password required"
}

// SignInWithPassword completes sign-in with 2FA password.
// Uses the DC-migrated connection if QR login triggered DC migration.
func (c *GotdClient) SignInWithPassword(ctx context.Context, password string) error {
	api, err := c.getAuthAPI()
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
// Handles AUTH_TOKEN_EXPIRED by retrying with a fresh token.
func (c *GotdClient) StartQRLogin(ctx context.Context) (QRLogin, error) {
	api, err := c.getAPI()
	if err != nil {
		return QRLogin{}, err
	}

	const maxRetries = 3
	for attempt := range maxRetries {
		result, err := api.AuthExportLoginToken(ctx, &tdtg.AuthExportLoginTokenRequest{
			APIID:   c.cfg.AppID,
			APIHash: c.cfg.AppHash,
		})
		if err != nil {
			// Stale token from previous attempt — retry to get a fresh one.
			if tgerr.Is(err, "AUTH_TOKEN_EXPIRED") || tgerr.Is(err, "AUTH_TOKEN_INVALID") {
				if attempt < maxRetries-1 {
					continue
				}
			}
			return QRLogin{}, mapGotdError(err)
		}

		switch r := result.(type) {
		case *tdtg.AuthLoginToken:
			url := "tg://login?token=" + base64.URLEncoding.EncodeToString(r.Token)
			return QRLogin{URL: url, token: r.Token}, nil
		case *tdtg.AuthLoginTokenSuccess:
			return QRLogin{}, nil
		case *tdtg.AuthLoginTokenMigrateTo:
			// Previous QR scan triggered DC migration; complete it now.
			if err := c.importLoginToken(ctx, r.DCID, r.Token); err != nil {
				if errors.Is(err, ErrLoginPending) {
					// Token imported but login not complete — retry.
					continue
				}
				return QRLogin{}, err
			}
			return QRLogin{}, nil
		default:
			return QRLogin{}, fmt.Errorf("unexpected login token type: %T", result)
		}
	}

	return QRLogin{}, fmt.Errorf("failed to export login token after %d attempts", maxRetries)
}

// WaitQRLogin polls until the QR code is scanned and login succeeds.
// When the server returns a fresh token (old one expired), onRefresh is
// called so the caller can re-render the QR code.
func (c *GotdClient) WaitQRLogin(ctx context.Context, initial QRLogin, onRefresh func(QRLogin)) error {
	api, err := c.getAPI()
	if err != nil {
		return err
	}

	const (
		pollInterval = 2 * time.Second
		maxTransient = 5
	)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	lastToken := initial.token
	transientCount := 0

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
			mapped := mapGotdError(err)
			if errors.Is(mapped, ErrRetryable) {
				transientCount++
				if transientCount >= maxTransient {
					return fmt.Errorf("too many transient errors: %w", mapped)
				}
				continue
			}
			return mapped
		}
		transientCount = 0

		switch r := result.(type) {
		case *tdtg.AuthLoginTokenSuccess:
			return nil

		case *tdtg.AuthLoginToken:
			// Only refresh QR when token actually changes (old one expired).
			if !bytes.Equal(r.Token, lastToken) {
				lastToken = r.Token
				if onRefresh != nil {
					url := "tg://login?token=" + base64.URLEncoding.EncodeToString(r.Token)
					onRefresh(QRLogin{URL: url})
				}
			}

		case *tdtg.AuthLoginTokenMigrateTo:
			if err := c.importLoginToken(ctx, r.DCID, r.Token); err != nil {
				if errors.Is(err, ErrLoginPending) {
					// Token imported but login not complete — keep polling.
					continue
				}
				return err
			}
			return nil
		}
	}
}

// importLoginToken handles DC migration during QR login by connecting to
// the target DC and importing the token there. The DC connection is kept
// alive so subsequent auth calls (e.g. SignInWithPassword) use it.
func (c *GotdClient) importLoginToken(ctx context.Context, dcID int, token []byte) error {
	// Clean up any previous migrated DC connection.
	if c.authCleanup != nil {
		c.authCleanup()
		c.authAPI = nil
		c.authCleanup = nil
	}

	dcInvoker, err := c.telegram.DC(ctx, dcID, 1)
	if err != nil {
		return fmt.Errorf("connect to DC %d: %w", dcID, err)
	}

	// Store migrated DC connection for subsequent auth calls.
	c.authAPI = tdtg.NewClient(dcInvoker)
	c.authCleanup = func() { dcInvoker.Close() }

	result, err := c.authAPI.AuthImportLoginToken(ctx, token)
	if err != nil {
		if tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
			return &PasswordRequiredError{}
		}
		return mapGotdError(err)
	}

	switch result.(type) {
	case *tdtg.AuthLoginTokenSuccess:
		return nil
	case *tdtg.AuthLoginToken:
		// Token accepted on target DC but login not yet complete.
		// Caller should continue polling rather than treating as success.
		return ErrLoginPending
	default:
		return fmt.Errorf("unexpected import token result: %T", result)
	}
}
