package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	authApp "github.com/soat13/ms-auth/internal/auth/application"
	authDomain "github.com/soat13/ms-auth/internal/auth/domain"
	fiberHelper "github.com/soat13/oficina-utils/pkg/http/fiber"
)

const (
	ClaimsContextKey = "auth.claims"
	UserIDContextKey = "auth.user_id"
)

type Middleware struct {
	tokens            authApp.TokenService
	errorHandler      *fiberHelper.ErrorHandler
	protectedPrefixes []string
	publicRoutes      []PublicRoute
}

type PublicRoute struct {
	Method string
	Path   string
}

func NewMiddleware(tokens authApp.TokenService, errorHandler *fiberHelper.ErrorHandler, prefixes []string, publicRoutes []PublicRoute) *Middleware {
	if len(prefixes) == 0 {
		prefixes = []string{"/"}
	}

	m := &Middleware{
		tokens:            tokens,
		errorHandler:      errorHandler,
		protectedPrefixes: normalizePrefixes(prefixes),
		publicRoutes:      normalizePublicRoutes(publicRoutes),
	}

	m.errorHandler.ErrorResolver.RegisterHTTPUnauthorizedError(authDomain.ErrMissingToken)
	m.errorHandler.ErrorResolver.RegisterHTTPUnauthorizedError(authDomain.ErrInvalidToken)
	m.errorHandler.ErrorResolver.RegisterHTTPUnauthorizedError(authDomain.ErrExpiredToken)

	return m
}

func (m *Middleware) Handle(ctx *fiber.Ctx) error {
	if !m.shouldProtect(ctx.Method(), ctx.Path()) {
		return ctx.Next()
	}

	rawToken := ctx.Get("Authorization")
	if strings.TrimSpace(rawToken) == "" {
		return m.errorHandler.Handle(ctx, authDomain.ErrMissingToken)
	}

	token, err := extractBearerToken(rawToken)
	if err != nil {
		return m.errorHandler.Handle(ctx, authDomain.ErrInvalidToken)
	}

	claims, err := m.tokens.Validate(ctx.Context(), token)
	if err != nil {
		return m.errorHandler.Handle(ctx, err)
	}

	ctx.Locals(ClaimsContextKey, claims)
	ctx.Locals(UserIDContextKey, claims.UserID)

	return ctx.Next()
}

func (m *Middleware) shouldProtect(method, path string) bool {
	if strings.EqualFold(method, fiber.MethodOptions) {
		return false
	}

	requestPath := normalizePath(path)
	requestMethod := strings.ToUpper(method)

	for _, route := range m.publicRoutes {
		if route.matches(requestMethod, requestPath) {
			return false
		}
	}

	for _, prefix := range m.protectedPrefixes {
		if strings.HasPrefix(requestPath, prefix) {
			return true
		}
	}
	return false
}

func extractBearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 {
		return "", authDomain.ErrInvalidToken
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", authDomain.ErrInvalidToken
	}

	return parts[1], nil
}

func normalizePrefixes(prefixes []string) []string {
	out := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		out = append(out, normalizePath(prefix))
	}
	return out
}

func normalizePublicRoutes(routes []PublicRoute) []PublicRoute {
	out := make([]PublicRoute, 0, len(routes))
	for _, route := range routes {
		out = append(out, PublicRoute{
			Method: strings.ToUpper(route.Method),
			Path:   normalizePath(route.Path),
		})
	}
	return out
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func (r PublicRoute) matches(method, path string) bool {
	if r.Path != path {
		return false
	}
	if strings.TrimSpace(r.Method) == "" {
		return true
	}
	return r.Method == method
}
