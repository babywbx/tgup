package upload

import (
	"fmt"

	"github.com/babywbx/tgup/internal/tg"
)

// PostcheckResult holds validation results after Telegram upload.
type PostcheckResult struct {
	OK     bool
	Issues []string
}

// PostcheckMessages validates the Telegram response against expectations.
func PostcheckMessages(sent tg.SendResult, expectedCount int) PostcheckResult {
	var issues []string

	actualCount := len(sent.Messages)
	if actualCount == 0 && len(sent.MessageIDs) > 0 {
		actualCount = len(sent.MessageIDs)
	}
	if actualCount != expectedCount {
		issues = append(issues, fmt.Sprintf(
			"message count mismatch: expected %d, got %d",
			expectedCount, actualCount,
		))
	}

	if len(sent.Messages) == 0 {
		return PostcheckResult{OK: len(issues) == 0, Issues: issues}
	}

	for _, msg := range sent.Messages {
		if msg.MediaKind == "video" {
			if msg.Duration <= 0 {
				issues = append(issues, fmt.Sprintf("msg %d: video duration=%v", msg.ID, msg.Duration))
			}
			if msg.Width <= 1 || msg.Height <= 1 {
				issues = append(issues, fmt.Sprintf("msg %d: video dimensions=%dx%d", msg.ID, msg.Width, msg.Height))
			}
			if !msg.SupportsStreaming {
				issues = append(issues, fmt.Sprintf("msg %d: supports_streaming=false", msg.ID))
			}
		}
	}

	return PostcheckResult{
		OK:     len(issues) == 0,
		Issues: issues,
	}
}

// SuccessRate returns sent/total ratio in range [0, 1].
func (s Summary) SuccessRate() float64 {
	if s.Total <= 0 {
		return 0
	}
	return float64(s.Sent) / float64(s.Total)
}
