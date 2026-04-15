package testsupport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	authApplication "github.com/soat13/oficina-auth/internal/auth/application"
	authHttp "github.com/soat13/oficina-auth/internal/auth/infra/in/http"
	authDdb "github.com/soat13/oficina-auth/internal/auth/infra/out/dynamodb"
	jwtAdapter "github.com/soat13/oficina-auth/internal/auth/infra/out/jwt"
	userApplication "github.com/soat13/oficina-auth/internal/user/application"
	userHttp "github.com/soat13/oficina-auth/internal/user/infra/in/http"
	userDdb "github.com/soat13/oficina-auth/internal/user/infra/out/dynamodb"
	errorHelper "github.com/soat13/oficina-utils/pkg/error"
	fiberHelper "github.com/soat13/oficina-utils/pkg/http/fiber"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret   = "test-secret-key-for-integration-tests"
	testJWTExpiry   = time.Hour
)

func getTestDDBEndpoint() string {
	if ep := os.Getenv("DYNAMODB_ENDPOINT"); ep != "" {
		return ep
	}
	return "http://localhost:8000"
}

// SetupConfig é o contrato exposto aos testes de integração.
type SetupConfig struct {
	FiberApp  *fiber.App
	DDBClient *dynamodb.Client
	TableName string
	AuthToken string // token Bearer de um usuário admin pré-criado
}

// SetupHTTP cria um ambiente de teste completo.
//
// O parâmetro register permite que cada suite registre seus próprios handlers
// além dos já configurados internamente (auth + user).
func SetupHTTP(t *testing.T, register func(app *fiber.App, ddb *dynamodb.Client)) *SetupConfig {
	t.Helper()

	// ── DynamoDB Local ──────────────────────────────────────────────────────
	ddbClient := newTestDDBClient(t)
	tableName := "users-" + uuid.New().String()[:8]

	createTestTable(t, ddbClient, tableName)
	t.Cleanup(func() { dropTestTable(ddbClient, tableName) })

	// ── Repositórios ────────────────────────────────────────────────────────
	authUserRepo := authDdb.NewUserRepositoryWithClient(ddbClient, tableName, "document-index")
	userRepo := userDdb.NewUserRepository(ddbClient, tableName, "email-index", "document-index")

	// ── JWT ─────────────────────────────────────────────────────────────────
	tokenSvc, err := jwtAdapter.NewTokenService(testJWTSecret, testJWTExpiry)
	require.NoError(t, err)

	// ── Use Cases ────────────────────────────────────────────────────────────
	authenticateUC := authApplication.NewAuthenticateUser(authUserRepo, tokenSvc)

	createUserUC := userApplication.NewCreateUser(userRepo)
	updateUserUC := userApplication.NewUpdateUser(userRepo)
	deleteUserUC := userApplication.NewDeleteUser(userRepo)
	getUserUC := userApplication.NewGetUser(userRepo)
	listUsersUC := userApplication.NewListUsers(userRepo)

	// ── Fiber ────────────────────────────────────────────────────────────────
	validate := validator.New()
	errorResolver := errorHelper.NewErrorResolver()
	errHandler := fiberHelper.NewErrorHandler(*errorResolver, validate)

	app := fiber.New(fiber.Config{
		AppName:      "oficina-auth-test",
		ErrorHandler: func(c *fiber.Ctx, err error) error { return errHandler.Handle(c, err) },
	})
	app.Use(recover.New())

	middleware := authHttp.NewMiddleware(
		tokenSvc,
		errHandler,
		[]string{"/"},
		[]authHttp.PublicRoute{
			{Method: fiber.MethodPost, Path: "/auth/login"},
		},
	)
	app.Use(middleware.Handle)

	authHandler := authHttp.NewHandler(authenticateUC, validate, errHandler)
	authHttp.Register(app, authHandler)

	userHandler := userHttp.NewHandler(createUserUC, updateUserUC, deleteUserUC, getUserUC, listUsersUC, validate, errHandler)
	userHttp.Register(app, userHandler)

	if register != nil {
		register(app, ddbClient)
	}

	// ── Admin pré-autenticado ─────────────────────────────────────────────
	adminCPF := "52998224725"
	ThereIsAUser(t, ddbClient, tableName, uuid.Nil,
		"Admin Test", adminCPF, "11900000000", "admin@test.com", "adminpass1", []string{"manager"},
	)
	token := authenticateViaHTTP(t, app, adminCPF, "adminpass1")

	return &SetupConfig{
		FiberApp:  app,
		DDBClient: ddbClient,
		TableName: tableName,
		AuthToken: token,
	}
}

// ── DynamoDB helpers ─────────────────────────────────────────────────────────

func newTestDDBClient(t *testing.T) *dynamodb.Client {
	t.Helper()
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "test", SecretAccessKey: "test"}, nil
		})),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: getTestDDBEndpoint(), HostnameImmutable: true}, nil
			}),
		),
	)
	require.NoError(t, err)
	return dynamodb.NewFromConfig(cfg)
}

func createTestTable(t *testing.T, client *dynamodb.Client, tableName string) {
	t.Helper()

	_, err := client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName:   aws.String(tableName),
		BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("email"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("document"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("email-index"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("email"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: aws.String("document-index"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("document"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
	})
	require.NoError(t, err)
}

func dropTestTable(client *dynamodb.Client, tableName string) {
	_, _ = client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
}

// ── Auth helper interno ───────────────────────────────────────────────────────

func authenticateViaHTTP(t *testing.T, app *fiber.App, cpf, password string) string {
	t.Helper()

	body := map[string]string{"cpf": cpf, "password": password}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode, "falha ao pré-autenticar usuário admin no setup")

	var response struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	require.NotEmpty(t, response.AccessToken)

	return response.AccessToken
}
