package testsupport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// Authenticate faz POST em /auth/login com cpf+password e retorna o access_token.
func Authenticate(t *testing.T, app *fiber.App, cpf, password string) string {
	t.Helper()

	body := map[string]string{
		"cpf":      cpf,
		"password": password,
	}

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	require.NotEmpty(t, response.AccessToken)

	return response.AccessToken
}

// AddAuthHeader injeta o header Authorization: Bearer <token> na requisição.
func AddAuthHeader(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
}
