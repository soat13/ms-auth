package user

import (
	"github.com/soat13/oficina-auth/internal/bootstrap"
	userApp "github.com/soat13/oficina-auth/internal/user/application"
	userHTTP "github.com/soat13/oficina-auth/internal/user/infra/in/http"
	userDDB "github.com/soat13/oficina-auth/internal/user/infra/out/dynamodb"
)

func SetupDefault(container *bootstrap.Container) {
	Setup(container, nil)
}

func Setup(container *bootstrap.Container, repository userApp.Repository) {
	if repository == nil {
		repository = userDDB.NewUserRepository(
			container.DDBClient,
			container.Cfg.TableName,
			container.Cfg.EmailIndex,
			container.Cfg.DocumentIndex,
		)
	}

	create := userApp.NewCreateUser(repository)
	update := userApp.NewUpdateUser(repository)
	del := userApp.NewDeleteUser(repository)
	get := userApp.NewGetUser(repository)
	list := userApp.NewListUsers(repository)

	userHandler := userHTTP.NewHandler(
		create,
		update,
		del,
		get,
		list,
		container.Validator,
		container.FiberErrorHandler,
	)

	userHTTP.Register(container.FiberApp, userHandler)
}
