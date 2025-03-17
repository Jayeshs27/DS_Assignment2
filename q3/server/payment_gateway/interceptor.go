package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	common "q3/common"
)

func logger(format string, a ...any) {
	timestamp := time.Now().Format(time.RFC3339) // YYYY-MM-DD HH:MM:SS
	fmt.Printf("[%s] %s\n", timestamp, fmt.Sprintf(format, a...))
}

// AuthInterceptor is a gRPC interceptor for authentication and authorization
func authInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {

	if strings.Contains(info.FullMethod, "Authenticate") {
		return handler(ctx, req)
	}

	if strings.Contains(info.FullMethod, "BankServerDiscovery") {
		return handler(ctx, req)
	}
	// Extract token from context metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	authHeader, exists := md["authorization"]
	if !exists || len(authHeader) == 0 {
		return nil, fmt.Errorf("missing authorization token")
	}

	tokenString := authHeader[0]
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(jwtKey), common.ErrSuccess
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return common.ErrSuccess, fmt.Errorf("unauthorized")
	}

	// Role-based authorization
	role := claims["role"].(string)
	method := info.FullMethod

	// Define role permissions
	permissions := map[string][]string{
		"/payment.PaymentService/ViewBalance":          {"customer"},
		"/payment.PaymentService/ViewTransactionHistory": {"customer"},
		"/payment.PaymentService/MakePayment":          {"customer"},
	}

	allowedRoles, exists := permissions[method]
	if exists {
		authorized := false
		for _, r := range allowedRoles {
			if r == role {
				authorized = true
				break
			}
		}
		if !authorized {
			return nil, fmt.Errorf("access denied")
		}
	}
	// Proceed with the request
	return handler(ctx, req)
}

func pgLoggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, 
	handler grpc.UnaryHandler) (any, error) {
	
	pgLogger.PrintLog("Method: %s, Request: %v", info.FullMethod, req)
    resp, err := handler(ctx, req)
    if err != nil {
        logger("RPC failed with error: %v", err)
    }
	pgLogger.PrintLog("Method: %s, Response: %v", info.FullMethod, resp)
	
    return resp, err
}
