package handler

import (
	"context"
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	limitQ := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitQ)
	if err != nil {
		log.Println(err)
		limit = 12 // default 12
	}

	ctx := r.Context()
	users, err := h.UserService.GetUserList(ctx, uint8(limit))
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}

	lastUID, err := h.UserService.GenerateUserID(ctx, "S")
	if err != nil {
		log.Println(err)
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	nik := r.URL.Query().Get("nik")

	user, err := h.UserService.GetUserByNik(r.Context(), nik)
	if err != nil {
		log.Println(err)
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
		log.Println(err)
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
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{"Error": fmt.Sprintf("error generating new ID for: %s | %s", status, err.Error())})
		return
	}

	rating := "0"
	if r.FormValue("rating") != "" {
		rating = r.FormValue("rating")
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
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]error{"Error": err})
		return
	}

	imgByte, err := util.StringtoByte(formData["photoData"])
	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{
			"Error": fmt.Sprintf("picture decoder: %s", err.Error()),
		})
		return
	}

	imgPath, err := util.ImageWriter(imgByte, `static\uploads`, userId, ".png")
	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{
			"Error": fmt.Sprintf("image writer: %s", err.Error()),
		})
		return
	}
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
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{
			"Error": fmt.Sprintf("db connection: %s", err.Error()),
		})
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *UserHandler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	ctx := r.Context()
	rating := "0"
	notes := " "
	if r.FormValue("rating") != "" {
		rating = r.FormValue("rating")
	}
	if r.FormValue("notes") != "" {
		notes = r.FormValue("notes")
	}

	formData := map[string]string{
		"id":        r.FormValue("userIdInput"),
		"nik":       r.FormValue("nik"),
		"name":      r.FormValue("name"),
		"status":    r.FormValue("status"),
		"phone":     r.FormValue("phone"),
		"address":   r.FormValue("address"),
		"rating":    rating,
		"notes":     notes,
		"photoData": r.FormValue("photo"),
	}

	check, errLog := util.CompletionCheck(formData)
	if !check {
		err := fmt.Errorf("lengkapi data, %s", errLog)
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{"Error": err.Error()})
		return
	}

	imgByte, err := util.StringtoByte(formData["photoData"])
	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{
			"Error": fmt.Sprintf("image decoder: %s", err.Error()),
		})
		return
	}

	imgPath, err := util.ImageWriter(imgByte, `static\uploads`, formData["id"], ".png")
	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{
			"Error": fmt.Sprintf("image writer: %s", err.Error()),
		})
		return
	}

	err = h.UserService.UpdateUserAction(ctx, &model.User{
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
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{
			"Error": fmt.Sprintf("user update service: %s", err.Error()),
		})
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *UserHandler) UploadRedirecthandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "upload file.html", nil)
}

func (h *UserHandler) UploadHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to get file", 400)
		return
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(r.Context(), util.Timeout)
	defer cancel()

	affected, err := h.UserService.BulkUpsertUser(ctx, file)
	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{
			"Error": fmt.Sprintf("picture decoder: %s", err.Error()),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message":  "Bulk update success",
		"affected": affected,
	})
}
