package common

import (
	"errors"
	"google.golang.org/grpc/status"
)


var (
	ErrSuccess error = nil
	ErrUnauthorized = errors.New("error: unauthorized")
	ErrInvalidAccountNo = errors.New("error: invalid account number")
	ErrInvalidBankId = errors.New("error: invalid bank id")
	ErrInvalidCredentials = errors.New("error: invalid credentials")
	ErrInvalidToken = errors.New("error: invalid token")
	ErrInsufficientBalance = errors.New("error: insufficient balance to complete transaction")
	ErrBankServerAlreadyExist = errors.New("error: bank server already exists")
	ErrInvalidUserName = errors.New("error: invalid user name")
	ErrTransactionInProgress = errors.New("error: transaction in progress")
	ErrInvalidAmount = errors.New("error: entered invalid amount")
	ErrBankServerBusy = errors.New("error: server taking too long to response")
	ErrTimeOut = errors.New("error: request timeout")
)

func IsEqual(err error, targetErr error) bool {
	if err == targetErr {   // to handle the case with targetErr = nil (ErrSuccess)
		return true
	}
	if targetErr == nil {  // if targetErr == nil, err != nil
		return false
	}
    if errors.Is(err, targetErr) {
        return true
    }
    s, ok := status.FromError(err)
    return ok && (s.Message() == targetErr.Error())   
}