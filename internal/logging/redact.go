package logging

import "strings"

// Redactor masks sensitive tokens in log strings.
type Redactor struct {
	secrets []string
	mask    string
}

// NewRedactor creates a value redactor.
func NewRedactor(secrets []string) Redactor {
	filtered := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		filtered = append(filtered, secret)
	}
	return Redactor{
		secrets: filtered,
		mask:    "***",
	}
}

// Redact replaces registered secrets with mask.
func (r Redactor) Redact(input string) string {
	out := input
	for _, secret := range r.secrets {
		out = strings.ReplaceAll(out, secret, r.mask)
	}
	return out
}
