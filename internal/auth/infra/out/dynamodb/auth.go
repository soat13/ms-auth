package dynamodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/soat13/oficina-auth/internal/auth/application"
	"github.com/soat13/oficina-auth/internal/shared/authz"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
)

// ErrInvalidUserItem is returned when a DynamoDB item cannot be mapped to a valid UserView.
var ErrInvalidUserItem = errors.New("invalid user item from dynamodb")

// QueryClient is the subset of the DynamoDB client API required by this adapter.
// It is an interface to enable unit testing without a real AWS connection.
type QueryClient interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// userItem mirrors the DynamoDB item structure for the users table.
//
// DynamoDB table design:
//   - Primary Key: id (S) — UUID string
//   - GSI "document-index": PK = document (S)
//
// All timestamp fields (created_at, updated_at) are stored as RFC3339 strings
// but are not needed by the auth service, so they are omitted here.
type userItem struct {
	ID           string   `dynamodbav:"id"`
	Name         string   `dynamodbav:"name"`
	Document     string   `dynamodbav:"document"`
	DocumentType string   `dynamodbav:"document_type"`
	Email        string   `dynamodbav:"email"`
	PhoneNumber  string   `dynamodbav:"phone_number"`
	Password     string   `dynamodbav:"password"`
	Roles        []string `dynamodbav:"roles"`
}

// userRepository implements application.UserReader backed by DynamoDB.
type userRepository struct {
	client         QueryClient
	tableName      string
	emailIndexName    string
	documentIndexName string
}

// NewUserRepository constructs a UserReader backed by the given *dynamodb.Client.
// tableName is the DynamoDB table name; documentIndexName is the GSI name for document lookups.
func NewUserRepository(client *dynamodb.Client, tableName, documentIndexName string) application.UserReader {
	return NewUserRepositoryWithClient(client, tableName, documentIndexName)
}

// NewUserRepositoryWithClient constructs a UserReader with any QueryClient implementation.
// Useful for unit testing with a fake/mock client.
func NewUserRepositoryWithClient(client QueryClient, tableName, documentIndexName string) application.UserReader {
	return &userRepository{
		client:            client,
		tableName:         tableName,
		documentIndexName: documentIndexName,
	}
}

// GetByDocument queries the document GSI to find a user by their document address.
// Returns nil, nil when there is no user with that document (not found).
func (r *userRepository) GetByDocument(ctx context.Context, doc string) (*application.UserView, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String(r.documentIndexName),
		KeyConditionExpression: aws.String("document = :doc"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":doc": &types.AttributeValueMemberS{Value: doc},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("dynamodb query by document: %w", err)
	}

	if len(out.Items) == 0 {
		return nil, nil
	}

	var item userItem
	if err := attributevalue.UnmarshalMap(out.Items[0], &item); err != nil {
		return nil, fmt.Errorf("unmarshal user item: %w", err)
	}

	return toUserView(item)
}

// toUserView converts a raw DynamoDB item into the application UserView.
func toUserView(item userItem) (*application.UserView, error) {
	id, err := uuid.Parse(item.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid uuid %q", ErrInvalidUserItem, item.ID)
	}

	pwd, err := password.FromHash(item.Password)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid password hash", ErrInvalidUserItem)
	}

	return &application.UserView{
		ID:       id,
		Name:     item.Name,
		Email:    item.Email,
		Password: pwd,
		Roles:    authz.NewRolesUnchecked(item.Roles),
	}, nil
}