package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soat13/ms-auth/internal/shared/authz"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"
)

func mustNewRoles(values []string) authz.Roles {
	roles, _ := authz.NewRoles(values)
	return roles
}

func TestNewUser(t *testing.T) {
	now := time.Now()
	validName := "John Doe"
	validDocument, _ := document.New("12345678900")
	validPhone, _ := phone.New("11987654321")
	validEmail, _ := email.New("test@example.com")
	validPassword, _ := password.New("password123")
	validRoles, _ := authz.NewRoles([]string{"attendant"})

	tests := []struct {
		name        string
		id          uuid.UUID
		userName    string
		document    document.Document
		phoneNumber phone.PhoneNumber
		email       email.Email
		password    password.Password
		roles       authz.Roles
		now         time.Time
		wantErr     error
	}{
		{
			name:        "valid user",
			id:          uuid.Nil,
			userName:    validName,
			document:    validDocument,
			phoneNumber: validPhone,
			email:       validEmail,
			password:    validPassword,
			roles:       validRoles,
			now:         now,
			wantErr:     nil,
		},
		{
			name:        "valid user with specific ID",
			id:          uuid.New(),
			userName:    validName,
			document:    validDocument,
			phoneNumber: validPhone,
			email:       validEmail,
			password:    validPassword,
			roles:       validRoles,
			now:         now,
			wantErr:     nil,
		},
		{
			name:        "empty name",
			id:          uuid.Nil,
			userName:    "",
			document:    validDocument,
			phoneNumber: validPhone,
			email:       validEmail,
			password:    validPassword,
			roles:       validRoles,
			now:         now,
			wantErr:     ErrInvalidUserName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUser(tt.id, tt.userName, tt.document, tt.phoneNumber, tt.email, tt.password, tt.roles)
			if err != tt.wantErr {
				t.Errorf("NewUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == nil {
				if got.ID == uuid.Nil {
					t.Error("NewUser() ID should not be Nil")
				}
				if got.Name != tt.userName {
					t.Errorf("NewUser() Name = %v, want %v", got.Name, tt.userName)
				}
				if got.Email != tt.email {
					t.Errorf("NewUser() Email = %v, want %v", got.Email, tt.email)
				}
				if len(got.Roles) != len(tt.roles) {
					t.Errorf("NewUser() Roles = %v, want %v", got.Roles, tt.roles)
				}
			}
		})
	}
}

func TestUserChangeName(t *testing.T) {
	doc, _ := document.New("12345678900")
	phone, _ := phone.New("11987654321")
	em, _ := email.New("test@example.com")
	pwd, _ := password.New("password123")
	roles, _ := authz.NewRoles([]string{"manager"})

	user, _ := NewUser(uuid.Nil, "Old Name", doc, phone, em, pwd, roles)
	err := user.ChangeName("New Name")

	if err != nil {
		t.Errorf("ChangeName() error = %v, want nil", err)
	}
	if user.Name != "New Name" {
		t.Errorf("ChangeName() Name = %v, want %v", user.Name, "New Name")
	}

	err = user.ChangeName("")
	if err != ErrInvalidUserName {
		t.Errorf("ChangeName() error = %v, want %v", err, ErrInvalidUserName)
	}
}

func TestUserChangePhoneNumber(t *testing.T) {
	doc, _ := document.New("12345678900")
	oldPhone, _ := phone.New("11987654321")
	newPhone, _ := phone.New("11987654322")
	em, _ := email.New("test@example.com")
	pwd, _ := password.New("password123")
	roles, _ := authz.NewRoles([]string{"mechanic"})

	user, _ := NewUser(uuid.Nil, "Test User", doc, oldPhone, em, pwd, roles)
	user.ChangePhoneNumber(newPhone)

	if user.PhoneNumber != newPhone {
		t.Errorf("ChangePhoneNumber() PhoneNumber = %v, want %v", user.PhoneNumber, newPhone)
	}
}

func TestUserChangeEmail(t *testing.T) {
	doc, _ := document.New("12345678900")
	phone, _ := phone.New("11987654321")
	oldEmail, _ := email.New("old@example.com")
	newEmail, _ := email.New("new@example.com")
	pwd, _ := password.New("password123")
	roles, _ := authz.NewRoles([]string{"attendant"})

	user, _ := NewUser(uuid.Nil, "Test User", doc, phone, oldEmail, pwd, roles)
	user.ChangeEmail(newEmail)

	if user.Email != newEmail {
		t.Errorf("ChangeEmail() Email = %v, want %v", user.Email, newEmail)
	}
}

func TestUserChangePassword(t *testing.T) {
	doc, _ := document.New("12345678900")
	phone, _ := phone.New("11987654321")
	em, _ := email.New("test@example.com")
	oldPwd, _ := password.New("oldpassword")
	newPwd, _ := password.New("newpassword")
	roles, _ := authz.NewRoles([]string{"mechanic"})

	user, _ := NewUser(uuid.Nil, "Test User", doc, phone, em, oldPwd, roles)
	user.ChangePassword(newPwd)

	if !user.Password.Matches("newpassword") {
		t.Error("ChangePassword() failed to update password")
	}
}

func TestUserChangeRoles(t *testing.T) {
	doc, _ := document.New("12345678900")
	phone, _ := phone.New("11987654321")
	em, _ := email.New("test@example.com")
	pwd, _ := password.New("password123")
	initialRoles, _ := authz.NewRoles([]string{"attendant"})

	user, _ := NewUser(uuid.Nil, "Test User", doc, phone, em, pwd, initialRoles)

	tests := []struct {
		name     string
		newRoles authz.Roles
		wantErr  error
	}{
		{
			name:     "valid single role change",
			newRoles: mustNewRoles([]string{"manager"}),
			wantErr:  nil,
		},
		{
			name:     "valid multiple roles change",
			newRoles: mustNewRoles([]string{"manager", "mechanic"}),
			wantErr:  nil,
		},
		{
			name:     "empty roles",
			newRoles: authz.Roles{},
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := user.ChangeRoles(tt.newRoles)
			if err != tt.wantErr {
				t.Errorf("ChangeRoles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == nil {
				if len(user.Roles) != len(tt.newRoles) {
					t.Errorf("ChangeRoles() Roles length = %v, want %v", len(user.Roles), len(tt.newRoles))
				}
			}
		})
	}
}
