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
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "q3/protofiles"
	common "q3/common"
)

const (
	paymentGatewayAddr = "localhost:45301"
	bankServerAddr = "localhost:45331"
)

var (
	pgLogger *common.Logger
)
// JWT Secret Key
var jwtKey = []byte("payment_gatway_key") 

// User struct
type User struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Role       string `json:"role"`
	AccountNo  string `json:"account_no"`
	BankId     int32 `json:"bank_id"`
}

// PaymentServer struct
type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	users map[string]User
}

func loadUsers(filename string) map[string]User {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read user file: %v", err)
	}
	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		log.Fatalf("Failed to parse user data: %v", err)
	}
	userMap := make(map[string]User)
	for _, user := range users {
		userMap[user.Username] = user
	}
	return userMap
}

func (s *PaymentServer) Authenticate(ctx context.Context, req *pb.UserCredentials) (*pb.AuthResponse, error) {
	user, exists := s.users[req.Username]
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

// Process payment
func (s *PaymentServer) MakePayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	// Validate JWT token
	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		return nil, common.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["role"] != "customer" {
		return nil, common.ErrUnauthorized
	}
	
	return &pb.PaymentResponse{
		Status:  "success",
		Message: "Payment processed successfully",
	}, common.ErrSuccess
}

// func (s *PaymentServer) CheckBalance(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
// 	// Validate JWT token
// 	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
// 		return jwtKey, nil
// 	})

// 	if err != nil || !token.Valid {
// 		return nil, common.ErrInvalidToken
// 	}

// 	claims, ok := token.Claims.(jwt.MapClaims)
// 	if !ok || claims["role"] != "customer" {
// 		return nil, common.ErrUnauthorized
// 	}

// 	return &pb.PaymentResponse{
// 		Status:  "success",
// 		Message: "Payment processed successfully",
// 	}, common.ErrSuccess
// }



func main() {
	users := loadUsers("sample_data/pg_users.json")

	cert, err := tls.LoadX509KeyPair("certs/payment_gateway.crt", "certs/payment_gateway.key")
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
	pgLogger = common.NewLogger("logs/payment_gateway")
	defer pgLogger.Close()
	
	server := grpc.NewServer(
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(loggingInterceptor, authInterceptor),
	)
	
	pb.RegisterPaymentServiceServer(server, &PaymentServer{users: users})

	listener, err := net.Listen("tcp", paymentGatewayAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	fmt.Printf("Payment Gateway running on addr:%s...\n",paymentGatewayAddr)
	server.Serve(listener)
}
