package http

import (
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	authApp "github.com/soat13/oficina-auth/internal/auth/application"
	fiberHelper "github.com/soat13/oficina-utils/pkg/http/fiber"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
)

// ErrInvalidJSON is returned when the request body cannot be parsed as JSON.
var ErrInvalidJSON = errors.New("invalid json body")

type Handler struct {
	authenticate *authApp.AuthenticateUser
	validator    *validator.Validate
	errorHandler *fiberHelper.ErrorHandler
}

func NewHandler(
	authenticate *authApp.AuthenticateUser,
	validator *validator.Validate,
	errorHandler *fiberHelper.ErrorHandler,
) *Handler {
	handler := &Handler{
		authenticate: authenticate,
		validator:    validator,
		errorHandler: errorHandler,
	}

	handler.errorHandler.ErrorResolver.RegisterHTTPUnauthorizedError(authApp.ErrInvalidCredentials)
	handler.errorHandler.ErrorResolver.RegisterHTTPBadRequestError(ErrInvalidJSON)
	handler.errorHandler.ErrorResolver.RegisterHTTPBadRequestError(document.ErrInvalidDocument)

	return handler
}

func Register(app *fiber.App, h *Handler) {
	app.Post("/auth/login", h.login)
}

type loginBody struct {
	CPF      string `json:"cpf"      validate:"required,min=11"`
	Password string `json:"password" validate:"required,min=8"`
}

type authenticatedUserJSON struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Roles []string  `json:"roles"`
}

type loginResponse struct {
	AccessToken string                `json:"access_token"`
	TokenType   string                `json:"token_type"`
	ExpiresAt   time.Time             `json:"expires_at"`
	User        authenticatedUserJSON `json:"user"`
}

func (h *Handler) login(ctx *fiber.Ctx) error {
	var body loginBody
	if err := ctx.BodyParser(&body); err != nil {
		return h.errorHandler.Handle(ctx, ErrInvalidJSON)
	}
	if err := h.validator.Struct(body); err != nil {
		return h.errorHandler.Handle(ctx, err)
	}

	docVO, err := document.New(body.CPF)
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}

	out, err := h.authenticate.Execute(ctx.Context(), authApp.AuthenticateInput{
		Document: docVO,
		Password: body.Password,
	})
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}

	resp := loginResponse{
		AccessToken: out.Token.Value,
		TokenType:   "Bearer",
		ExpiresAt:   out.Token.ExpiresAt,
		User: authenticatedUserJSON{
			ID:    out.User.ID,
			Name:  out.User.Name,
			Email: out.User.Email,
			Roles: out.User.Roles.Strings(),
		},
	}

	return ctx.Status(fiber.StatusOK).JSON(resp)
}
