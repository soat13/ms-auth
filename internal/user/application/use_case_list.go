package application

import (
	"context"

	"github.com/soat13/oficina-utils/pkg/maps"
	"github.com/soat13/oficina-utils/pkg/pagination"
)

type (
	ListInput struct {
		Pager pagination.Pagination
	}

	ListOutput struct {
		Users []UserView
	}

	ListUsers struct {
		repo Repository
	}
)

func NewListUsers(repo Repository) *ListUsers {
	return &ListUsers{repo: repo}
}

func (uc *ListUsers) Execute(ctx context.Context, in ListInput) (*ListOutput, error) {
	items, err := uc.repo.List(ctx, in.Pager)
	if err != nil {
		return nil, err
	}

	users := maps.Map(items, toView)
	return &ListOutput{Users: users}, nil
}
