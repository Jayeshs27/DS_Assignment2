package main

import (
	"context"
	"fmt"
	// "time"

	// "golang.org/x/crypto/bcrypt"
	// "github.com/golang-jwt/jwt/v5"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials"
	pb "q3/protofiles"
	common "q3/common"
)


func (s *BankServer) CheckBalance(ctx context.Context, req *pb.CheckBalanceRequest) (*pb.CheckBalanceResponse, error) {
	accNo := req.AccNo
	customer, exists := bankServer.Customers[accNo]
	fmt.Printf("check balance request received for acc_no:%s\n", accNo)
	if !exists {
		return &pb.CheckBalanceResponse{CurrBalance: 0}, common.ErrInvalidAccountNo
	}
	return &pb.CheckBalanceResponse{CurrBalance: customer.CurrBalance}, common.ErrSuccess
}

func (s *BankServer) DebitBalance(ctx context.Context, req *pb.DebitRequest) (*pb.DebitResponse, error) {
	accNo := req.AccNo
	customer, exists := bankServer.Customers[accNo]
	fmt.Printf("debit balance request received for acc_no:%s\n", accNo)
	if !exists {
		return &pb.DebitResponse{}, common.ErrInvalidAccountNo
	}
	if customer.CurrBalance < req.Amount{
		return &pb.DebitResponse{}, common.ErrInsufficientBalance
	}
	bankServer.Customers[accNo].SubtractAmount(req.Amount)
	return &pb.DebitResponse{}, common.ErrSuccess
}

func (s *BankServer) CreditBalance(ctx context.Context, req *pb.CreditRequest) (*pb.CreditResponse, error) {
	accNo := req.AccNo
	customer, exists := bankServer.Customers[accNo]
	fmt.Printf("credit balance request received for acc_no:%s\n", accNo)
	if !exists {
		return &pb.CreditResponse{}, common.ErrInvalidAccountNo
	}
	if customer.CurrBalance < req.Amount{
		return &pb.CreditResponse{}, common.ErrInsufficientBalance
	}
	bankServer.Customers[accNo].AddAmount(req.Amount)
	return &pb.CreditResponse{}, common.ErrSuccess
}
