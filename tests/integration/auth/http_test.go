package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/soat13/ms-auth/tests/testsupport"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// Setup
// -----------------------------------------------------------------------------

func ensureSetup(t *testing.T) *testsupport.SetupConfig {
	t.Helper()
	return testsupport.SetupHTTP(t, nil)
}

// -----------------------------------------------------------------------------
// DTOs
// -----------------------------------------------------------------------------

type authenticateBody struct {
	CPF      string `json:"cpf"`
	Password string `json:"password"`
}

type userInformation struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Roles []string  `json:"roles"`
}

type authenticateResponse struct {
	AccessToken string          `json:"access_token"`
	TokenType   string          `json:"token_type"`
	ExpiresAt   time.Time       `json:"expires_at"`
	User        userInformation `json:"user"`
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

func TestAuthenticate(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		setup := ensureSetup(t)

		userID := testsupport.ThereIsAUser(
			t,
			setup.DDBClient,
			setup.TableName,
			uuid.Nil,
			"John Mechanic",
			"58457673009",
			"11987654321",
			"mechanic@example.com",
			"TestPass123!",
			[]string{"mechanic"},
		)

		payload := authenticateBody{
			CPF:      "58457673009",
			Password: "TestPass123!",
		}

		resp := postAuthenticate(t, setup.FiberApp, payload)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body authenticateResponse
		decodeJSON(t, resp, &body)

		require.NotEmpty(t, body.AccessToken)
		require.Equal(t, "Bearer", body.TokenType)
		require.False(t, body.ExpiresAt.IsZero())
		require.Equal(t, userID, body.User.ID)
		require.Equal(t, "John Mechanic", body.User.Name)
		require.Contains(t, body.User.Roles, "mechanic")
	})

	t.Run("Success with Multiple Roles", func(t *testing.T) {
		setup := ensureSetup(t)

		userID := testsupport.ThereIsAUser(
			t,
			setup.DDBClient,
			setup.TableName,
			uuid.Nil,
			"Manager User",
			"45352381030",
			"11987654322",
			"manager@example.com",
			"password456",
			[]string{"manager", "attendant"},
		)

		payload := authenticateBody{
			CPF:      "45352381030",
			Password: "password456",
		}

		resp := postAuthenticate(t, setup.FiberApp, payload)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body authenticateResponse
		decodeJSON(t, resp, &body)

		require.NotEmpty(t, body.AccessToken)
		require.Equal(t, "Bearer", body.TokenType)
		require.Equal(t, userID, body.User.ID)
		require.Contains(t, body.User.Roles, "manager")
		require.Contains(t, body.User.Roles, "attendant")
	})

	t.Run("Invalid Credentials - Wrong Password", func(t *testing.T) {
		setup := ensureSetup(t)

		testsupport.ThereIsAUser(
			t,
			setup.DDBClient,
			setup.TableName,
			uuid.Nil,
			"Test User",
			"02383727075",
			"11987654323",
			"test@example.com",
			"correctpassword",
			[]string{"attendant"},
		)

		payload := authenticateBody{
			CPF:      "02383727075",
			Password: "wrongpassword",
		}

		resp := postAuthenticate(t, setup.FiberApp, payload)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid Credentials - User Not Found", func(t *testing.T) {
		setup := ensureSetup(t)

		payload := authenticateBody{
			CPF:      "11144477735",
			Password: "password123",
		}

		resp := postAuthenticate(t, setup.FiberApp, payload)
		require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid Body - Empty CPF", func(t *testing.T) {
		setup := ensureSetup(t)

		payload := authenticateBody{
			CPF:      "",
			Password: "password123",
		}

		resp := postAuthenticate(t, setup.FiberApp, payload)
		require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
	})

	t.Run("Invalid Body - Empty Password", func(t *testing.T) {
		setup := ensureSetup(t)

		payload := authenticateBody{
			CPF:      "58457673009",
			Password: "",
		}

		resp := postAuthenticate(t, setup.FiberApp, payload)
		require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
	})

	t.Run("Invalid Body - CPF Too Short", func(t *testing.T) {
		setup := ensureSetup(t)

		payload := authenticateBody{
			CPF:      "123",
			Password: "password123",
		}

		resp := postAuthenticate(t, setup.FiberApp, payload)
		require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		setup := ensureSetup(t)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := setup.FiberApp.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func postAuthenticate(t *testing.T, fiberApp *fiber.App, body authenticateBody) *http.Response {
	t.Helper()
	bs, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	resp, err := fiberApp.Test(req, -1)
	require.NoError(t, err)
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	require.NoError(t, dec.Decode(v))
}
