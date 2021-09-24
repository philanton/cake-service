package jwt

import (
	"github.com/openware/rango/pkg/auth"
)

const (
    privKeyPath = "privkey.rsa"
    pubKeyPath = "pubkey.rsa"
)

type JWTService struct {
	keys *auth.KeyStore
}

func NewJWTService() (*JWTService, error) {
	keys, err := auth.LoadOrGenerateKeys(privKeyPath, pubKeyPath)
	if err != nil {
		return nil, err
	}

	return &JWTService{keys: keys}, nil
}

func (j *JWTService) GenerateJWT(u User) (string, error) {
	return auth.ForgeToken("empty", u.Email, "empty", 0, j.keys.PrivateKey, nil)
}

func (j *JWTService) ParseJWT(jwt string) (auth.Auth, error) {
	return auth.ParseAndValidate(jwt, j.keys.PublicKey)
}
