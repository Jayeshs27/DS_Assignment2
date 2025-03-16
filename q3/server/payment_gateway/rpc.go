package main

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials"
	pb "q3/protofiles"
	common "q3/common"
)


func (s *PaymentServer) Authenticate(ctx context.Context, req *pb.UserCredentials) (*pb.AuthResponse, error) {
	user, exists := s.Users[req.Username]
	if !exists || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		return nil, common.ErrInvalidCredentials
	}

	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &jwt.MapClaims{
		"username": req.Username,
		"role":     user.Role,
		"exp":      expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)

	if err != nil {
		return nil, fmt.Errorf("could not generate token")
	}

	return &pb.AuthResponse{Token: tokenString, Role: user.Role}, common.ErrSuccess
}

func checkTokenValidity(reqToken string) (jwt.MapClaims, error){
	token, err := jwt.Parse(reqToken, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return  nil, common.ErrInvalidToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, common.ErrInvalidToken
	} else if claims["role"] != "customer" {
		return nil, common.ErrUnauthorized
	}
	return claims, common.ErrSuccess	
}

func (s *PaymentServer) checkUserValidity(claims jwt.MapClaims)(User, error){
	userName := claims["username"].(string)
	user, exists := s.Users[userName] 
	if !exists {
		return User{}, common.ErrInvalidUserName
	}
	return user, common.ErrSuccess
}

func (s *PaymentServer) checkBankValidity(bankName string)(string, error){
	bankAddr, exists := s.BankServers[bankName]
	if !exists {
		return "", common.ErrInvalidBankId
	}
	return bankAddr, common.ErrSuccess
}

// Process payment
func (s *PaymentServer) MakePayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	// Validate JWT token
	claims, err := checkTokenValidity(req.Token)
	if err != common.ErrSuccess {
		return nil, err
	}
	user, err := pgServer.checkUserValidity(claims)
	if err != common.ErrSuccess {
		return nil, err
	}
	bankAddr, err := pgServer.checkBankValidity(user.BankName)
	if err != common.ErrSuccess {
		return nil, err
	}
	recpBankAddr, err := pgServer.checkBankValidity(req.RecpBankName)
	if err != common.ErrSuccess {
		return nil, err
	}
	_, amount, txId := req.RecpAccNo, req.Amount, req.TransID
	err = SendDebitRequest(bankAddr, user.AccountNo, amount, txId)
	if err != common.ErrSuccess{
		return nil, err
	}
	err = SendCreditRequest(recpBankAddr, req.RecpAccNo, amount, txId)
	if err != common.ErrSuccess{
		return nil, err
	}

	// time.Sleep(5 * time.Second)
	return &pb.PaymentResponse{
		Status:  "success",
		Message: "Payment processed successfully",
	}, common.ErrSuccess
}

func (s *PaymentServer) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error){
	claims, err := checkTokenValidity(req.Token)
	if err != common.ErrSuccess {
		return nil, err
	}
	user, err := pgServer.checkUserValidity(claims)
	if err != common.ErrSuccess {
		return nil, err
	}
	bankAddr, err := pgServer.checkBankValidity(user.BankName)
	if err != common.ErrSuccess {
		return nil, err
	}
	currBalance, err := SendCheckBalanceRequest(bankAddr, user.AccountNo)
	if err != common.ErrSuccess{
		return nil, err
	}
	fmt.Printf("Current Balance is %f\n", currBalance)
	return &pb.GetBalanceResponse{Amount: currBalance}, common.ErrSuccess
}

func (s *PaymentServer) BankServerDiscovery(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error){
	bankName, bankAddr := req.BankName, req.BankServerAddr
	_, exists := s.BankServers[bankName]
	if exists {
		return nil, common.ErrBankServerAlreadyExist
	}
	s.BankServers[bankName] = bankAddr
	fmt.Printf("Bank server-%s, Addr-%s, registered Sucessfully!\n", bankName, bankAddr)
	return &pb.RegisterResponse{}, common.ErrSuccess
}