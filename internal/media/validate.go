package media

import "math"

// ViolationReason returns a reason string if video metadata is invalid for
// Telegram upload, or empty string if valid. Matches the constraints:
// duration > 0, width > 1, height > 1.
func ViolationReason(m VideoMetadata) string {
	if math.IsNaN(m.DurationSeconds) || math.IsInf(m.DurationSeconds, 0) {
		return "duration_invalid"
	}
	if m.DurationSeconds <= 0 {
		if m.DurationSeconds == 0 {
			return "duration_missing"
		}
		return "duration_non_positive"
	}
	if m.Width <= 1 {
		if m.Width == 0 {
			return "width_missing"
		}
		return "width_too_small"
	}
	if m.Height <= 1 {
		if m.Height == 0 {
			return "height_missing"
		}
		return "height_too_small"
	}
	return ""
}
