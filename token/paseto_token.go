package token

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/o1egl/paseto"
	"golang.org/x/crypto/chacha20poly1305"
)

type PasetoMaker struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

func NewPasetoMaker(symmetricKey string) (Maker, error) {
	if len(symmetricKey) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: must be %d characters", chacha20poly1305.KeySize)
	}

	maker := &PasetoMaker{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
	}
	return maker, nil
}

func (maker *PasetoMaker) CreateToken(username string, duration time.Duration) (string, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	token := Payload{
		ID:        tokenID,
		Username:  username,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}
	return maker.paseto.Encrypt(maker.symmetricKey, token, nil)
}

func (maker *PasetoMaker) VerifyToken(tokenString string) (*Payload, error) {
	token := &Payload{}

	err := maker.paseto.Decrypt(tokenString, maker.symmetricKey, token, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if time.Now().After(token.ExpiredAt) {
		return nil, ErrExpiredToken
	}
	return token, nil
}
