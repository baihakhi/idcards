package main

import (
	"idcard/internal/config"
	"idcard/internal/handler"
	"idcard/internal/migrate"
	"idcard/internal/repository"
	"idcard/internal/service"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := config.InitDB("./data/users.db")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer config.CloseDB()

	if err := migrate.CreateTable(db); err != nil {
		log.Fatal(err)
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/pdf/", http.StripPrefix("/pdf/", http.FileServer(http.Dir("pdf"))))

	userRepo := repository.NewUserRepository(db)
	pdfSvc := service.NewPdfService()
	exclSvc := service.NewExcelService()
	userService := service.NewUserService(userRepo, pdfSvc, exclSvc)
	userHandler := handler.NewUserHandler(userService)

	// "/" Page
	http.HandleFunc("/", userHandler.IndexHandler)
	http.HandleFunc("/get", userHandler.GetUserHandler)
	http.HandleFunc("/get-id", userHandler.GetIdHandler)
	http.HandleFunc("/create", userHandler.CreateUserHandler)
	http.HandleFunc("/update", userHandler.UpdateUserHandler)

	// "/upload" Page
	http.HandleFunc("/upload", userHandler.UploadRedirecthandler)
	http.HandleFunc("/upload/upsert", userHandler.UploadHandler)

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
