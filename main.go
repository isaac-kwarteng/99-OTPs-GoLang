package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
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

	address := os.Getenv("CONNECT_ADDR")
	if address == "" {
		address = ":50051"
	}
	mux := http.NewServeMux()
	RegisterOTPServiceHandler(mux, NewOTPService(store, sms), newRateLimiter().UnaryInterceptor())
	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownContext.Done()
		shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()
	log.Printf("OTP Connect RPC server listening on %s", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve Connect RPC: %v", err)
	}
}
