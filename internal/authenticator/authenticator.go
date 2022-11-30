package authenticator

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func NewAuthenticator(storage storage) *authenticator {
	return &authenticator{storage: storage}
}

func (auth *authenticator) Authenticate(ctx context.Context, username, password string) (string, error) {
	if err := auth.storage.IsUserExisting(ctx, username, password); err != nil {
		return "", err
	}

	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenDuration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (auth *authenticator) VerifyToken(token string) error {
	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		return err
	}

	if !tkn.Valid {
		return errors.New("invalid token")
	}

	return nil
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var jwtKey = []byte("lime_secret_key")

const tokenDuration = 666 * time.Minute

type storage interface {
	IsUserExisting(ctx context.Context, username, password string) error
}

type authenticator struct {
	storage storage
}
