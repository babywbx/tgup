package tg

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	tdtg "github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// IsSessionAuthorized checks whether the session file already holds
// an authenticated session, without running a full login flow.
func IsSessionAuthorized(ctx context.Context, cfg GotdConfig) (bool, error) {
	client := newLoginClient(cfg, nil)

	authorized := false
	if err := client.Run(ctx, func(runCtx context.Context) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return err
		}
		authorized = status.Authorized
		return nil
	}); err != nil {
		return false, mapGotdError(err)
	}
	return authorized, nil
}

// LoginWithCode authenticates using gotd's official phone+code auth flow.
// Auth runs inside telegram.Client.Run so DC migration and session
// persistence are handled correctly by gotd.
func LoginWithCode(ctx context.Context, cfg GotdConfig, opts CodeLoginOptions) error {
	phone := strings.TrimSpace(opts.Phone)
	if phone == "" {
		return fmt.Errorf("phone number is required")
	}
	if opts.Code == nil {
		return fmt.Errorf("code callback is required")
	}

	client := newLoginClient(cfg, nil)

	flow := auth.NewFlow(codeFlowAuth{
		phone:    phone,
		code:     opts.Code,
		password: opts.Password,
	}, auth.SendCodeOptions{})

	return client.Run(ctx, func(runCtx context.Context) error {
		if err := flow.Run(runCtx, client.Auth()); err != nil {
			return mapGotdError(err)
		}
		return nil
	})
}

// LoginWithQR authenticates using gotd's official QR login helper.
// Auth runs inside telegram.Client.Run so DC migration and session
// persistence are handled correctly by gotd.
func LoginWithQR(ctx context.Context, cfg GotdConfig, opts QRLoginOptions) error {
	if opts.Show == nil {
		return fmt.Errorf("QR show callback is required")
	}

	dispatcher := tdtg.NewUpdateDispatcher()
	loggedIn := qrlogin.OnLoginToken(dispatcher)

	// QR login needs UpdateHandler to receive UpdateLoginToken events.
	client := newLoginClient(cfg, dispatcher)

	return client.Run(ctx, func(runCtx context.Context) error {
		_, err := client.QR().Auth(runCtx, loggedIn, func(ctx context.Context, token qrlogin.Token) error {
			return opts.Show(ctx, token.URL())
		})
		if err == nil {
			return nil
		}
		// QR auth returns a raw tgerr for 2FA, not auth.ErrPasswordAuthNeeded.
		if !tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
			return mapGotdError(err)
		}
		if opts.Password == nil {
			return fmt.Errorf("2FA password required but no password callback provided")
		}
		password, pwErr := opts.Password(runCtx)
		if pwErr != nil {
			return pwErr
		}
		if _, err := client.Auth().Password(runCtx, password); err != nil {
			return mapGotdError(err)
		}
		return nil
	})
}

// newLoginClient creates a telegram.Client configured for a login session.
// If updateHandler is non-nil it is installed (needed for QR login events).
func newLoginClient(cfg GotdConfig, updateHandler telegram.UpdateHandler) *telegram.Client {
	opts := telegram.Options{
		SessionStorage: &FileSessionStore{Path: cfg.SessionPath},
		Device:         gotdDeviceConfig(cfg),
	}
	if updateHandler != nil {
		opts.UpdateHandler = updateHandler
	} else {
		opts.NoUpdates = true
	}
	return telegram.NewClient(cfg.AppID, cfg.AppHash, opts)
}

// codeFlowAuth implements auth.UserAuthenticator using callbacks.
type codeFlowAuth struct {
	phone    string
	code     CodePrompt
	password PasswordPrompt
}

func (a codeFlowAuth) Phone(context.Context) (string, error) { return a.phone, nil }

func (a codeFlowAuth) Code(ctx context.Context, sentCode *tdtg.AuthSentCode) (string, error) {
	return a.code(ctx, sentCode)
}

func (a codeFlowAuth) Password(ctx context.Context) (string, error) {
	if a.password == nil {
		return "", auth.ErrPasswordNotProvided
	}
	return a.password(ctx)
}

func (a codeFlowAuth) AcceptTermsOfService(context.Context, tdtg.HelpTermsOfService) error {
	return fmt.Errorf("sign-up required: this phone number is not registered")
}

func (a codeFlowAuth) SignUp(context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign-up required: this phone number is not registered")
}
