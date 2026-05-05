package bootstrap

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	authApp "github.com/soat13/ms-auth/internal/auth/application"
	jwtAdapter "github.com/soat13/ms-auth/internal/auth/infra/out/jwt"
	helper "github.com/soat13/oficina-utils/pkg/http/fiber"
	"github.com/soat13/oficina-utils/pkg/observability"
	awstrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go-v2/aws"
)

type (
	Config struct {
		AppEnv        string
		AWSRegion     string
		TableName     string
		EmailIndex    string
		DocumentIndex string
		JWTSecret     string
		JWTExpiration time.Duration
		HTTPPort      int
		DynamoDBEndpoint string
	}

	Container struct {
		DDBClient         *dynamodb.Client
		FiberApp          *fiber.App
		FiberErrorHandler *helper.ErrorHandler
		Validator         *validator.Validate
		TokenService      authApp.TokenService
		Metrics           *observability.Metrics
		Cfg               Config
	}
)

func BuildDefault() *Container {
	return Build(nil, nil, nil, nil, nil)
}

func Build(
	ddbClient *dynamodb.Client,
	fiberApp *fiber.App,
	fiberErrorHandler *helper.ErrorHandler,
	structValidator *validator.Validate,
	tokenService authApp.TokenService,
) *Container {
	cfg := loadConfig()

	if fiberApp == nil {
		fiberApp = newApp()
	}

	if structValidator == nil {
		structValidator = validator.New()
	}

	if fiberErrorHandler == nil {
		fiberErrorHandler = helper.NewErrorHandler(*NewErrorResolver(), structValidator)
	}

	if ddbClient == nil {
		ddbClient = newDDB(cfg)
	}

	if tokenService == nil {
		ts, err := jwtAdapter.NewTokenService(cfg.JWTSecret, cfg.JWTExpiration)
		if err != nil {
			log.Fatalf("init JWT service: %v", err)
		}
		tokenService = ts
	}

	return &Container{
		DDBClient:         ddbClient,
		FiberApp:          fiberApp,
		FiberErrorHandler: fiberErrorHandler,
		Validator:         structValidator,
		TokenService:      tokenService,
		Cfg:               cfg,
	}
}

func newDDB(cfg Config) *dynamodb.Client {
	loadOpts := []func(*awsConfig.LoadOptions) error{
		awsConfig.WithRegion(cfg.AWSRegion),
	}

	if cfg.AppEnv == "local" || cfg.DynamoDBEndpoint != "" {
		loadOpts = append(loadOpts,
			awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")),
		)
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}

	awstrace.AppendMiddleware(&awsCfg, awstrace.WithErrorCheck(func(err error) bool {
		var ccf *dynamodbtypes.ConditionalCheckFailedException
		return !errors.As(err, &ccf)
	}))

	return dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		if cfg.DynamoDBEndpoint != "" {
			o.BaseEndpoint = aws.String(cfg.DynamoDBEndpoint)
		}
	})
}

func newApp() *fiber.App {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // todo: In production, set specific allowed origins
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS,HEAD",
		AllowHeaders: "*",
		MaxAge:       3600,
	}))

	app.Use(func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	})
	return app
}

func loadConfig() Config {
	appEnv := envOrDefault("APP_ENV", "local")
	
	// If APP_ENV is local and DYNAMODB_ENDPOINT is not set, default to localhost
	ddbEndpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if appEnv == "local" && ddbEndpoint == "" {
		ddbEndpoint = "http://localhost:8000"
	}

	return Config{
		AppEnv:        appEnv,
		JWTSecret:     mustEnv("JWT_SECRET"),
		AWSRegion:     envOrDefault("AWS_REGION", "us-east-1"),
		TableName:     envOrDefault("DYNAMODB_TABLE", "users"),
		EmailIndex:    envOrDefault("DYNAMODB_EMAIL_INDEX", "email-index"),
		DocumentIndex: envOrDefault("DYNAMODB_DOCUMENT_INDEX", "document-index"),
		JWTExpiration: parseDuration(envOrDefault("JWT_EXPIRATION", "1h")),
		HTTPPort:      parseInt(envOrDefault("PORT", "8090")),
		DynamoDBEndpoint: ddbEndpoint,
	}
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
