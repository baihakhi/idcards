package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"idcard/internal/model"
	"idcard/internal/service"
	"idcard/internal/util"
	"log"
	"net/http"
	"strconv"
)

type (
	UserHandler struct {
		UserService service.UserService
	}
)

var (
	tmpl *template.Template
)

func NewUserHandler(svc service.UserService) *UserHandler {
	tmpl = template.Must(template.ParseGlob("templates/*.html"))
	return &UserHandler{UserService: svc}
}

func (h *UserHandler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	limitQ := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitQ)
	if err != nil {
		limit = 12 // default 12
	}

	ctx := r.Context()
	users, err := h.UserService.GetUserList(ctx, uint8(limit))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	lastUID, err := h.UserService.GenerateUserID(ctx, "s")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"Error": fmt.Sprintf("error generating new ID for: %s", err.Error())})
		return
	}

	tmpl.ExecuteTemplate(w, "index.html", map[string]any{
		"Action": "/create",
		"Method": "POST",
		"UserID": lastUID,
		"User":   users,
	})
}

func (h *UserHandler) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	nik := r.URL.Query().Get("nik")

	user, err := h.UserService.GetUserByNik(r.Context(), nik)
	if err != nil {
		log.Println("err:", err)
		json.NewEncoder(w).Encode(map[string]string{"Error": fmt.Sprintf("error getting user of NIK: %s | %s", nik, err.Error())})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"Data": user})
}

func (h *UserHandler) GetIdHandler(w http.ResponseWriter, r *http.Request) {
	status := "s"
	statusQ := r.URL.Query().Get("status")
	if statusQ != "" {
		status = statusQ
	}

	userID, err := h.UserService.GenerateUserID(r.Context(), status)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"Error": fmt.Sprintf("error generating new ID for: %s | %s", status, err.Error())})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"Data": userID})
}

func (h *UserHandler) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	ctx := r.Context()
	status := r.FormValue("status")
	userId, err := h.UserService.GenerateUserID(ctx, status)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"Error": fmt.Sprintf("error generating new ID for: %s | %s", status, err.Error())})
		return
	}

	rating := "0"
	if r.FormValue("ratig") != "" {
		rating = r.FormValue("ratig")
	}

	formData := map[string]string{
		"id":        userId,
		"nik":       r.FormValue("nik"),
		"name":      r.FormValue("name"),
		"status":    status,
		"phone":     r.FormValue("phone"),
		"address":   r.FormValue("address"),
		"rating":    rating,
		"notes":     r.FormValue("notes"),
		"photoData": r.FormValue("photo"),
	}

	check, errLog := util.CompletionCheck(formData)
	if !check {
		err := fmt.Errorf("lengkapi data, %s", errLog)
		json.NewEncoder(w).Encode(map[string]error{"Error": err})
		return
	}

	imgByte, err := util.StringtoByte(formData["photoData"])
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"Error": fmt.Sprintf("picture decoder: %s", err.Error()),
		})
		return
	}

	imgPath := util.ImageWriter(imgByte, `static\uploads`, userId, ".png")

	err = h.UserService.CreateUserAction(ctx, &model.User{
		ID:      formData["id"],
		NIK:     formData["nik"],
		Name:    formData["name"],
		Status:  formData["status"],
		Phone:   formData["phone"],
		Address: formData["address"],
		Rating:  util.ParseInt(rating),
		Notes:   formData["notes"],
		Photo:   imgPath,
	})
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"Error": fmt.Sprintf("picture decoder: %s", err.Error()),
		})
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
