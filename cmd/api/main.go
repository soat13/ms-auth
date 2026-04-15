package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/soat13/oficina-auth/assets/docs"
	"github.com/soat13/oficina-auth/internal/auth/application"
	httpAdapter "github.com/soat13/oficina-auth/internal/auth/infra/in/http"
	authDdb "github.com/soat13/oficina-auth/internal/auth/infra/out/dynamodb"
	jwtAdapter "github.com/soat13/oficina-auth/internal/auth/infra/out/jwt"
	userUseCase "github.com/soat13/oficina-auth/internal/user/application"
	userHttp "github.com/soat13/oficina-auth/internal/user/infra/in/http"
	userDdb "github.com/soat13/oficina-auth/internal/user/infra/out/dynamodb"
	errorHelper "github.com/soat13/oficina-utils/pkg/error"
	fiberHelper "github.com/soat13/oficina-utils/pkg/http/fiber"
)

func main() {
	cfg := loadConfig()

	// ── AWS / DynamoDB ──────────────────────────────────────────────────────
	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(cfg.awsRegion),
	)
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(awsCfg)
	authUserRepo := authDdb.NewUserRepository(ddbClient, cfg.tableName, cfg.documentIndex)
	userRepo := userDdb.NewUserRepository(ddbClient, cfg.tableName, cfg.emailIndex, cfg.documentIndex)

	// ── JWT Token Service ───────────────────────────────────────────────────
	tokenSvc, err := jwtAdapter.NewTokenService(cfg.jwtSecret, cfg.jwtExpiration)
	if err != nil {
		log.Fatalf("init JWT service: %v", err)
	}

	// ── Use Cases ────────────────────────────────────────────────────────────
	authenticateUC := application.NewAuthenticateUser(authUserRepo, tokenSvc)

	createUserUC := userUseCase.NewCreateUser(userRepo)
	updateUserUC := userUseCase.NewUpdateUser(userRepo)
	deleteUserUC := userUseCase.NewDeleteUser(userRepo)
	getUserUC := userUseCase.NewGetUser(userRepo)
	listUsersUC := userUseCase.NewListUsers(userRepo)

	// ── HTTP (Fiber) ─────────────────────────────────────────────────────────
	validate := validator.New()
	errorResolver := errorHelper.NewErrorResolver()
	errHandler := fiberHelper.NewErrorHandler(*errorResolver, validate)

	app := fiber.New(fiber.Config{
		AppName:      "oficina-auth",
		ErrorHandler: func(c *fiber.Ctx, err error) error { return errHandler.Handle(c, err) },
	})

	app.Use(recover.New())

	middleware := httpAdapter.NewMiddleware(
		tokenSvc,
		errHandler,
		[]string{"/"},
		[]httpAdapter.PublicRoute{
			{Method: fiber.MethodPost, Path: "/auth/login"},
			{Method: fiber.MethodGet, Path: "/docs"},
			{Method: fiber.MethodGet, Path: "/openapi.yaml"},
			{Method: fiber.MethodGet, Path: "/health"},
		},
	)
	app.Use(middleware.Handle)

	// Register routes
	authHandler := httpAdapter.NewHandler(authenticateUC, validate, errHandler)
	httpAdapter.Register(app, authHandler)

	userHandler := userHttp.NewHandler(createUserUC, updateUserUC, deleteUserUC, getUserUC, listUsersUC, validate, errHandler)
	userHttp.Register(app, userHandler)

	docs.Register(app)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	addr := fmt.Sprintf(":%d", cfg.httpPort)
	log.Printf("oficina-auth listening on %s", addr)
	log.Fatal(app.Listen(addr))
}

// ── Config ──────────────────────────────────────────────────────────────────

type appConfig struct {
	awsRegion     string
	tableName     string
	emailIndex    string
	documentIndex string
	jwtSecret     string
	jwtExpiration time.Duration
	httpPort      int
}

func loadConfig() appConfig {
	cfg := appConfig{
		awsRegion:     envOrDefault("AWS_REGION", "us-east-1"),
		tableName:     envOrDefault("DYNAMODB_TABLE", "users"),
		emailIndex:    envOrDefault("DYNAMODB_EMAIL_INDEX", "email-index"),
		documentIndex: envOrDefault("DYNAMODB_DOCUMENT_INDEX", "document-index"),
		jwtSecret:     mustEnv("JWT_SECRET"),
		jwtExpiration: parseDuration(envOrDefault("JWT_EXPIRATION", "1h")),
		httpPort:      parseInt(envOrDefault("HTTP_PORT", "8090")),
	}
	return cfg
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %q is not set", key)
	}
	return v
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("invalid duration %q: %v", s, err)
	}
	return d
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("invalid int %q: %v", s, err)
	}
	return n
}
