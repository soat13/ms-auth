package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/soat13/ms-auth/internal/auth/application"
	"github.com/soat13/ms-auth/internal/auth/application/mocks"
	authDomain "github.com/soat13/ms-auth/internal/auth/domain"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
)

func TestAuthenticateUserExecuteFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("should propagates error when UserReader.GetByDocument fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userReader := mocks.NewMockUserReader(ctrl)
		tokenService := mocks.NewMockTokenService(ctrl)

		usersErr := errors.New("db error")

		userReader.EXPECT().
			GetByDocument(gomock.Any(), gomock.Any()).
			Return(nil, usersErr)

		tokenService.EXPECT().
			Generate(gomock.Any(), gomock.Any()).
			Times(0)

		uc := application.NewAuthenticateUser(userReader, tokenService)

		_, err := uc.Execute(ctx, application.AuthenticateInput{
			Document: setupDoc("12345678909"),
			Password: "anything",
		})

		if !errors.Is(err, usersErr) {
			t.Fatalf("expected %v, got %v", usersErr, err)
		}
	})

	t.Run("should returns ErrInvalidCredentials when user is nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userReader := mocks.NewMockUserReader(ctrl)
		tokenService := mocks.NewMockTokenService(ctrl)

		userReader.EXPECT().
			GetByDocument(gomock.Any(), gomock.Any()).
			Return(nil, nil)

		tokenService.EXPECT().
			Generate(gomock.Any(), gomock.Any()).
			Times(0)

		uc := application.NewAuthenticateUser(userReader, tokenService)

		_, err := uc.Execute(ctx, application.AuthenticateInput{
			Document: setupDoc("12345678909"),
			Password: "anything",
		})

		if !errors.Is(err, application.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("should returns ErrInvalidCredentials when password does not match", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userReader := mocks.NewMockUserReader(ctrl)
		tokenService := mocks.NewMockTokenService(ctrl)

		p, _ := password.New("correct-password")

		foundUser := &application.UserView{
			Password: p,
		}

		userReader.EXPECT().
			GetByDocument(gomock.Any(), gomock.Any()).
			Return(foundUser, nil)

		tokenService.EXPECT().
			Generate(gomock.Any(), gomock.Any()).
			Times(0)

		uc := application.NewAuthenticateUser(userReader, tokenService)

		_, execErr := uc.Execute(ctx, application.AuthenticateInput{
			Document: setupDoc("12345678909"),
			Password: "wrong-password",
		})

		if !errors.Is(execErr, application.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", execErr)
		}
	})

	t.Run("should propagates error when TokenService.Generate fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userReader := mocks.NewMockUserReader(ctrl)
		tokenService := mocks.NewMockTokenService(ctrl)

		genErr := errors.New("token service error")

		p, _ := password.New("valid-password")

		user := &application.UserView{
			Password: p,
		}

		gomock.InOrder(
			userReader.EXPECT().
				GetByDocument(gomock.Any(), gomock.Any()).
				Return(user, nil),

			tokenService.EXPECT().
				Generate(gomock.Any(), user).
				Return(authDomain.Token{}, genErr),
		)

		uc := application.NewAuthenticateUser(userReader, tokenService)

		_, execErr := uc.Execute(ctx, application.AuthenticateInput{
			Document: setupDoc("12345678909"),
			Password: "valid-password",
		})

		if !errors.Is(execErr, genErr) {
			t.Fatalf("expected %v, got %v", genErr, execErr)
		}
	})
}

func setupDoc(s string) document.Document {
	d, _ := document.New(s)
	return d
}
