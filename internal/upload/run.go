package upload

import "context"

// Input is a bootstrap upload run input placeholder.
type Input struct {
	Total int
}

// Run is a placeholder upload orchestration entry.
func Run(ctx context.Context, in Input) (Summary, error) {
	_ = ctx
	return Summary{Total: in.Total}, nil
}
