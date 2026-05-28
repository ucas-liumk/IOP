package iam

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/leo/iop/server/internal/shared/errors"
)

// TokenSigner abstracts JWT sign/verify. v1 implementation is HS256.
type TokenSigner interface {
	Sign(c Claims) (string, error)
	Verify(token string) (*Claims, error)
}

type hsSigner struct {
	secret []byte
}

func NewHS256Signer(secret string) TokenSigner {
	return &hsSigner{secret: []byte(secret)}
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

func (s *hsSigner) Sign(c Claims) (string, error) {
	hdr := jwtHeader{Alg: "HS256", Typ: "JWT"}
	hb, _ := json.Marshal(hdr)
	cb, _ := json.Marshal(c)
	h := base64URL(hb) + "." + base64URL(cb)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(h))
	sig := base64URL(mac.Sum(nil))
	return h + "." + sig, nil
}

func (s *hsSigner) Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New(errors.KindAuth, "iam.invalid_token", "token 格式错误")
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := base64URL(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, errors.New(errors.KindAuth, "iam.invalid_signature", "token 签名错误")
	}
	cb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Wrap(errors.KindAuth, "iam.invalid_token", "token payload 无法解码", err)
	}
	var c Claims
	if err := json.Unmarshal(cb, &c); err != nil {
		return nil, errors.Wrap(errors.KindAuth, "iam.invalid_token", "token payload 无效", err)
	}
	if c.ExpiresAt > 0 && time.Now().Unix() > c.ExpiresAt {
		return nil, errors.New(errors.KindAuth, "iam.token_expired", "token 已过期")
	}
	return &c, nil
}

func base64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
