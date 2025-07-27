package handler

import (
	"idcard/internal/service"
)

type (
	UserHandler struct {
		UserService service.UserService
	}
)

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{UserService: svc}
}
