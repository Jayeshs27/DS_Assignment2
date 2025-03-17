package main

import (
	"context"
	// "fmt"
	"log"
	"sync"
	"time"

	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"

	"q3/common"
	pb "q3/protofiles"
)

type Request struct {
	Type         RequestType
	Username     string
	Password     string
	AuthToken    string
	Amount       float32
	RecpAccNo    string
	RecpBankName string
	context      context.Context
}

type OfflineHandler struct {
	requestQueue []Request 
	queueMutex sync.Mutex
	setMutex sync.Mutex
	inQue map[Request]bool
	startTime time.Time
	isSetOffline bool
};

func NewOfflineHandler()(*OfflineHandler){
	return &OfflineHandler{
		inQue:make(map[Request]bool),
		isSetOffline:false,
		startTime: time.Now(),
	}
}

func (handler *OfflineHandler) setOffline(){
	handler.setMutex.Lock()
	defer offlineClientHandler.setMutex.Unlock()
	handler.isSetOffline = true
	handler.startTime = time.Now()
	log.Println("Client is Offline Now")
}

func (handler *OfflineHandler) isOffline()(bool){
	handler.setMutex.Lock()
	defer offlineClientHandler.setMutex.Unlock()
	if handler.isSetOffline {
		elaspesdTime := time.Since(handler.startTime)
		if time.Duration(elaspesdTime.Seconds()) >= time.Duration(offlineInterval){
			handler.isSetOffline = false
		}
	}
	return handler.isSetOffline
}

var offlineClientHandler *OfflineHandler
var offlineInterval int = 15

// func isServerDown(err error) bool {
// 	st, ok := status.FromError(err)
// 	if ok {
// 		return st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded
// 	}
// 	return false
// }

func (handler *OfflineHandler) queueRequest(req Request) {
	handler.queueMutex.Lock()
	defer handler.queueMutex.Unlock()
	_, exists := offlineClientHandler.inQue[req]
	if !exists {
		handler.requestQueue = append(handler.requestQueue, req)
	}
}

func (handler *OfflineHandler) deQueueRequest(){
	handler.queueMutex.Lock()
	defer handler.queueMutex.Unlock()
	handler.requestQueue = handler.requestQueue[1:]
}

func sendRequest(ctx context.Context, client pb.PaymentServiceClient, req Request) (any, error) {
	var response any
	var err error

	if offlineClientHandler.isOffline() {
		offlineClientHandler.queueRequest(req)
		return response, common.ErrRequestQueued
	}

	switch req.Type {
		case Authetication:
			response, err = sendAutheticationRequest(ctx, client, req.Username, req.Password)

		case balanceEnquiry:
			response, err = sendGetBalanceRequest(ctx, client, req.AuthToken)

		case makePayment:
			response, err = sendPaymentRequest(ctx, client, req.Amount, req.RecpAccNo, req.RecpBankName, req.AuthToken)
	}
	return response, err
}

func (handler *OfflineHandler) processRequestQueue(client pb.PaymentServiceClient) {
	handler.inQue = make(map[Request]bool)
	for {	
		if handler.isOffline(){
			time.Sleep(3 * time.Second)
			continue
		}
		handler.queueMutex.Lock()
		queueLen := len(handler.requestQueue)
		handler.queueMutex.Unlock()

		if queueLen > 0 {
			print("\n")
			log.Println("Connection Restored, Processing Queued Requests...")
			for {
				handler.queueMutex.Lock()
				if len(handler.requestQueue) == 0 {
					handler.queueMutex.Unlock()
					break
				}
				req := handler.requestQueue[0]
				handler.queueMutex.Unlock()
				
				resp, err := sendRequest(req.context, client, req)
				log.Printf("Processed RequestType: %v", req.Type)
				handler.deQueueRequest()

				if common.IsEqual(err, common.ErrSuccess) {
					if req.Type == balanceEnquiry {
						log.Printf("Response : Success, Current Balance: %f", resp.(float64))
					} else {
						log.Printf("Response : Success")
					}
				} else{
					log.Printf("Response : %v\n", err)
				}
			}
		}
	}
}
