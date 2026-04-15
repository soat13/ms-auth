package testsupport

import (
	"context"
	"testing"

	"github.com/google/uuid"
	userDdb "github.com/soat13/oficina-auth/internal/user/infra/out/dynamodb"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/soat13/oficina-auth/internal/shared/authz"
	"github.com/soat13/oficina-auth/internal/user/domain"
	"github.com/stretchr/testify/require"
)

// ThereIsAUser cria um usuário diretamente no DynamoDB via repositório real.
// Retorna o UUID do usuário criado.
//
// Parâmetros:
//   - id: se uuid.Nil, o repositório gera um UUID automaticamente
//   - cpf: CPF numérico (11 dígitos), ex: "52998224725"
//   - phoneNumber: número de telefone, ex: "11987654321"
func ThereIsAUser(
	t *testing.T,
	ddbClient *dynamodb.Client,
	tableName string,
	id uuid.UUID,
	name, cpf, phoneNumber, emailAddr, rawPassword string,
	roles []string,
) uuid.UUID {
	t.Helper()

	docVO, err := document.New(cpf)
	require.NoError(t, err, "CPF inválido: %s", cpf)

	phoneVO, err := phone.New(phoneNumber)
	require.NoError(t, err, "telefone inválido: %s", phoneNumber)

	emailVO, err := email.New(emailAddr)
	require.NoError(t, err, "email inválido: %s", emailAddr)

	passwordVO, err := password.New(rawPassword)
	require.NoError(t, err, "password inválido")

	rolesVO, err := authz.NewRoles(roles)
	require.NoError(t, err, "roles inválidos: %v", roles)

	user, err := domain.NewUser(id, name, docVO, phoneVO, emailVO, passwordVO, rolesVO)
	require.NoError(t, err)

	repo := userDdb.NewUserRepository(ddbClient, tableName, "email-index", "document-index")
	require.NoError(t, repo.Create(context.Background(), user))

	return user.ID
}
