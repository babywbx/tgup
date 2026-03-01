package app

import (
	"context"

	"github.com/wbx/tgup/internal/xerrors"
)

// Run executes upload flow (implemented in later milestones).
func Run(ctx context.Context) error {
	_ = ctx
	return xerrors.Wrap(xerrors.CodeUpload, "run flow not implemented yet", nil)
}
