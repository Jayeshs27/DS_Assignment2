package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	// "github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/metadata"
	// "golang.org/x/crypto/bcrypt"
	pb "q3/protofiles"
	common "q3/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
}


var (
	clientLogger *common.Logger
)

func main() {
	// Load TLS credentials
	cert, err := tls.LoadX509KeyPair("certs/client.crt", "certs/client.key")
	if err != nil {
		log.Fatalf("Failed to load client certificates: %v", err)
	}
	
	caCert, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		ServerName: "localhost",
	})

	clientLogger = common.NewLogger("logs/client")
	defer clientLogger.Close()
	// Connect to server
	conn, err := grpc.NewClient("localhost:45301", 
								grpc.WithTransportCredentials(creds),
							  	grpc.WithUnaryInterceptor(loggingInterceptor))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewPaymentServiceClient(conn)

	// Authenticate
	var username, password string
	fmt.Print("Enter username: ")
	fmt.Scanln(&username)
	fmt.Print("Enter password: ")
	fmt.Scanln(&password)

	authResp, err := client.Authenticate(context.Background(), &pb.UserCredentials{Username: username, Password: password})
	if err != common.ErrSuccess {
		log.Fatalf("Authentication failed: %v", err)
	}

	fmt.Println("Authenticated! Token:", authResp.Token)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", authResp.Token))

	for {
		var reqType int
		fmt.Scanln(&reqType)
		if(reqType > 2 || reqType < 1){
			log.Println("Invalid reqType")
		}
		switch; reqType {
			case 1:
				payResp, err := client.GetBalance(ctx, &pb.GetBalanceRequest{Token: authResp.Token})
				if err != common.ErrSuccess {
					log.Fatalf("Request Failed: %v", err)
				}
				fmt.Printf("Current Balance is %f\n", payResp.Amount)
				
			case 2:
				var recpAccNo, recpBankName string
				var amount float32
				fmt.Print("Enter recipitent Bank Name:")
				fmt.Scanln(&recpBankName)
				fmt.Print("Enter recipitent Acc. No.:")
				fmt.Scanln(&recpAccNo)
				fmt.Print("Enter Amount:")
				fmt.Scanln(&amount)
				req := &pb.PaymentRequest{Token: authResp.Token, RecpBankName:recpBankName, RecpAccNo: recpAccNo, Amount: amount}
				payResp, err := client.MakePayment(ctx, req)
				if err != common.ErrSuccess {
					log.Fatalf("Payment failed: %v", err)
				}
				fmt.Println("Payment Status:", payResp.Status, "- Message:", payResp.Message)
				
		}
	}
}
