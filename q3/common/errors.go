package common

import (
	"errors"
)

var (
	ErrSuccess error = nil
	ErrUnauthorized = errors.New("error: unauthorized")
	ErrInvalidAccountNo = errors.New("error: invalid account number")
	ErrInvalidBankId = errors.New("error: invalid bank id")
	ErrInvalidCredentials = errors.New("error: invalid credentials")
	ErrInvalidToken = errors.New("error: invalid token")
	ErrInsufficientBalance = errors.New("error: insufficient balance to complete transaction")
)