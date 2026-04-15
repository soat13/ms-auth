package http_test

import (
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/soat13/oficina-auth/internal/auth/application/mocks"
	"github.com/soat13/oficina-auth/internal/auth/domain"
	httpAdapter "github.com/soat13/oficina-auth/internal/auth/infra/in/http"
	errorHelper "github.com/soat13/oficina-utils/pkg/error"
	fiberHelper "github.com/soat13/oficina-utils/pkg/http/fiber"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	commonCtrl := gomock.NewController(t)
	defer commonCtrl.Finish()

	commonTokenSvc := mocks.NewMockTokenService(commonCtrl)

	commonResolver := errorHelper.NewErrorResolver()
	commonErrHandler := fiberHelper.NewErrorHandler(*commonResolver, validator.New())

	commonMiddleware := httpAdapter.NewMiddleware(commonTokenSvc, commonErrHandler, []string{"/protected"}, nil)

	app := fiber.New()
	app.Use(commonMiddleware.Handle)

	app.Get("/protected", func(c *fiber.Ctx) error {
		require.NotNil(t, c.Locals(httpAdapter.ClaimsContextKey))
		require.NotNil(t, c.Locals(httpAdapter.UserIDContextKey))
		return c.SendStatus(fiber.StatusOK)
	})

	t.Run("should return 401 when missing token", func(t *testing.T) {
		req := httptest.NewRequest(fiber.MethodGet, "/protected", nil)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("should return 401 when invalid header", func(t *testing.T) {
		req := httptest.NewRequest(fiber.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Basic abc")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("should return 200 when token is valid", func(t *testing.T) {
		commonTokenSvc.EXPECT().
			Validate(gomock.Any(), "token-ok").
			Return(domain.Claims{UserID: uuid.Nil}, nil)

		req := httptest.NewRequest(fiber.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer token-ok")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("should return 401 when token service says expired", func(t *testing.T) {
		commonTokenSvc.EXPECT().
			Validate(gomock.Any(), "token-expired").
			Return(domain.Claims{}, domain.ErrExpiredToken)

		req := httptest.NewRequest(fiber.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer token-expired")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("should return skips OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest(fiber.MethodOptions, "/protected", nil)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		require.NotEqual(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("should return 401 when authorization token has too many parts", func(t *testing.T) {
		req := httptest.NewRequest(fiber.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer token extra")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("should not protect route outside prefix", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tokenService := mocks.NewMockTokenService(ctrl)
		resolver := errorHelper.NewErrorResolver()
		handler := fiberHelper.NewErrorHandler(*resolver, validator.New())

		middleware := httpAdapter.NewMiddleware(tokenService, handler, []string{"/protected"}, nil)

		appFiber := fiber.New()
		appFiber.Use(middleware.Handle)
		appFiber.Get("/free", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(fiber.MethodGet, "/free", nil)
		resp, err := appFiber.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("should default prefixes to / when empty", func(t *testing.T) {
		privateCtrl := gomock.NewController(t)
		defer privateCtrl.Finish()

		privateTokenService := mocks.NewMockTokenService(privateCtrl)
		privateResolver := errorHelper.NewErrorResolver()
		privateErrHandler := fiberHelper.NewErrorHandler(*privateResolver, validator.New())

		privateMiddleware := httpAdapter.NewMiddleware(privateTokenService, privateErrHandler, nil, nil)

		privateApp := fiber.New()
		privateApp.Use(privateMiddleware.Handle)
		privateApp.Get("/any", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(fiber.MethodGet, "/any", nil)
		resp, err := privateApp.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("should protect all routes when prefix is empty string (covers normalizePath path == '')", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tokenService := mocks.NewMockTokenService(ctrl)
		resolver := errorHelper.NewErrorResolver()
		errorHandler := fiberHelper.NewErrorHandler(*resolver, validator.New())

		middleware := httpAdapter.NewMiddleware(tokenService, errorHandler, []string{""}, nil)

		appFiber := fiber.New()
		appFiber.Use(middleware.Handle)
		appFiber.Get("/any", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(fiber.MethodGet, "/any", nil)
		resp, err := appFiber.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("should normalize prefix without leading slash (covers normalizePath adds '/')", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tokenService := mocks.NewMockTokenService(ctrl)
		resolver := errorHelper.NewErrorResolver()
		errorHandler := fiberHelper.NewErrorHandler(*resolver, validator.New())

		newMiddleware := httpAdapter.NewMiddleware(tokenService, errorHandler, []string{"protected"}, nil)

		appFiber := fiber.New()
		appFiber.Use(newMiddleware.Handle)
		appFiber.Get("/protected", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(fiber.MethodGet, "/protected", nil)
		resp, err := appFiber.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("should skip auth when public route method is empty (covers PublicRoute.matches Method == '')", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tokenService := mocks.NewMockTokenService(ctrl)
		resolver := errorHelper.NewErrorResolver()
		errorHandler := fiberHelper.NewErrorHandler(*resolver, validator.New())

		publicRoutes := []httpAdapter.PublicRoute{
			{Method: "", Path: "/public"},
		}

		newMiddleware := httpAdapter.NewMiddleware(tokenService, errorHandler, []string{"/"}, publicRoutes)

		appFiber := fiber.New()
		appFiber.Use(newMiddleware.Handle)
		appFiber.Post("/public", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(fiber.MethodPost, "/public", nil)
		resp, err := appFiber.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("should require auth when public route method does not match (covers PublicRoute.matches Method != request)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		tokenService := mocks.NewMockTokenService(ctrl)
		errorResolver := errorHelper.NewErrorResolver()
		errorHandler := fiberHelper.NewErrorHandler(*errorResolver, validator.New())

		publicRoutes := []httpAdapter.PublicRoute{
			{Method: fiber.MethodGet, Path: "/public"},
		}

		middleware := httpAdapter.NewMiddleware(tokenService, errorHandler, []string{"/"}, publicRoutes)

		appFiber := fiber.New()
		appFiber.Use(middleware.Handle)
		appFiber.Post("/public", func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest(fiber.MethodPost, "/public", nil)
		resp, err := appFiber.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("should return 401 when bearer token is empty", func(t *testing.T) {
		req := httptest.NewRequest(fiber.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer   ")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})
}
