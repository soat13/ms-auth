package auth

import (
	"github.com/gofiber/fiber/v2"
	authApp "github.com/soat13/ms-auth/internal/auth/application"
	authHTTP "github.com/soat13/ms-auth/internal/auth/infra/in/http"
	authDDB "github.com/soat13/ms-auth/internal/auth/infra/out/dynamodb"
	"github.com/soat13/ms-auth/internal/bootstrap"
)

func SetupDefault(container *bootstrap.Container) {
	Setup(container, nil)
}

func Setup(container *bootstrap.Container, userReader authApp.UserReader) {
	if userReader == nil {
		userReader = authDDB.NewUserRepository(
			container.DDBClient,
			container.Cfg.TableName,
			container.Cfg.DocumentIndex,
		)
	}

	authenticate := authApp.NewAuthenticateUser(userReader, container.TokenService)

	middleware := authHTTP.NewMiddleware(
		container.TokenService,
		container.FiberErrorHandler,
		[]string{"/"},
		[]authHTTP.PublicRoute{
			{Method: fiber.MethodPost, Path: "/auth/login"},
			{Method: fiber.MethodGet, Path: "/docs"},
			{Method: fiber.MethodGet, Path: "/openapi.yaml"},
			{Method: fiber.MethodGet, Path: "/favicon.ico"},
			{Method: fiber.MethodGet, Path: "/health"},
			{Method: fiber.MethodGet, Path: "/health/ready"},
			{Method: fiber.MethodGet, Path: "/health/live"},
			{Method: fiber.MethodGet, Path: "/health/startup"},
		},
	)
	container.FiberApp.Use(middleware.Handle)

	handler := authHTTP.NewHandler(authenticate, container.Validator, container.FiberErrorHandler)
	authHTTP.Register(container.FiberApp, handler)
}
