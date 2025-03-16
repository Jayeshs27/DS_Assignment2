package main

import (
	"context"
	"fmt"
	// "time"

	// "time"

	// "golang.org/x/crypto/bcrypt"
	// "github.com/golang-jwt/jwt/v5"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials"
	common "q3/common"
	pb "q3/protofiles"
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

func PerformDebit(req *pb.DebitRequest) (error){
	accNo := req.AccNo
	customer, exists := bankServer.Customers[accNo]
	fmt.Printf("debit balance request received for acc_no:%s\n", accNo)
	if !exists {
		return common.ErrInvalidAccountNo
	}
	if customer.CurrBalance < req.Amount{
		return common.ErrInsufficientBalance
	}
	bankServer.Customers[accNo].SubtractAmount(req.Amount)
	return common.ErrSuccess
}

func PerformCredit(req *pb.CreditRequest) (error){
	accNo := req.AccNo
	_, exists := bankServer.Customers[accNo]
	fmt.Printf("credit balance request received for acc_no:%s\n", accNo)
	if !exists {
		return common.ErrInvalidAccountNo
	}
	bankServer.Customers[accNo].AddAmount(req.Amount)
	return common.ErrSuccess
}

func (s *BankServer) DebitBalance(ctx context.Context, req *pb.DebitRequest) (*pb.DebitResponse, error) {
	txId := req.TransID
	status, exists := bankServer.DebitTransactions[txId]
	if exists {
		return &pb.DebitResponse{}, status
	}
	bankServer.DebitTransactions[txId] = common.ErrTransactionInProgress
	status = PerformDebit(req)
	bankServer.DebitTransactions[txId] = status
	return &pb.DebitResponse{}, status
}

func (s *BankServer) CreditBalance(ctx context.Context, req *pb.CreditRequest) (*pb.CreditResponse, error) {
	txId := req.TransID
	status, exists := bankServer.CreditTransactions[txId]
	if exists { 
		return &pb.CreditResponse{}, status
	}
	bankServer.CreditTransactions[txId] = common.ErrTransactionInProgress
	status = PerformCredit(req)
	bankServer.CreditTransactions[txId] = status
	return &pb.CreditResponse{}, status
}
