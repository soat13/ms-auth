package application

import (
	"context"

	"github.com/google/uuid"
)

type (
	GetInput struct {
		ID uuid.UUID
	}

	GetOutput struct {
		User UserView
	}

	GetUser struct {
		repo Repository
	}
)

func NewGetUser(repo Repository) *GetUser {
	return &GetUser{repo: repo}
}

func (uc *GetUser) Execute(ctx context.Context, in GetInput) (*GetOutput, error) {
	user, err := uc.repo.GetByID(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return &GetOutput{User: toView(user)}, nil
}
