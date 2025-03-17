package main

import (
	"context"
	"time"
	// "fmt"
	"log"
	// "time"

	// "time"

	// "time"

	// "golang.org/x/crypto/bcrypt"
	// "github.com/golang-jwt/jwt/v5"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials"
	common "q3/common"
	pb "q3/protofiles"
)

var (
	maxAcquireAttempts = 3
	AdditionalDelay = 0
)


func (s *BankServer) CheckBalance(ctx context.Context, req *pb.CheckBalanceRequest) (*pb.CheckBalanceResponse, error) {
	accNo := req.AccNo
	customer, exists := bankServer.Customers[accNo]
	log.Printf("check balance request received for account no.:%s\n", accNo)
	if !exists {
		return &pb.CheckBalanceResponse{CurrBalance: 0}, common.ErrInvalidAccountNo
	}
	return &pb.CheckBalanceResponse{CurrBalance: customer.CurrBalance}, common.ErrSuccess
}

func checkPrepareRequestValidity(req *pb.PrepareRequest)(error){
	accNo := req.AccNo
	customer, exists := bankServer.Customers[accNo]
	if !exists {
		return common.ErrInvalidAccountNo
	}
	if RequestType(req.ReqType) == debitRequest && (customer.getBalance() < req.Amount) {
		return common.ErrInsufficientBalance
	}
	return common.ErrSuccess
}

func checkCommitRequestValidity(req *pb.CommitRequest)(error){
	accNo := req.AccNo
	customer, exists := bankServer.Customers[accNo]
	if !exists {
		return common.ErrInvalidAccountNo
	}
	if RequestType(req.ReqType) == debitRequest && (customer.getBalance() < req.Amount) {
		return common.ErrInsufficientBalance
	}
	return common.ErrSuccess
}

func (s *BankServer) acquireAccountLock(accNo string)(bool){
	cust := s.Customers[accNo]
	for i := range maxAcquireAttempts {
		if !cust.checkAndAcquire(){
			log.Printf("account No.%s, lock acquired", accNo)
			return true;
		}
		time.Sleep(time.Duration((i + 1) * 100) * time.Millisecond)
	}
	log.Printf("account No.%s, can't acquire lock", accNo)
	return false
}

func (s *BankServer) PrepareTransaction(ctx context.Context, req *pb.PrepareRequest) (*pb.PrepareResponse, error) {
	log.Printf("prepare transcaction request received for account no.:%s\n", req.AccNo)
	err := checkPrepareRequestValidity(req)
	if !common.IsEqual(err, common.ErrSuccess){
		return nil, err
	}
	// time.Sleep(5 * time.Second)
	if !s.acquireAccountLock(req.AccNo) {   // check and acquires account lock
		return nil, common.ErrBankServerBusy
	}
	// Ignored this for now
	// txId := req.TransID
	// status, exists := bankServer.DebitTransactions[txId]
	// if exists {
	// 	return &pb.PrepareResponse{}, status
	// }
	// bankServer.DebitTransactions[txId] = common.ErrTransactionInProgress
	// bankServer.DebitTransactions[txId] = status

	return &pb.PrepareResponse{}, common.ErrSuccess
}

func (s *BankServer) CommitTransaction(ctx context.Context, req *pb.CommitRequest) (*pb.CommitResponse, error) {
	log.Printf("commit transcaction request received for account no.:%s\n", req.AccNo)
	err := checkCommitRequestValidity(req)
	if !common.IsEqual(err, common.ErrSuccess){
		panic("Commit Request Should be valid")
	}
	if !s.Customers[req.AccNo].isAccountLocked() {   // check and acquires account lock
		panic("Account Should be lock while commit")
	}
	resp, err := s.ProcessTransaction(ctx, req)
	s.Customers[req.AccNo].unLockAccount()
	return resp, err
}

func (s *BankServer) DebitBalance(ctx context.Context, req *pb.CommitRequest) (*pb.CommitResponse, error) {
	// txId := req.TransID
	// status, exists := bankServer.DebitTransactions[txId]
	// if exists {
	// 	return &pb.CommitResponse{}, status
	// }
	// bankServer.DebitTransactions[txId] = common.ErrTransactionInProgress

	// Addition delay to simulate timeouts (to check idopotency feature)
	time.Sleep(time.Duration(AdditionalDelay) * time.Second)
	/////////////////
	s.Customers[req.AccNo].subtractAmount(req.Amount)
	// bankServer.DebitTransactions[txId] = status
	return &pb.CommitResponse{}, common.ErrSuccess
}

func (s *BankServer) CreditBalance(ctx context.Context, req *pb.CommitRequest) (*pb.CommitResponse, error) {
	// txId := req.TransID
	// status, exists := bankServer.CreditTransactions[txId]
	// if exists { 
	// 	return &pb.CommitResponse{}, status
	// }
	// bankServer.CreditTransactions[txId] = common.ErrTransactionInProgress
	s.Customers[req.AccNo].addAmount(req.Amount)
	// bankServer.CreditTransactions[txId] = status
	return &pb.CommitResponse{}, common.ErrSuccess
}

func (s *BankServer) ProcessTransaction(ctx context.Context, req *pb.CommitRequest) (*pb.CommitResponse, error) {
	// txId := req.TransID
	// status, exists := bankServer.CreditTransactions[txId]
	// if exists { 
	// 	return &pb.CommitResponse{}, status
	// }
	// bankServer.CreditTransactions[txId] = common.ErrTransactionInProgress
	// bankServer.CreditTransactions[txId] = status
	if RequestType(req.ReqType) == debitRequest {
		return s.DebitBalance(ctx, req)
	}else if RequestType(req.ReqType) == creditRequest {
		return s.CreditBalance(ctx, req)
	}else{
		panic("reqType should be valid")
	}
}

func (s *BankServer) ReleaseResource(ctx context.Context, req *pb.ReleaseRequest) (*pb.ReleaseResponse, error) {
	log.Printf("release resource request received for account no.:%s\n", req.AccNo)
	s.Customers[req.AccNo].unLockAccount()
	return &pb.ReleaseResponse{}, common.ErrSuccess
}
