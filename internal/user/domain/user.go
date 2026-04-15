package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/soat13/oficina-auth/internal/shared/authz"
	"github.com/soat13/oficina-auth/internal/shared/uuidutil"
	"github.com/soat13/oficina-utils/pkg/entity"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"
)

type User struct {
	ID          uuid.UUID
	Name        string
	Document    document.Document
	Email       email.Email
	PhoneNumber phone.PhoneNumber
	Password    password.Password
	Roles       authz.Roles
	entity.Timestamps
}

func NewUser(id uuid.UUID, name string, document document.Document, phoneNumber phone.PhoneNumber, email email.Email, password password.Password, roles authz.Roles) (*User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidUserName
	}

	now := time.Now()
	user := &User{
		ID:          uuidutil.IDOrNew(id),
		Name:        name,
		Document:    document,
		PhoneNumber: phoneNumber,
		Email:       email,
		Password:    password,
		Roles:       roles,
		Timestamps:  entity.NewTimestamps(now, now),
	}

	return user, nil
}

func (user *User) ChangeName(newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return ErrInvalidUserName
	}
	user.Name = newName
	user.Touch()
	return nil
}

func (user *User) ChangePhoneNumber(phoneNumber phone.PhoneNumber) {
	user.PhoneNumber = phoneNumber
	user.Touch()
}

func (user *User) ChangeEmail(email email.Email) {
	user.Email = email
	user.Touch()
}

func (user *User) ChangePassword(password password.Password) {
	user.Password = password
	user.Touch()
}

func (user *User) ChangeRoles(roles authz.Roles) error {
	user.Roles = roles
	user.Touch()
	return nil
}
