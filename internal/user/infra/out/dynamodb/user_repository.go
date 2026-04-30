package dynamodb

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/soat13/ms-auth/internal/shared/authz"
	"github.com/soat13/ms-auth/internal/user/application"
	"github.com/soat13/ms-auth/internal/user/domain"
	"github.com/soat13/oficina-utils/pkg/pagination"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"
)

type userModel struct {
	ID           string   `dynamodbav:"id"`
	Name         string   `dynamodbav:"name"`
	Document     string   `dynamodbav:"document"`
	DocumentType string   `dynamodbav:"document_type"`
	Email        string   `dynamodbav:"email"`
	PhoneNumber  string   `dynamodbav:"phone_number"`
	Password     string   `dynamodbav:"password"`
	Roles        []string `dynamodbav:"roles"`
	CreatedAt    string   `dynamodbav:"created_at"`
	UpdatedAt    string   `dynamodbav:"updated_at"`
}

type UserRepository struct {
	client        *dynamodb.Client
	tableName     string
	emailIndex    string
	documentIndex string
}

func NewUserRepository(client *dynamodb.Client, tableName, emailIndex, documentIndex string) application.Repository {
	return &UserRepository{
		client:        client,
		tableName:     tableName,
		emailIndex:    emailIndex,
		documentIndex: documentIndex,
	}
}

func (repo *UserRepository) Create(ctx context.Context, u *domain.User) error {
	model := toModel(u)
	item, err := attributevalue.MarshalMap(model)
	if err != nil {
		return err
	}

	_, err = repo.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(repo.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})

	var ccf *types.ConditionalCheckFailedException
	if errors.As(err, &ccf) {
		return domain.ErrUserAlreadyExists
	}
	return err
}

func (repo *UserRepository) Update(ctx context.Context, u *domain.User) error {
	model := toModel(u)

	update := expression.Set(expression.Name("name"), expression.Value(model.Name)).
		Set(expression.Name("document"), expression.Value(model.Document)).
		Set(expression.Name("document_type"), expression.Value(model.DocumentType)).
		Set(expression.Name("email"), expression.Value(model.Email)).
		Set(expression.Name("phone_number"), expression.Value(model.PhoneNumber)).
		Set(expression.Name("password"), expression.Value(model.Password)).
		Set(expression.Name("roles"), expression.Value(model.Roles)).
		Set(expression.Name("updated_at"), expression.Value(model.UpdatedAt))

	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return err
	}

	key, err := attributevalue.MarshalMap(map[string]string{"id": model.ID})
	if err != nil {
		return err
	}

	_, err = repo.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(repo.tableName),
		Key:                       key,
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	return err
}

func (repo *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	key, err := attributevalue.MarshalMap(map[string]string{"id": id.String()})
	if err != nil {
		return err
	}

	_, err = repo.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(repo.tableName),
		Key:       key,
	})
	return err
}

func (repo *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	key, err := attributevalue.MarshalMap(map[string]string{"id": id.String()})
	if err != nil {
		return nil, err
	}

	out, err := repo.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(repo.tableName),
		Key:       key,
	})
	if err != nil {
		return nil, err
	}
	if len(out.Item) == 0 {
		return nil, nil
	}

	var model userModel
	if err := attributevalue.UnmarshalMap(out.Item, &model); err != nil {
		return nil, err
	}

	return toDomain(&model), nil
}

func (repo *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	out, err := repo.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(repo.tableName),
		IndexName:              aws.String(repo.emailIndex),
		KeyConditionExpression: aws.String("email = :e"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":e": &types.AttributeValueMemberS{Value: email},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, nil
	}

	var model userModel
	if err := attributevalue.UnmarshalMap(out.Items[0], &model); err != nil {
		return nil, err
	}

	return toDomain(&model), nil
}

func (repo *UserRepository) List(ctx context.Context, pager pagination.Pagination) ([]*domain.User, error) {
	limit := int32(pager.Limit)
	if limit <= 0 {
		limit = 50
	}

	out, err := repo.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(repo.tableName),
		Limit:     aws.Int32(limit),
	})
	if err != nil {
		return nil, err
	}

	var models []userModel
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &models); err != nil {
		return nil, err
	}

	var users []*domain.User
	for i := range models {
		if u := toDomain(&models[i]); u != nil {
			users = append(users, u)
		}
	}
	return users, nil
}

func (repo *UserRepository) ExistsByEmail(ctx context.Context, em string) (bool, error) {
	user, err := repo.GetByEmail(ctx, em)
	if err != nil {
		return false, err
	}
	return user != nil, nil
}

func (repo *UserRepository) ExistsByDocument(ctx context.Context, docStr string) (bool, error) {
	out, err := repo.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(repo.tableName),
		IndexName:              aws.String(repo.documentIndex),
		KeyConditionExpression: aws.String("document = :d"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":d": &types.AttributeValueMemberS{Value: docStr},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return false, err
	}
	return len(out.Items) > 0, nil
}

func toModel(u *domain.User) *userModel {
	return &userModel{
		ID:           u.ID.String(),
		Name:         u.Name,
		Document:     u.Document.Value,
		DocumentType: u.Document.Type(),
		Email:        u.Email.String(),
		PhoneNumber:  u.PhoneNumber.String(),
		Password:     u.Password.Hash,
		Roles:        u.Roles.Strings(),
		CreatedAt:    u.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:    u.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func toDomain(model *userModel) *domain.User {
	if model == nil || model.ID == "" {
		return nil
	}

	id, err := uuid.Parse(model.ID)
	if err != nil {
		id = uuid.Nil
	}

	doc, _ := document.New(model.Document)
	phoneNumber, _ := phone.New(model.PhoneNumber)
	em, _ := email.New(model.Email)
	pwd, _ := password.FromHash(model.Password)
	roles := authz.NewRolesUnchecked(model.Roles)

	user, err := domain.NewUser(id, model.Name, doc, phoneNumber, em, pwd, roles)
	if err != nil {
		return nil
	}

	parsedCreated, _ := time.Parse(time.RFC3339Nano, model.CreatedAt)
	parsedUpdated, _ := time.Parse(time.RFC3339Nano, model.UpdatedAt)
	user.CreatedAt = parsedCreated
	user.UpdatedAt = parsedUpdated

	return user
}
