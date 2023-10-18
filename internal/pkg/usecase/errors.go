package usecase

import (
	"errors"
)

var (
	ErrPublicKeyNotFound = errors.New("public key is not found")

	ErrTokenNotFound = errors.New("token not found")
)
