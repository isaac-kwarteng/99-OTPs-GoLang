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

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/validate"
	"github.com/Vigil-HQ/domecore-go/domecore"
	_ "github.com/joho/godotenv/autoload"
	"github.com/marketdome/otp-api/gen/marketdome/otp/v1/otpv1connect"
	"github.com/marketdome/otp-api/pkg/config"
	"github.com/marketdome/otp-api/pkg/data/repositories"
	"github.com/marketdome/otp-api/pkg/database"
	"github.com/marketdome/otp-api/pkg/interceptors"
	"github.com/marketdome/otp-api/pkg/routes"
	"github.com/marketdome/otp-api/pkg/servers"
	"github.com/marketdome/otp-api/pkg/services"
	"github.com/marketdome/otp-api/thirdparty/sms"
	"golang.org/x/net/http2/h2c"
)

var cfg = config.New()

func main() {
	// Get server address
	serverAddress := cfg.GetServerAddr()

	logger := cfg.Logger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Engines
	redisClient, err := database.NewRedisClient(ctx, cfg.GetRedisURL())
	if err != nil {
		log.Println("failed to connect redis:", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	smsSender, err := sms.New(sms.Config{
		Provider: os.Getenv("SMS.PROVIDER"),
		URL:      os.Getenv("SMS.API_URL"),
		APIKey:   os.Getenv("SMS.API_KEY"),
		Username: os.Getenv("SMS.USERNAME"),
		Password: os.Getenv("SMS.PASSWORD"),
		SenderID: os.Getenv("SMS.SENDER_ID"),
	})
	if err != nil {
		log.Println("failed to configure SMS provider:", err)
		os.Exit(1)
	}
	log.Println("SMS provider:", os.Getenv("SMS.PROVIDER"))

	// OTP endpoints are called before sign-in, so no authenticator is attached
	middleware := domecore.NewMiddleware(nil, logger, nil, nil)

	mux := http.NewServeMux()

	otpRepository := repositories.NewOTPRepository(redisClient)
	otpService := services.NewOTPService(otpRepository, smsSender)
	otpServer := servers.NewOTPServer(otpService)

	// Rate limit interceptor
	rateLimitInterceptor := interceptors.NewRateLimiter(routes.RateLimits).UnaryInterceptor()

	// Validation interceptor
	validateInterceptor := validate.NewInterceptor()

	interceptorOptions := connect.WithInterceptors(rateLimitInterceptor, validateInterceptor)
	path, handler := otpv1connect.NewOTPServiceHandler(otpServer, interceptorOptions)

	corsMiddleware := middleware.Cors(handler)
	reflector := grpcreflect.NewStaticReflector(otpv1connect.OTPServiceName)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	mux.Handle(path, corsMiddleware)

	server := &http.Server{
		Addr:              serverAddress,
		Handler:           h2c.NewHandler(mux, cfg.Http2()),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Println("graceful shutdown failed:", err)
		}
	}()

	log.Println("🚀🚀 gRPC server started on port:", serverAddress)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Println("failed to start server:", err)
		os.Exit(1)
	}
}
