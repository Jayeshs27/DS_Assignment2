package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	// "time"
	// "errors"
	// "github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/metadata"
	// "golang.org/x/crypto/bcrypt"
	pb "q3/protofiles"
	common "q3/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"github.com/google/uuid"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
}

type RequestType int

var (
	balanceEnquiry RequestType = 1
	makePayment RequestType = 2
	exit        RequestType = 3
)

var (
	clientLogger *common.Logger
	paymentGatewayAddr = "localhost:45301"
)

func sendGetBalanceRequest(ctx context.Context, client pb.PaymentServiceClient, authToken string)(float64, error){
	payResp, err := client.GetBalance(ctx, &pb.GetBalanceRequest{Token: authToken})
	if common.IsEqual(err, common.ErrSuccess) {
		return float64(payResp.Amount), common.ErrSuccess
	}
	return 0, err
}

func sendPaymentRequest(ctx context.Context, client pb.PaymentServiceClient, amount float32, recpAccNo string, recpBankName string, authToken string) (error){
	if amount < 0 {
		return common.ErrInvalidAmount
	}
	transID := uuid.New().String()
	req := &pb.PaymentRequest{Token: authToken, RecpBankName:recpBankName, RecpAccNo: recpAccNo, Amount: amount, TransID: transID}
	_, err := client.MakePayment(ctx, req)
	return err
}

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
	conn, err := grpc.NewClient(paymentGatewayAddr, 
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
		var shouldExit bool = false
		fmt.Printf("Enter Request Type:")
		fmt.Scanln(&reqType)
		if(reqType > 3 || reqType < 1){
			log.Println("Invalid reqType")
		}
		switch; RequestType(reqType) {
			case balanceEnquiry:
				currBalance, err := sendGetBalanceRequest(ctx, client, authResp.Token)
				if !common.IsEqual(err, common.ErrSuccess) {
					log.Printf("Request Failed: %v\n", err)
				} else {
					log.Printf("Current Balance is %f\n", currBalance)
				}
				
			case makePayment:   // to do - check timeout for makepayment
				var recpAccNo, recpBankName string
				var amount float32
				fmt.Print("Enter recipitent Bank Name:")
				fmt.Scanln(&recpBankName)
				fmt.Print("Enter recipitent Acc. No.:")
				fmt.Scanln(&recpAccNo)
				fmt.Print("Enter Amount:")
				fmt.Scanln(&amount)
				err = sendPaymentRequest(ctx, client, amount, recpAccNo, recpBankName, authResp.Token)
				if !common.IsEqual(err, common.ErrSuccess) {
					log.Printf("Payment failed: %v\n", err)
				} else{
					log.Println("Payment Status: Success")
				}
			case exit:
				shouldExit = true
		}
		if shouldExit {
			break
		}
	}
}
