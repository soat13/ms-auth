package dynamodb

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soat13/ms-auth/internal/shared/authz"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"

	"github.com/soat13/ms-auth/internal/user/domain"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// toModel
// ---------------------------------------------------------------------------

func TestToModel(t *testing.T) {
	user := newValidUser(t)

	model := toModel(user)

	if model.ID != user.ID.String() {
		t.Errorf("expected ID %q, got %q", user.ID.String(), model.ID)
	}
	if model.Name != user.Name {
		t.Errorf("expected Name %q, got %q", user.Name, model.Name)
	}
	if model.Document != user.Document.Value {
		t.Errorf("expected Document %q, got %q", user.Document.Value, model.Document)
	}
	if model.DocumentType != user.Document.Type() {
		t.Errorf("expected DocumentType %q, got %q", user.Document.Type(), model.DocumentType)
	}
	if model.Email != user.Email.String() {
		t.Errorf("expected Email %q, got %q", user.Email.String(), model.Email)
	}
	if model.PhoneNumber != user.PhoneNumber.String() {
		t.Errorf("expected PhoneNumber %q, got %q", user.PhoneNumber.String(), model.PhoneNumber)
	}
	if model.Password != user.Password.Hash {
		t.Errorf("expected Password hash %q, got %q", user.Password.Hash, model.Password)
	}
	if len(model.Roles) != len(user.Roles) {
		t.Errorf("expected %d roles, got %d", len(user.Roles), len(model.Roles))
	}
	if model.CreatedAt == "" {
		t.Error("expected CreatedAt to be set")
	}
	if model.UpdatedAt == "" {
		t.Error("expected UpdatedAt to be set")
	}
}

// ---------------------------------------------------------------------------
// toDomain
// ---------------------------------------------------------------------------

func TestToDomain(t *testing.T) {
	t.Run("Valid Model", func(t *testing.T) {
		now := time.Now()
		id := uuid.New()

		pwd, _ := password.New("Password123!")

		model := &userModel{
			ID:           id.String(),
			Name:         "Test User",
			Document:     "12345678909",
			DocumentType: "CPF",
			Email:        "test@example.com",
			PhoneNumber:  "11999999999",
			Password:     pwd.Hash,
			Roles:        []string{"attendant"},
			CreatedAt:    now.Format(time.RFC3339Nano),
			UpdatedAt:    now.Format(time.RFC3339Nano),
		}

		user := toDomain(model)
		if user == nil {
			t.Fatal("expected user, got nil")
		}
		if user.ID != id {
			t.Errorf("expected ID %v, got %v", id, user.ID)
		}
		if user.Name != model.Name {
			t.Errorf("expected Name %q, got %q", model.Name, user.Name)
		}
		if user.Email.String() != model.Email {
			t.Errorf("expected Email %q, got %q", model.Email, user.Email.String())
		}
		if len(user.Roles) != 1 {
			t.Errorf("expected 1 role, got %d", len(user.Roles))
		}
	})

	t.Run("Nil Model", func(t *testing.T) {
		user := toDomain(nil)
		if user != nil {
			t.Errorf("expected nil, got %v", user)
		}
	})

	t.Run("Empty ID", func(t *testing.T) {
		model := &userModel{ID: ""}
		user := toDomain(model)
		if user != nil {
			t.Errorf("expected nil for empty ID, got %v", user)
		}
	})

	t.Run("Malformed UUID Produces Valid User", func(t *testing.T) {
		pwd, _ := password.New("Password123!")
		model := &userModel{
			ID:          "not-a-uuid",
			Name:        "Test",
			Document:    "12345678909",
			Email:       "test@example.com",
			PhoneNumber: "11999999999",
			Password:    pwd.Hash,
			Roles:       []string{"attendant"},
			CreatedAt:   time.Now().Format(time.RFC3339Nano),
			UpdatedAt:   time.Now().Format(time.RFC3339Nano),
		}

		// toDomain does not panic and returns a user (uuid.Parse may fall back to
		// uuid.Nil or derive a hash-based UUID — both are acceptable).
		user := toDomain(model)
		if user == nil {
			t.Fatal("expected user, got nil")
		}
	})

	t.Run("Invalid Name Returns Nil", func(t *testing.T) {
		pwd, _ := password.New("Password123!")
		model := &userModel{
			ID:          uuid.New().String(),
			Name:        "", // domain.NewUser rejects empty name
			Document:    "12345678909",
			Email:       "test@example.com",
			PhoneNumber: "11999999999",
			Password:    pwd.Hash,
			Roles:       []string{"attendant"},
			CreatedAt:   time.Now().Format(time.RFC3339Nano),
			UpdatedAt:   time.Now().Format(time.RFC3339Nano),
		}

		user := toDomain(model)
		if user != nil {
			t.Errorf("expected nil for invalid name, got %v", user)
		}
	})
}

// ---------------------------------------------------------------------------
// Round-trip: toModel -> toDomain
// ---------------------------------------------------------------------------

func TestRoundTrip(t *testing.T) {
	original := newValidUser(t)

	model := toModel(original)
	restored := toDomain(model)

	if restored == nil {
		t.Fatal("round-trip produced nil user")
	}
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: %v != %v", original.ID, restored.ID)
	}
	if restored.Name != original.Name {
		t.Errorf("Name mismatch: %q != %q", original.Name, restored.Name)
	}
	if restored.Email.String() != original.Email.String() {
		t.Errorf("Email mismatch: %q != %q", original.Email.String(), restored.Email.String())
	}
	if restored.Document.Value != original.Document.Value {
		t.Errorf("Document mismatch: %q != %q", original.Document.Value, restored.Document.Value)
	}
	if len(restored.Roles) != len(original.Roles) {
		t.Errorf("Roles mismatch: %d != %d", len(original.Roles), len(restored.Roles))
	}
}
