package tg

import (
	"github.com/gotd/td/telegram/auth"
	tdtg "github.com/gotd/td/tg"
)

// computeSRPAnswer computes the SRP answer for 2FA password check
// using gotd's built-in SRP implementation.
func computeSRPAnswer(password string, pwd *tdtg.AccountPassword) (tdtg.InputCheckPasswordSRPClass, error) {
	if !pwd.HasPassword {
		return &tdtg.InputCheckPasswordEmpty{}, nil
	}
	return auth.PasswordHash([]byte(password), pwd.SRPID, pwd.SRPB, pwd.SecureRandom, pwd.CurrentAlgo)
}
