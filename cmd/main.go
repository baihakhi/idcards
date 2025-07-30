package main

import (
	"database/sql"
	"html/template"
	"idcard/internal/handler"
	"idcard/internal/repository"
	"idcard/internal/service"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Result struct {
	NIK     string
	Updated bool
	Err     error
}

var (
	db   *sql.DB
	tmpl *template.Template
)

const (
	pathToTempl string = "static/assets/"
	pathToCard  string = "cards/idcards/"
	pathToFont  string = "static/assets/fonts/"
)

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./data/users.db")
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	createTable()

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
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "upload file.html", nil)
	})
	http.HandleFunc("/upload/upsert", userHandler.UploadHandler)

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func createTable() {
	query := `CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(4) PRIMARY KEY NOT NULL,
		nik VARCHAR(16) NOT NULL UNIQUE,
		status CHAR(1) NOT NULL,
		name VARCHAR(255) NOT NULL,
		phone VARCHAR(20) NOT NULL,
		address VARCHAR(255) NOT NULL,
		rating INTEGER DEFAULT 0,
		notes TEXT DEFAULT NULL,
		photo VARCHAR(255) NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	trigger := `CREATE TRIGGER IF NOT EXISTS update_users_updated_at
		AFTER UPDATE ON users
		FOR EACH ROW
		BEGIN
		UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
		END;`

	idxNik := `CREATE INDEX IF NOT EXISTS idx_users_nik ON users(nik);`

	idxStatus := `CREATE INDEX IF NOT EXISTS idx_users_sopir ON users(status)
		WHERE status = 'S';`

	ExecOrFail(query)
	ExecOrFail(trigger)
	ExecOrFail(idxNik)
	ExecOrFail(idxStatus)
}

func ExecOrFail(query string) {
	if _, err := db.Exec(query); err != nil {
		log.Fatal(err)
	}
}
