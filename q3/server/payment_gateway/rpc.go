package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials"
	common "q3/common"
	pb "q3/protofiles"
)

type RequestType int 
var (
	creditRequest RequestType = 0
	debitRequest RequestType = 1
)
var (
	timeoutInterval = 5
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

func (s *PaymentServer) checkUserValidity(reqToken string)(User, error){
	token, err := jwt.Parse(reqToken, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		panic(common.ErrInvalidToken)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		panic(common.ErrInvalidToken)
	}
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

func (s *PaymentServer) checkAndUpdateTranscation(txId string)(bool, error){
	s.TransListmutex.Lock()
	defer s.TransListmutex.Unlock()
	status, exists := s.UserTransactions[txId] 
	if exists {
		if status == common.ErrTransactionInProgress || status  == common.ErrSuccess {
			return false, status
		}
	}
	s.UserTransactions[txId] = common.ErrTransactionInProgress
	return true, status
}

func (s *PaymentServer) UpdateTransaction(txId string, status error){
	s.TransListmutex.Lock()
	s.UserTransactions[txId] = status
	s.TransListmutex.Unlock()
}

// Process payment
func (s *PaymentServer) MakePayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	user, err := pgServer.checkUserValidity(req.Token)
	if !common.IsEqual(err, common.ErrSuccess) {
		return nil, err
	}
	_, err = pgServer.checkBankValidity(user.BankName)
	if !common.IsEqual(err, common.ErrSuccess) {
		return nil, err
	}
	_, err = pgServer.checkBankValidity(req.RecpBankName)
	if !common.IsEqual(err, common.ErrSuccess) {
		return nil, err
	}
	ok, status := pgServer.checkAndUpdateTranscation(req.TransID)
	if !ok {
		return nil, status
	}
	err = s.sendPrepare(req, user)
	pgServer.UpdateTransaction(req.TransID, err)
	if !common.IsEqual(err, common.ErrSuccess) {
		return nil, err
	}
	err = s.sendCommit(req, user)
	pgServer.UpdateTransaction(req.TransID, err)
	if !common.IsEqual(err, common.ErrSuccess) {
		return nil, err
	}
	return nil, common.ErrSuccess
}

func (s *PaymentServer) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error){
	log.Printf("Sending check balance request...\n")
	user, err := pgServer.checkUserValidity(req.Token)
	if !common.IsEqual(err, common.ErrSuccess) {
		return nil, err
	}
	bankAddr, err := pgServer.checkBankValidity(user.BankName)
	if !common.IsEqual(err, common.ErrSuccess) {
		return nil, err
	}
	currBalance, err := SendCheckBalanceRequest(bankAddr, user.AccountNo)
	if !common.IsEqual(err, common.ErrSuccess){
		return nil, err
	}
	return &pb.GetBalanceResponse{Amount: currBalance}, common.ErrSuccess
}

func (s *PaymentServer) BankServerDiscovery(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error){
	bankName, bankAddr := req.BankName, req.BankServerAddr
	s.bankListmutex.Lock()
	defer s.bankListmutex.Unlock()
	_, exists := s.BankServers[bankName]
	if exists {
		return nil, common.ErrBankServerAlreadyExist
	}
	s.BankServers[bankName] = bankAddr
	log.Printf("Bank server-%s, Addr-%s, registered Sucessfully!\n", bankName, bankAddr)
	return &pb.RegisterResponse{}, common.ErrSuccess
}

func (s *PaymentServer) sendPrepareRequest(reqType RequestType, bankAddr string, accNo string, amount float32, txID string)(error){
	conn, err := grpc.NewClient(bankAddr, 
		grpc.WithTransportCredentials(credsForBankServer),
	)
	if !common.IsEqual(err, common.ErrSuccess){
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutInterval) * time.Second)
	defer cancel() 

	client := pb.NewBankServiceClient(conn)
	_, err = client.PrepareTransaction(ctx, &pb.PrepareRequest{ReqType: int32(reqType), 
														AccNo: accNo, Amount: amount, TransID: txID})
	if !common.IsEqual(err, common.ErrSuccess) {
		if common.IsEqual(ctx.Err(), context.DeadlineExceeded){
			log.Println("Timeout: Bank", bankAddr, "did not respond in time")
			return common.ErrTimeOut
		}
		return err
	}
	return common.ErrSuccess
}

func (s *PaymentServer) sendCommitRequest(reqType RequestType, bankAddr string, accNo string, amount float32, txID string)(error){
	conn, err := grpc.NewClient(bankAddr, 
								grpc.WithTransportCredentials(credsForBankServer),
							  	)
	if !common.IsEqual(err, common.ErrSuccess){
		return err
	}
	defer conn.Close()

	client := pb.NewBankServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutInterval) * time.Second)
	defer cancel() 

	_, err = client.CommitTransaction(context.Background(), &pb.CommitRequest{ReqType: int32(reqType), 
														AccNo: accNo, Amount: amount, TransID: txID})
	if !common.IsEqual(err, common.ErrSuccess) {
		if common.IsEqual(ctx.Err(), context.DeadlineExceeded){
			log.Println("Timeout: Bank", bankAddr, "did not respond in time")
			return common.ErrTimeOut
		}
		return err
	}
	return common.ErrSuccess
}

func (s *PaymentServer) sendReleaseResourceRequest(bankAddr string, accNo string)(error){
	conn, err := grpc.NewClient(bankAddr, 
								grpc.WithTransportCredentials(credsForBankServer),
							  	)
	if !common.IsEqual(err, common.ErrSuccess){
		return err
	}
	defer conn.Close()

	client := pb.NewBankServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutInterval) * time.Second)
	defer cancel() 

	_, err = client.ReleaseResource(context.Background(), &pb.ReleaseRequest{AccNo: accNo})
	if !common.IsEqual(err, common.ErrSuccess) {
		if common.IsEqual(ctx.Err(), context.DeadlineExceeded){
			log.Println("Timeout: Bank", bankAddr, "did not respond in time")
			return common.ErrTimeOut
		}
		return err
	}
	return common.ErrSuccess
}

func (s *PaymentServer) sendPrepare(req *pb.PaymentRequest, user User)(error){
	log.Printf("Sending prepare transaction request...\n")
	senderBankAddr := s.BankServers[user.BankName]
	senderAccNo := user.AccountNo
	recpAccNo, amount, txId:= req.RecpAccNo, req.Amount, req.TransID
	recpBankAddr := s.BankServers[req.RecpBankName]
	err := s.sendPrepareRequest(debitRequest, senderBankAddr, senderAccNo, amount, txId)
	if !common.IsEqual(err, common.ErrSuccess) {
		return err
	}
	err = s.sendPrepareRequest(creditRequest, recpBankAddr, recpAccNo, amount, txId)
	if !common.IsEqual(err, common.ErrSuccess) {
		s.sendReleaseResourceRequest(senderBankAddr, senderAccNo)
		return err
	}
	return common.ErrSuccess
}

func (s *PaymentServer) sendCommit(req *pb.PaymentRequest, user User)(error){
	log.Printf("Sending commit transaction request...\n")
	senderBankAddr := s.BankServers[user.BankName]
	senderAccNo := user.AccountNo
	recpAccNo, amount, txId:= req.RecpAccNo, req.Amount, req.TransID
	recpBankAddr := s.BankServers[req.RecpBankName]
	err := s.sendCommitRequest(debitRequest, senderBankAddr, senderAccNo, amount, txId)
	if !common.IsEqual(err, common.ErrSuccess) {
		return err
	}
	err = s.sendCommitRequest(creditRequest, recpBankAddr, recpAccNo, amount, txId)
	if !common.IsEqual(err, common.ErrSuccess) {
		return err
	}
	return common.ErrSuccess
}