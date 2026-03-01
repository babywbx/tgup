package app

import (
	"context"

	"github.com/babywbx/tgup/internal/xerrors"
)

// Login executes login flow (implemented in later milestones).
func Login(ctx context.Context) error {
	_ = ctx
	return xerrors.Wrap(xerrors.CodeAuth, "login flow not implemented yet", nil)
}
