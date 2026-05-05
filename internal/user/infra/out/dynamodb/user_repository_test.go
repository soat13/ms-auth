package dynamodb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/soat13/ms-auth/internal/shared/authz"
	"github.com/soat13/ms-auth/internal/user/domain"
	"github.com/soat13/oficina-utils/pkg/pagination"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"
)

// ---------------------------------------------------------------------------
// Mock
// ---------------------------------------------------------------------------

type mockDDB struct {
	putItem    func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	updateItem func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	deleteItem func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	getItem    func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	query      func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	scan       func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

func (m *mockDDB) PutItem(ctx context.Context, p *dynamodb.PutItemInput, o ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m.putItem(ctx, p, o...)
}
func (m *mockDDB) UpdateItem(ctx context.Context, p *dynamodb.UpdateItemInput, o ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return m.updateItem(ctx, p, o...)
}
func (m *mockDDB) DeleteItem(ctx context.Context, p *dynamodb.DeleteItemInput, o ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return m.deleteItem(ctx, p, o...)
}
func (m *mockDDB) GetItem(ctx context.Context, p *dynamodb.GetItemInput, o ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.getItem(ctx, p, o...)
}
func (m *mockDDB) Query(ctx context.Context, p *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m.query(ctx, p, o...)
}
func (m *mockDDB) Scan(ctx context.Context, p *dynamodb.ScanInput, o ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	return m.scan(ctx, p, o...)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var errFake = errors.New("fake dynamo error")

func newValidUser(t *testing.T) *domain.User {
	t.Helper()
	doc, _ := document.New("12345678909")
	ph, _ := phone.New("11999999999")
	em, _ := email.New("test@example.com")
	pwd, _ := password.New("Password123!")
	roles, _ := authz.NewRoles([]string{"attendant"})

	user, err := domain.NewUser(uuid.Nil, "Test User", doc, ph, em, pwd, roles)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func newRepo(mock *mockDDB) *UserRepository {
	return &UserRepository{
		client:        mock,
		tableName:     "users",
		emailIndex:    "email-index",
		documentIndex: "document-index",
	}
}

// marshalUser returns a DynamoDB attribute map for a valid user model.
func marshalUser(t *testing.T) map[string]types.AttributeValue {
	t.Helper()
	pwd, _ := password.New("Password123!")
	model := &userModel{
		ID:          uuid.New().String(),
		Name:        "Test User",
		Document:    "12345678909",
		DocumentType: "CPF",
		Email:       "test@example.com",
		PhoneNumber: "11999999999",
		Password:    pwd.Hash,
		Roles:       []string{"attendant"},
		CreatedAt:   time.Now().Format(time.RFC3339Nano),
		UpdatedAt:   time.Now().Format(time.RFC3339Nano),
	}
	item, err := attributevalue.MarshalMap(model)
	if err != nil {
		t.Fatalf("failed to marshal user model: %v", err)
	}
	return item
}

// ═══════════════════════════════════════════════════════════════════════════
// toModel / toDomain (pure logic)
// ═══════════════════════════════════════════════════════════════════════════

func TestToModel(t *testing.T) {
	user := newValidUser(t)
	model := toModel(user)

	if model.ID != user.ID.String() {
		t.Errorf("ID: got %q, want %q", model.ID, user.ID.String())
	}
	if model.Name != user.Name {
		t.Errorf("Name: got %q, want %q", model.Name, user.Name)
	}
	if model.Document != user.Document.Value {
		t.Errorf("Document: got %q, want %q", model.Document, user.Document.Value)
	}
	if model.DocumentType != user.Document.Type() {
		t.Errorf("DocumentType: got %q, want %q", model.DocumentType, user.Document.Type())
	}
	if model.Email != user.Email.String() {
		t.Errorf("Email: got %q, want %q", model.Email, user.Email.String())
	}
	if model.PhoneNumber != user.PhoneNumber.String() {
		t.Errorf("PhoneNumber: got %q, want %q", model.PhoneNumber, user.PhoneNumber.String())
	}
	if model.Password != user.Password.Hash {
		t.Errorf("Password: got %q, want %q", model.Password, user.Password.Hash)
	}
	if len(model.Roles) != len(user.Roles) {
		t.Errorf("Roles len: got %d, want %d", len(model.Roles), len(user.Roles))
	}
	if model.CreatedAt == "" || model.UpdatedAt == "" {
		t.Error("timestamps should not be empty")
	}
}

func TestToDomain(t *testing.T) {
	t.Run("Valid Model", func(t *testing.T) {
		now := time.Now()
		id := uuid.New()
		pwd, _ := password.New("Password123!")

		model := &userModel{
			ID:          id.String(),
			Name:        "Test User",
			Document:    "12345678909",
			DocumentType: "CPF",
			Email:       "test@example.com",
			PhoneNumber: "11999999999",
			Password:    pwd.Hash,
			Roles:       []string{"attendant"},
			CreatedAt:   now.Format(time.RFC3339Nano),
			UpdatedAt:   now.Format(time.RFC3339Nano),
		}

		user := toDomain(model)
		if user == nil {
			t.Fatal("expected user, got nil")
		}
		if user.ID != id {
			t.Errorf("ID: got %v, want %v", user.ID, id)
		}
		if user.Name != model.Name {
			t.Errorf("Name: got %q, want %q", user.Name, model.Name)
		}
	})

	t.Run("Nil Model", func(t *testing.T) {
		if toDomain(nil) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("Empty ID", func(t *testing.T) {
		if toDomain(&userModel{ID: ""}) != nil {
			t.Error("expected nil for empty ID")
		}
	})

	t.Run("Invalid Name Returns Nil", func(t *testing.T) {
		pwd, _ := password.New("Password123!")
		model := &userModel{
			ID: uuid.New().String(), Name: "",
			Document: "12345678909", Email: "test@example.com",
			PhoneNumber: "11999999999", Password: pwd.Hash,
			Roles: []string{"attendant"},
			CreatedAt: time.Now().Format(time.RFC3339Nano),
			UpdatedAt: time.Now().Format(time.RFC3339Nano),
		}
		if toDomain(model) != nil {
			t.Error("expected nil for empty name")
		}
	})
}

func TestRoundTrip(t *testing.T) {
	original := newValidUser(t)
	restored := toDomain(toModel(original))

	if restored == nil {
		t.Fatal("round-trip produced nil")
	}
	if restored.ID != original.ID {
		t.Errorf("ID mismatch")
	}
	if restored.Name != original.Name {
		t.Errorf("Name mismatch")
	}
	if restored.Email.String() != original.Email.String() {
		t.Errorf("Email mismatch")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Create – error paths
// ═══════════════════════════════════════════════════════════════════════════

func TestCreate_PutItemError(t *testing.T) {
	mock := &mockDDB{
		putItem: func(ctx context.Context, p *dynamodb.PutItemInput, o ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, errFake
		},
	}
	repo := newRepo(mock)

	err := repo.Create(context.Background(), newValidUser(t))
	if !errors.Is(err, errFake) {
		t.Errorf("expected errFake, got %v", err)
	}
}

func TestCreate_ConditionalCheckFailed(t *testing.T) {
	mock := &mockDDB{
		putItem: func(ctx context.Context, p *dynamodb.PutItemInput, o ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{Message: strPtr("already exists")}
		},
	}
	repo := newRepo(mock)

	err := repo.Create(context.Background(), newValidUser(t))
	if !errors.Is(err, domain.ErrUserAlreadyExists) {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestCreate_Success(t *testing.T) {
	mock := &mockDDB{
		putItem: func(ctx context.Context, p *dynamodb.PutItemInput, o ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	repo := newRepo(mock)

	if err := repo.Create(context.Background(), newValidUser(t)); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Update – error paths
// ═══════════════════════════════════════════════════════════════════════════

func TestUpdate_UpdateItemError(t *testing.T) {
	mock := &mockDDB{
		updateItem: func(ctx context.Context, p *dynamodb.UpdateItemInput, o ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, errFake
		},
	}
	repo := newRepo(mock)

	err := repo.Update(context.Background(), newValidUser(t))
	if !errors.Is(err, errFake) {
		t.Errorf("expected errFake, got %v", err)
	}
}

func TestUpdate_Success(t *testing.T) {
	mock := &mockDDB{
		updateItem: func(ctx context.Context, p *dynamodb.UpdateItemInput, o ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}
	repo := newRepo(mock)

	if err := repo.Update(context.Background(), newValidUser(t)); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Delete – error paths
// ═══════════════════════════════════════════════════════════════════════════

func TestDelete_DeleteItemError(t *testing.T) {
	mock := &mockDDB{
		deleteItem: func(ctx context.Context, p *dynamodb.DeleteItemInput, o ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return nil, errFake
		},
	}
	repo := newRepo(mock)

	if err := repo.Delete(context.Background(), uuid.New()); !errors.Is(err, errFake) {
		t.Errorf("expected errFake, got %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	mock := &mockDDB{
		deleteItem: func(ctx context.Context, p *dynamodb.DeleteItemInput, o ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return &dynamodb.DeleteItemOutput{}, nil
		},
	}
	repo := newRepo(mock)

	if err := repo.Delete(context.Background(), uuid.New()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GetByID – error paths
// ═══════════════════════════════════════════════════════════════════════════

func TestGetByID_GetItemError(t *testing.T) {
	mock := &mockDDB{
		getItem: func(ctx context.Context, p *dynamodb.GetItemInput, o ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, errFake
		},
	}
	repo := newRepo(mock)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, errFake) {
		t.Errorf("expected errFake, got %v", err)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	mock := &mockDDB{
		getItem: func(ctx context.Context, p *dynamodb.GetItemInput, o ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}
	repo := newRepo(mock)

	user, err := repo.GetByID(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
}

func TestGetByID_UnmarshalError(t *testing.T) {
	// roles is []string in the model — a BOOL value cannot be deserialized into it.
	mock := &mockDDB{
		getItem: func(ctx context.Context, p *dynamodb.GetItemInput, o ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"id":    &types.AttributeValueMemberS{Value: uuid.New().String()},
					"name":  &types.AttributeValueMemberS{Value: "Test"},
					"roles": &types.AttributeValueMemberBOOL{Value: true},
				},
			}, nil
		},
	}
	repo := newRepo(mock)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected unmarshal error, got nil")
	}
}

func TestGetByID_Success(t *testing.T) {
	item := marshalUser(t)
	mock := &mockDDB{
		getItem: func(ctx context.Context, p *dynamodb.GetItemInput, o ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: item}, nil
		},
	}
	repo := newRepo(mock)

	user, err := repo.GetByID(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if user == nil {
		t.Error("expected user, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GetByEmail – error paths
// ═══════════════════════════════════════════════════════════════════════════

func TestGetByEmail_QueryError(t *testing.T) {
	mock := &mockDDB{
		query: func(ctx context.Context, p *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errFake
		},
	}
	repo := newRepo(mock)

	_, err := repo.GetByEmail(context.Background(), "test@example.com")
	if !errors.Is(err, errFake) {
		t.Errorf("expected errFake, got %v", err)
	}
}

func TestGetByEmail_NotFound(t *testing.T) {
	mock := &mockDDB{
		query: func(ctx context.Context, p *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: nil}, nil
		},
	}
	repo := newRepo(mock)

	user, err := repo.GetByEmail(context.Background(), "nobody@example.com")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
}

func TestGetByEmail_UnmarshalError(t *testing.T) {
	mock := &mockDDB{
		query: func(ctx context.Context, p *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"id":    &types.AttributeValueMemberS{Value: uuid.New().String()},
						"name":  &types.AttributeValueMemberS{Value: "Test"},
						"roles": &types.AttributeValueMemberBOOL{Value: true},
					},
				},
			}, nil
		},
	}
	repo := newRepo(mock)

	_, err := repo.GetByEmail(context.Background(), "test@example.com")
	if err == nil {
		t.Error("expected unmarshal error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// List – error paths
// ═══════════════════════════════════════════════════════════════════════════

func TestList_ScanError(t *testing.T) {
	mock := &mockDDB{
		scan: func(ctx context.Context, p *dynamodb.ScanInput, o ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
			return nil, errFake
		},
	}
	repo := newRepo(mock)

	_, err := repo.List(context.Background(), pagination.Pagination{Limit: 10})
	if !errors.Is(err, errFake) {
		t.Errorf("expected errFake, got %v", err)
	}
}

func TestList_UnmarshalError(t *testing.T) {
	mock := &mockDDB{
		scan: func(ctx context.Context, p *dynamodb.ScanInput, o ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
			return &dynamodb.ScanOutput{
				Items: []map[string]types.AttributeValue{
					{
						"id":    &types.AttributeValueMemberS{Value: uuid.New().String()},
						"name":  &types.AttributeValueMemberS{Value: "Test"},
						"roles": &types.AttributeValueMemberBOOL{Value: true},
					},
				},
			}, nil
		},
	}
	repo := newRepo(mock)

	_, err := repo.List(context.Background(), pagination.Pagination{Limit: 10})
	if err == nil {
		t.Error("expected unmarshal error, got nil")
	}
}

func TestList_DefaultLimit(t *testing.T) {
	mock := &mockDDB{
		scan: func(ctx context.Context, p *dynamodb.ScanInput, o ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
			if *p.Limit != 50 {
				t.Errorf("expected default limit 50, got %d", *p.Limit)
			}
			return &dynamodb.ScanOutput{Items: nil}, nil
		},
	}
	repo := newRepo(mock)

	_, _ = repo.List(context.Background(), pagination.Pagination{Limit: 0})
}

func TestList_Success(t *testing.T) {
	item := marshalUser(t)
	mock := &mockDDB{
		scan: func(ctx context.Context, p *dynamodb.ScanInput, o ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
			return &dynamodb.ScanOutput{Items: []map[string]types.AttributeValue{item}}, nil
		},
	}
	repo := newRepo(mock)

	users, err := repo.List(context.Background(), pagination.Pagination{Limit: 10})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ExistsByEmail – error path
// ═══════════════════════════════════════════════════════════════════════════

func TestExistsByEmail_Error(t *testing.T) {
	mock := &mockDDB{
		query: func(ctx context.Context, p *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errFake
		},
	}
	repo := newRepo(mock)

	_, err := repo.ExistsByEmail(context.Background(), "test@example.com")
	if !errors.Is(err, errFake) {
		t.Errorf("expected errFake, got %v", err)
	}
}

func TestExistsByEmail_True(t *testing.T) {
	item := marshalUser(t)
	mock := &mockDDB{
		query: func(ctx context.Context, p *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{item}}, nil
		},
	}
	repo := newRepo(mock)

	exists, err := repo.ExistsByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ExistsByDocument – error path
// ═══════════════════════════════════════════════════════════════════════════

func TestExistsByDocument_Error(t *testing.T) {
	mock := &mockDDB{
		query: func(ctx context.Context, p *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, errFake
		},
	}
	repo := newRepo(mock)

	_, err := repo.ExistsByDocument(context.Background(), "12345678909")
	if !errors.Is(err, errFake) {
		t.Errorf("expected errFake, got %v", err)
	}
}

func TestExistsByDocument_Found(t *testing.T) {
	item := marshalUser(t)
	mock := &mockDDB{
		query: func(ctx context.Context, p *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{item}}, nil
		},
	}
	repo := newRepo(mock)

	exists, err := repo.ExistsByDocument(context.Background(), "12345678909")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true")
	}
}

func TestExistsByDocument_NotFound(t *testing.T) {
	mock := &mockDDB{
		query: func(ctx context.Context, p *dynamodb.QueryInput, o ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: nil}, nil
		},
	}
	repo := newRepo(mock)

	exists, err := repo.ExistsByDocument(context.Background(), "99999999999")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected false")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// helpers
// ═══════════════════════════════════════════════════════════════════════════

func strPtr(s string) *string { return &s }
