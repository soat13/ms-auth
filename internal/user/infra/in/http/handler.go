package http

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/soat13/ms-auth/internal/shared/authz"
	"github.com/soat13/ms-auth/internal/user/application"
	"github.com/soat13/ms-auth/internal/user/domain"
	fiberHelper "github.com/soat13/oficina-utils/pkg/http/fiber"
	"github.com/soat13/oficina-utils/pkg/maps"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"
)

var ErrInvalidJSON = errors.New("invalid JSON body")

type Handler struct {
	createUseCase *application.CreateUser
	updateUseCase *application.UpdateUser
	deleteUseCase *application.DeleteUser
	getUseCase    *application.GetUser
	listUseCase   *application.ListUsers
	validator     *validator.Validate
	errorHandler  *fiberHelper.ErrorHandler
}

func NewHandler(
	create *application.CreateUser,
	update *application.UpdateUser,
	delete *application.DeleteUser,
	get *application.GetUser,
	list *application.ListUsers,
	validator *validator.Validate,
	errorHandler *fiberHelper.ErrorHandler,
) *Handler {
	handler := &Handler{
		createUseCase: create,
		updateUseCase: update,
		deleteUseCase: delete,
		getUseCase:    get,
		listUseCase:   list,
		validator:     validator,
		errorHandler:  errorHandler,
	}

	handler.errorHandler.ErrorResolver.RegisterHTTPBadRequestError(ErrInvalidJSON)
	handler.errorHandler.ErrorResolver.RegisterHTTPNotFoundError(application.ErrUserNotFound)
	handler.errorHandler.ErrorResolver.RegisterHTTPConflictError(application.ErrDuplicateEmail)
	handler.errorHandler.ErrorResolver.RegisterHTTPConflictError(application.ErrDuplicateDocument)
	handler.errorHandler.ErrorResolver.RegisterHTTPUnprocessableError(domain.ErrInvalidUserName)
	handler.errorHandler.ErrorResolver.RegisterHTTPUnprocessableError(authz.ErrInvalidRole)
	handler.errorHandler.ErrorResolver.RegisterHTTPUnprocessableError(authz.ErrRolesRequired)

	return handler
}

func Register(app *fiber.App, h *Handler) {
	grp := app.Group("/admin/users")
	grp.Post("/", h.create)
	grp.Patch("/:id", h.update)
	grp.Delete("/:id", h.delete)
	grp.Get("/:id", h.getByID)
	grp.Get("/", h.list)
}

type createBody struct {
	Name        string   `json:"name"         validate:"required,min=3"`
	Document    string   `json:"document"     validate:"required,min=11"`
	Email       string   `json:"email"        validate:"required,email"`
	PhoneNumber string   `json:"phone_number" validate:"required,min=11"`
	Password    string   `json:"password"     validate:"required,min=8"`
	Roles       []string `json:"roles"        validate:"required,min=1,dive,required"`
}

type updateBody struct {
	Name        *string   `json:"name"         validate:"omitempty,min=3"`
	Email       *string   `json:"email"        validate:"omitempty,email"`
	PhoneNumber *string   `json:"phone_number" validate:"omitempty,min=11"`
	Password    *string   `json:"password"     validate:"omitempty,min=8"`
	Roles       *[]string `json:"roles"        validate:"omitempty,min=1,dive,required"`
}

type userJSON struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Document     string    `json:"document"`
	DocumentType string    `json:"document_type"`
	Email        string    `json:"email"`
	PhoneNumber  string    `json:"phone_number"`
	Roles        []string  `json:"roles"`
}

func toJSON(v application.UserView) userJSON {
	return userJSON{
		ID:           v.ID,
		Name:         v.Name,
		Document:     v.Document.Value,
		DocumentType: v.Document.Type(),
		Email:        v.Email.String(),
		PhoneNumber:  v.PhoneNumber.String(),
		Roles:        v.Roles.Strings(),
	}
}

func (h *Handler) create(ctx *fiber.Ctx) error {
	var body createBody
	if err := ctx.BodyParser(&body); err != nil {
		return h.errorHandler.Handle(ctx, ErrInvalidJSON)
	}
	if err := h.validator.Struct(body); err != nil {
		return h.errorHandler.Handle(ctx, err)
	}

	documentVO, err := document.New(body.Document)
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	emailVO, err := email.New(body.Email)
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	phoneVO, err := phone.New(body.PhoneNumber)
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	passwordVO, err := password.New(body.Password)
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	rolesVO, err := authz.NewRoles(body.Roles)
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}

	err = h.createUseCase.Execute(ctx.Context(), application.CreateInput{
		Name:        body.Name,
		Document:    documentVO,
		PhoneNumber: phoneVO,
		Email:       emailVO,
		Password:    passwordVO,
		Roles:       rolesVO,
	})
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusCreated)
}

func (h *Handler) update(ctx *fiber.Ctx) error {
	id, err := fiberHelper.GetUuidParam(ctx, "id")
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}

	var body updateBody
	if err := ctx.BodyParser(&body); err != nil {
		return h.errorHandler.Handle(ctx, ErrInvalidJSON)
	}

	if err := h.validator.Struct(body); err != nil {
		return h.errorHandler.Handle(ctx, err)
	}

	var emailVO *email.Email
	if body.Email != nil {
		vo, err := email.New(*body.Email)
		if err != nil {
			return h.errorHandler.Handle(ctx, err)
		}
		emailVO = &vo
	}

	var phoneVO *phone.PhoneNumber
	if body.PhoneNumber != nil {
		vo, err := phone.New(*body.PhoneNumber)
		if err != nil {
			return h.errorHandler.Handle(ctx, err)
		}
		phoneVO = &vo
	}

	var passwordVO *password.Password
	if body.Password != nil {
		vo, err := password.New(*body.Password)
		if err != nil {
			return h.errorHandler.Handle(ctx, err)
		}
		passwordVO = &vo
	}

	var rolesVO *authz.Roles
	if body.Roles != nil {
		vo, err := authz.NewRoles(*body.Roles)
		if err != nil {
			return h.errorHandler.Handle(ctx, err)
		}
		rolesVO = &vo
	}

	err = h.updateUseCase.Execute(ctx.Context(), application.UpdateInput{
		ID:          id,
		Name:        body.Name,
		PhoneNumber: phoneVO,
		Email:       emailVO,
		Password:    passwordVO,
		Roles:       rolesVO,
	})
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) delete(ctx *fiber.Ctx) error {
	id, err := fiberHelper.GetUuidParam(ctx, "id")
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	if err := h.deleteUseCase.Execute(ctx.Context(), application.DeleteInput{ID: id}); err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) getByID(ctx *fiber.Ctx) error {
	id, err := fiberHelper.GetUuidParam(ctx, "id")
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	out, err := h.getUseCase.Execute(ctx.Context(), application.GetInput{ID: id})
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(toJSON(out.User))
}

func (h *Handler) list(ctx *fiber.Ctx) error {
	pager := fiberHelper.NewPagination(ctx, 50, 0)

	out, err := h.listUseCase.Execute(ctx.Context(), application.ListInput{Pager: *pager})
	if err != nil {
		return h.errorHandler.Handle(ctx, err)
	}
	resp := maps.Map(out.Users, toJSON)
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"data": resp})
}
