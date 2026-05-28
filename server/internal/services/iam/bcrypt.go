package iam

import (
	"github.com/leo/iop/server/internal/shared/errors"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword returns bcrypt hash of plaintext.
func HashPassword(plain string) (string, error) {
	if len(plain) < 10 {
		return "", errors.New(errors.KindParam, "iam.password.too_short", "密码至少 10 位")
	}
	if !hasLetterAndDigit(plain) {
		return "", errors.New(errors.KindParam, "iam.password.weak", "密码需包含字母和数字")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword returns nil if plain matches hash.
func CheckPassword(plain, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return errors.New(errors.KindAuth, "iam.invalid_password", "用户名或密码错误")
	}
	return nil
}

func hasLetterAndDigit(s string) bool {
	hasL, hasD := false, false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			hasL = true
		case r >= '0' && r <= '9':
			hasD = true
		}
	}
	return hasL && hasD
}
