package main

import (
	"idcard/internal/config"
	"idcard/internal/handler"
	"idcard/internal/repository"
	"idcard/internal/service"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}
	err := godotenv.Load(".env." + env)
	if err != nil {
		log.Println("Error loading environtment")
		return
	}

	// Initialize the database
	db, err := config.InitDB()
	if err != nil {
		log.Fatal(err)
		return
	}
	defer config.CloseDB()

	// if err := migrate.CreateTable(db); err != nil {
	// 	log.Fatal(err)
	// }

	http.Handle("/static/", withCORS(http.StripPrefix("/static/", http.FileServer(http.Dir("static")))))
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

// withCORS is a middleware that adds CORS headers to the response
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	})
}
