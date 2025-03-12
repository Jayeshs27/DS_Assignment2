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
	"strconv"
	// "time"

	// "golang.org/x/crypto/bcrypt"
	// "github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "q3/protofiles"
	common "q3/common"
)

const (
	paymentGatewayAddr = "localhost:45301"
)

var (
	bSLogger *common.Logger
	bankServer *BankServer
	credsAsClient credentials.TransportCredentials
	credsAsServer credentials.TransportCredentials
)

func getAvaliablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0") 
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// JWT Secret Key
// var jwtKey = []byte("bank_server_key") 

// User struct
type Customer struct {
	CustomerName   string `json:"customer_name"`
	AccNo   string `json:"acc_no"`
	CurrBalance       float32 `json:"curr_balance"`
}

func (c *Customer) SubtractAmount(amount float32){
	c.CurrBalance -= amount
}

func (c *Customer) AddAmount(amount float32){
	c.CurrBalance += amount
}
// PaymentServer struct
type BankServer struct {
	pb.UnimplementedBankServiceServer
	Customers map[string]*Customer
	bankName string
	bankServerAddr string
}

func NewBankServer(bankName string)(*BankServer, error){
	customers := loadUsers("sample_data/bank_customers.json")
	port, err := getAvaliablePort()
	if err != common.ErrSuccess{
		return &BankServer{}, err
	}
	return &BankServer{
		Customers: customers,
		bankName: bankName,
		bankServerAddr: fmt.Sprintf("localhost:%d",port),
	}, common.ErrSuccess
}

func loadUsers(filename string) map[string]*Customer {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Bank Server:Failed to read user file: %v", err)
	}
	var customers []*Customer
	if err := json.Unmarshal(data, &customers); err != nil {
		log.Fatalf("Bank Server:Failed to parse user data: %v", err)
	}
	customerMap := make(map[string]*Customer)
	for _, customer := range customers {
		customerMap[customer.AccNo] = customer
	}
	return customerMap
}

func SendRegisterRequest()(error){
	// Connect to server
	conn, err := grpc.NewClient(paymentGatewayAddr, 
								grpc.WithTransportCredentials(credsAsClient),
							  	)
	if err != common.ErrSuccess{
		return err
	}
	client := pb.NewPaymentServiceClient(conn)
	req := &pb.RegisterRequest{BankName: bankServer.bankName, 
							   BankServerAddr: bankServer.bankServerAddr}
	_, err = client.BankServerDiscovery(context.Background(), req)
	if err != common.ErrSuccess {
		return err
	}
	defer conn.Close()

	return common.ErrSuccess
}

func main() {
	args := os.Args[1:]

	if len(args) < 1{
		log.Fatalf("missing command line argument")
	}
	bankId, err1 := strconv.Atoi(args[0])
	if err1 != nil {
		log.Fatalf("Invalid command line arguments")
	}
	bankName := fmt.Sprintf("bank%d",bankId)

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

	credsAsServer = credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})

	credsAsClient = credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		ServerName: "localhost",
	})
	
	bSLogger = common.NewLogger("logs/bank_server")
	defer bSLogger.Close()
	
	server := grpc.NewServer(
		grpc.Creds(credsAsServer),
		grpc.UnaryInterceptor(bankLoggingInterceptor),
	)
	
	bankServer, err = NewBankServer(bankName)
	if err != common.ErrSuccess {
		log.Fatalf("Failed to create bank server: %v", err)
	}

	pb.RegisterBankServiceServer(server, bankServer)

	err = SendRegisterRequest()
	if err != common.ErrSuccess {
		log.Fatalf("Failed to register with payment gateway: %v", err)
	}

	listener, err := net.Listen("tcp", bankServer.bankServerAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	fmt.Printf("%s Server running on addr: %s...\n", bankServer.bankName, bankServer.bankServerAddr)
	server.Serve(listener)
}
