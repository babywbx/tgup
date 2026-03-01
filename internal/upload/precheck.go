package upload

import "fmt"

// ValidateDuplicatePolicy validates duplicate policy values.
func ValidateDuplicatePolicy(policy string) error {
	switch DuplicatePolicy(policy) {
	case DuplicateSkip, DuplicateAsk, DuplicateUpload:
		return nil
	default:
		return fmt.Errorf("invalid duplicate policy: %s", policy)
	}
}
