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
	// "google.golang.org/grpc/status"
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
	Authetication RequestType = 0
	balanceEnquiry RequestType = 1
	makePayment RequestType = 2
	exit        RequestType = 3
	setOffline  RequestType = 4
)

var (
	maxLoginAttempts = 5
	authToken string
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

func sendPaymentRequest(ctx context.Context, client pb.PaymentServiceClient, amount float32, recpAccNo string, recpBankName string, authToken string) (pb.PaymentResponse, error){
	if amount < 0 {
		return pb.PaymentResponse{}, common.ErrInvalidAmount
	}
	transID := uuid.New().String()
	req := &pb.PaymentRequest{Token: authToken, RecpBankName:recpBankName, RecpAccNo: recpAccNo, Amount: amount, TransID: transID}
	_, err := client.MakePayment(ctx, req)
	return pb.PaymentResponse{}, err
}

func sendAutheticationRequest(ctx context.Context, client pb.PaymentServiceClient, username string, password string)(string, error){
	authResp, err := client.Authenticate(ctx, &pb.UserCredentials{Username: username, Password: password})
	if !common.IsEqual(err, common.ErrSuccess) {
		return "", err
	}
	return authResp.Token, err
}

func main() {
	// Load TLS credentials
	cert, err := tls.LoadX509KeyPair("certs/client.crt", "certs/client.key")
	if !common.IsEqual(err, common.ErrSuccess) {
		log.Fatalf("Failed to load client certificates: %v", err)
	}
	
	caCert, err := os.ReadFile("certs/ca.crt")
	if !common.IsEqual(err, common.ErrSuccess) {
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
							  	grpc.WithUnaryInterceptor(ClientRequestInterceptor))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewPaymentServiceClient(conn)
	offlineClientHandler = NewOfflineHandler()

	for i := range maxLoginAttempts {
		fmt.Printf("Attempt-(%d)\n",i + 1)
		var username, password string
		fmt.Print("Enter username: ")
		fmt.Scanln(&username)
		fmt.Print("Enter password: ")
		fmt.Scanln(&password)
		req := Request{Type: Authetication, Username: username, Password: password, context: context.Background()}
		token, err := sendRequest(context.Background(), client, req)
		authToken = token.(string)
		// authToken, err = sendAutheticationRequest(context.Background(), client, username, password)
		if !common.IsEqual(err, common.ErrSuccess){
			log.Printf("Authentication failed: %v", common.ErrorMessage(err))
		} else {
			break
		}
		if i == maxLoginAttempts - 1 {
			os.Exit(0)
		}
	}
	
	log.Println("Authentication Successful")
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", authToken))

	go offlineClientHandler.processRequestQueue(client)

	for {
		var reqType int
		var shouldExit bool = false
		fmt.Printf("Enter Request Type:")
		fmt.Scanln(&reqType)
		switch; RequestType(reqType) {
			case balanceEnquiry:
				req := Request{Type: balanceEnquiry, AuthToken: authToken, context: ctx}
				resp, err := sendRequest(ctx, client, req)
				// currBalance, err := sendGetBalanceRequest(ctx, client, authToken)
				if common.IsEqual(err, common.ErrRequestQueued) {
					log.Printf("Request Stored in the queue")
				} else if !common.IsEqual(err, common.ErrSuccess) {
					log.Printf("Payment failed: %v\n", common.ErrorMessage(err))
				} else{
					log.Printf("Currence balance : %f\n", resp.(float64))
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
				req := Request{Type: makePayment, RecpAccNo: recpAccNo, 
					RecpBankName: recpBankName, AuthToken: authToken, context: ctx, Amount: amount}
				_, err := sendRequest(ctx, client, req)
				// _, err = sendPaymentRequest(ctx, client, amount, recpAccNo, recpBankName, authToken)
				if common.IsEqual(err, common.ErrRequestQueued) {
					log.Printf("Request Stored in the queue")
				} else if !common.IsEqual(err, common.ErrSuccess) {
					log.Printf("Payment failed: %v\n", common.ErrorMessage(err))
				} else{
					log.Println("Payment Status: Success")
				}
			case setOffline:
				offlineClientHandler.setOffline()
			case exit:
				shouldExit = true
			default:
				log.Println("Invalid ReqType") 
		}
		if shouldExit {
			break
		}
	}
}
