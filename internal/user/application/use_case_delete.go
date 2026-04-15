package application

import (
	"context"

	"github.com/google/uuid"
)

type (
	DeleteInput struct {
		ID uuid.UUID
	}

	DeleteUser struct {
		repo Repository
	}
)

func NewDeleteUser(repo Repository) *DeleteUser {
	return &DeleteUser{repo: repo}
}

func (uc *DeleteUser) Execute(ctx context.Context, in DeleteInput) error {
	return uc.repo.Delete(ctx, in.ID)
}
