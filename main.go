package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	_ = godotenv.Load()
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL is required")
	}

	store, err := NewRedisOTPStore(redisURL)
	if err != nil {
		log.Fatalf("configure Redis: %v", err)
	}
	defer store.Close()

	pingContext, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	if err := store.Ping(pingContext); err != nil {
		cancelPing()
		log.Fatalf("connect to Redis: %v", err)
	}
	cancelPing()

	sms, err := NewSMSClient(os.Getenv("SMS_API_URL"), os.Getenv("SMS_API_KEY"), os.Getenv("SMS_SENDER_ID"))
	if err != nil {
		log.Fatalf("configure SMS provider: %v", err)
	}

	address := os.Getenv("GRPC_ADDR")
	if address == "" {
		address = ":50051"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("listen on %s: %v", address, err)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(newRateLimiter().UnaryInterceptor()))
	RegisterOTPServiceServer(server, NewOTPService(store, sms))

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownContext.Done()
		server.GracefulStop()
	}()
	log.Printf("OTP gRPC server listening on %s", address)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("serve gRPC: %v", err)
	}
}