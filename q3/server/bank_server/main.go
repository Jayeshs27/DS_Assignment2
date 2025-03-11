package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"log"
	"net"
	// "time"

	// "golang.org/x/crypto/bcrypt"
	// "github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "q3/protofiles"
	common "q3/common"
)

const (
	bankServerAddr = "localhost:45331"
)

var (
	bSLogger *common.Logger
)

// JWT Secret Key
// var jwtKey = []byte("bank_server_key") 

// User struct
type Customer struct {
	CustomerName   string `json:"customer_name"`
	AccNo   string `json:"acc_no"`
	CurrBalance       int64 `json:"curr_balance"`
}

// PaymentServer struct
type BankServer struct {
	pb.UnimplementedBankServiceServer
	Customers map[string]Customer
}

func loadUsers(filename string) map[string]Customer {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Bank Server:Failed to read user file: %v", err)
	}
	var customers []Customer
	if err := json.Unmarshal(data, &customers); err != nil {
		log.Fatalf("Bank Server:Failed to parse user data: %v", err)
	}
	customerMap := make(map[string]Customer)
	for _, customer := range customers {
		customerMap[customer.AccNo] = customer
	}
	return customerMap
}

var (
	bankServer *BankServer
)

func (s *BankServer) CheckBalance(ctx context.Context, req *pb.CheckBalanceRequest) (*pb.CheckBalanceResponse, error) {
	accNo := req.AccNo
	customer, exists := bankServer.Customers[accNo]
	if !exists {
		return &pb.CheckBalanceResponse{CurrBalance: 0}, common.ErrInvalidAccountNo
	}
	return &pb.CheckBalanceResponse{CurrBalance: customer.CurrBalance}, common.ErrSuccess
}

func main() {
	customers := loadUsers("sample_data/bank_customers.json")

	cert, err := tls.LoadX509KeyPair("certs/bank_server.crt", "certs/bank_server.key")
	if err != nil {
		log.Fatalf("Failed to load server certificates: %v", err)
	}

	caCert, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})
	
	// serverOpts := []grpc.ServerOption{grpc.Creds(creds)}
	// serverOpts = append(serverOpts, grpc.UnaryInterceptor(logggingInterceptor))
	// server := grpc.NewServer(serverOpts...)
	bSLogger = common.NewLogger("logs/bank_server")
	defer bSLogger.Close()
	
	server := grpc.NewServer(
		grpc.Creds(creds),
		// grpc.ChainUnaryInterceptor(loggingInterceptor, authInterceptor),
	)
	
	pb.RegisterBankServiceServer(server, &BankServer{Customers: customers})

	listener, err := net.Listen("tcp", bankServerAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	fmt.Printf("Bank Server running on addr: %s...\n",bankServerAddr)
	server.Serve(listener)
}
