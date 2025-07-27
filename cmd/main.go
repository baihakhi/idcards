package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
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

	tmpl = template.Must(template.ParseGlob("templates/*.html"))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/pdf/", http.StripPrefix("/pdf/", http.FileServer(http.Dir("pdf"))))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/get", getUserHandler)
	http.HandleFunc("/get-id", getIdHandler)
	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/update", updateHandler)

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "upload file.html", nil)
	})
	http.HandleFunc("/upload/upsert", uploadHandler)

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

func indexHandler(w http.ResponseWriter, r *http.Request) {
	users := []User{}
	uID := "S001"
	rows, err := db.Query("SELECT id, nik, status, name, phone, address, rating, notes, photo, created_at, updated_at FROM users ORDER BY updated_at DESC LIMIT 15")

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.NIK, &u.Status, &u.Name, &u.Phone, &u.Address, &u.Rating, &u.Notes, &u.Photo, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		users = append(users, u)
	}

	if len(users) > 0 {
		lID, _ := strconv.Atoi(users[0].ID[1:])
		uID = fmt.Sprintf("S%03d", lID)
	}

	for i := 0; i < len(users); i++ {
		users[i].Name = normalizeName(users[i].Name)
	}

	tmpl.ExecuteTemplate(w, "index.html", map[string]any{
		"Action": "/create",
		"Method": "POST",
		"UserID": uID,
		"User":   users,
	})
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	var u User

	nik := r.URL.Query().Get("nik")

	err := db.QueryRow("SELECT * FROM users WHERE nik = ?", nik).Scan(&u.ID, &u.NIK, &u.Status, &u.Name, &u.Phone, &u.Address, &u.Rating, &u.Notes, &u.Photo, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		log.Println("err:", err)
		json.NewEncoder(w).Encode(map[string]string{"Error": fmt.Sprintf("error getting user of NIK: %s | %s", nik, err.Error())})
		return
	}

	log.Println("send data:", u)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"Data": u})
}

func getIdHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("get id")
	status := r.URL.Query().Get("status")

	userID, err := generateUserID(status)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"Error": fmt.Sprintf("error generating new ID for: %s | %s", status, err.Error())})
		return
	}

	log.Println("get ", status, " - ", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"Data": userID})
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("create handler")

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	nik := r.FormValue("nik")
	name := r.FormValue("name")
	status := r.FormValue("status")
	phone := r.FormValue("phone")
	address := r.FormValue("address")
	rating := r.FormValue("rating")
	if rating == "" {
		rating = "0"
	}
	ratingInt, _ := strconv.Atoi(rating)
	notes := r.FormValue("notes")
	photoData := r.FormValue("photo")
	userID, err := generateUserID(status)
	if err != nil {
		tmpl.ExecuteTemplate(w, "index.html", map[string]interface{}{
			"Error": err.Error(),
		})
		return
	}

	check, errLog := completionCheck(name, nik, photoData, userID, status)
	if !check {
		err := fmt.Errorf("lengkapi data, %s", errLog)
		tmpl.ExecuteTemplate(w, "index.html", map[string]interface{}{
			"Error": err.Error(),
		})
		return
	}

	data := photoData[strings.Index(photoData, ",")+1:]
	imgBytes, _ := base64.StdEncoding.DecodeString(data)
	photoPath := filepath.Join("static", "uploads", userID+".png")

	webPhotoPath := strings.ReplaceAll(photoPath, `\`, `/`)
	os.WriteFile(photoPath, imgBytes, 0644)

	log.Println(name, userID, webPhotoPath)
	_, err = db.Exec("INSERT INTO users (id, nik, status, name, phone, address, rating, notes, photo)"+
		"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userID, nik, status, name, phone, address, rating, notes, webPhotoPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	templatePath := pathToTempl + "kartu.png"
	outputPath := pathToCard + userID + ".png"
	err = generateIDCard(templatePath, webPhotoPath, outputPath, normalizeName(name), userID, address)
	if err != nil {
		tmpl.ExecuteTemplate(w, "index.html", map[string]interface{}{
			"Error": err.Error(),
		})
		return
	}

	u := User{
		ID:      userID,
		Status:  status,
		NIK:     nik,
		Name:    name,
		Phone:   phone,
		Address: address,
		Rating:  ratingInt,
		Notes:   notes,
		Photo:   webPhotoPath,
	}

	err = UpdateExcel(u)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_ = printPDF(u)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {

	log.Println("update handler ", r.Method)
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	userID := r.FormValue("userIdInput")
	nik := r.FormValue("nik")
	name := r.FormValue("name")
	status := r.FormValue("status")
	phone := r.FormValue("phone")
	address := r.FormValue("address")
	rating := r.FormValue("rating")
	if rating == "" {
		rating = "1"
	}
	notes := r.FormValue("notes")
	photoData := r.FormValue("photo")

	check, errLog := completionCheck(name, nik, photoData, userID, status)
	if !check {
		err := fmt.Errorf("lengkapi data, %s", errLog)
		log.Println(err)
		tmpl.ExecuteTemplate(w, "index.html", map[string]any{
			"Error": err.Error(),
		})
		return
	}

	data := photoData[strings.Index(photoData, ",")+1:]
	imgBytes, _ := base64.StdEncoding.DecodeString(data)
	photoPath := filepath.Join("static", "uploads", userID+".png")

	webPhotoPath := strings.ReplaceAll(photoPath, `\`, `/`)
	os.WriteFile(photoPath, imgBytes, 0644)

	log.Println("file written", name, userID, webPhotoPath)
	_, err := db.Exec("UPDATE users SET nik=?, status=?, name=?, phone=?, address=?, rating=?, notes=?, photo=? "+
		"WHERE users.id=?",
		nik, status, name, phone, address, rating, notes, webPhotoPath, userID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	templatePath := pathToTempl + "kartu.png"
	outputPath := pathToCard + userID + ".png"
	err = generateIDCard(templatePath, webPhotoPath, outputPath, normalizeName(name), userID, address)
	if err != nil {
		tmpl.ExecuteTemplate(w, "index.html", map[string]interface{}{
			"Error": err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("uploaded")
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Fatal(err, r.Header)
		http.Error(w, "Failed to parse request", 400)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", 400)
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, "Excel read error", 500)
		return
	}
	rows, err := f.GetRows("data")
	if err != nil || len(rows) < 2 {
		http.Error(w, "Invalid Excel content", 400)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", 500)
		return
	}

	// Channels and counters
	jobs := make(chan User)
	results := make(chan Result)
	var inserted, updated int

	// Spawn workers
	numWorkers := 4
	for range numWorkers {
		go userWorker(tx, jobs, results)
	}

	// Feed jobs
	go func() {
		for i := 1; i < len(rows); i++ {
			ket := ""
			foto := "static/assets/avatar.png"
			row := rows[i]
			if len(row) < 7 {
				results <- Result{Err: fmt.Errorf("row %d incomplete", i+1)}
				continue
			}
			if len(row) == 8 {
				ket = row[7]
			}
			if len(row) == 9 {
				foto = row[8]
			}
			u := User{
				ID:      row[0],
				Status:  row[1],
				NIK:     row[2],
				Name:    row[3],
				Phone:   row[4],
				Address: row[5],
				Rating:  parseInt(row[6]),
				Notes:   ket,
				Photo:   foto,
			}
			jobs <- u
		}
		close(jobs)
	}()

	// Collect results
	total := len(rows) - 1
	for range total {
		res := <-results
		if res.Err != nil {
			tx.Rollback()
			http.Error(w, "Bulk update failed: "+res.Err.Error(), 500)
			return
		}
		if res.Updated {
			updated++
		} else {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Commit failed", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message":  "Bulk update success",
		"inserted": inserted,
		"updated":  updated,
	})
}

func userWorker(tx *sql.Tx, jobs <-chan User, results chan<- Result) {
	for u := range jobs {
		res := Result{NIK: u.NIK}

		// Try update
		r, err := tx.Exec(`UPDATE users SET status=?, name=?, phone=?, address=?, rating=?, notes=?, photo=? WHERE nik=?`,
			u.Status, u.Name, u.Phone, u.Address, u.Rating, u.Notes, u.Photo, u.NIK)
		if err != nil {
			res.Err = fmt.Errorf("update NIK %s: %w", u.NIK, err)
			results <- res
			continue
		}
		count, _ := r.RowsAffected()
		if count > 0 {
			res.Updated = true
			results <- res
			continue
		}

		// Insert if not updated
		_, err = tx.Exec(`INSERT INTO users (id, status, nik, name, phone, address, rating, notes, photo)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			u.ID, u.Status, u.NIK, u.Name, u.Phone, u.Address, u.Rating, u.Notes, u.Photo)
		if err != nil {
			res.Err = fmt.Errorf("insert NIK %s: %w", u.NIK, err)
		}
		results <- res
	}
}

func GetUserByNik(nik string) (u User, err error) {
	err = db.QueryRow("SELECT * FROM users WHERE nik = ?", nik).Scan(&u.ID, &u.NIK, &u.Status, &u.Name, &u.Phone, &u.Address, &u.Rating, &u.Notes, &u.Photo, &u.CreatedAt, &u.UpdatedAt)
	if err != nil && err != sql.ErrNoRows {
		return u, fmt.Errorf("error checking NIK: %w", err)
	}
	return u, nil
}

func generateUserID(status string) (string, error) {
	var (
		total int
	)

	err := db.QueryRow("SELECT COUNT(id) AS total_user FROM users where status=?", status).Scan(&total)
	if err != nil {
		return "", err
	}

	userID := status + fmt.Sprintf("%03d", total+2)
	return userID, nil
}

func completionCheck(m map[string]string) (bool, string) {
	check := false
	for key, value := range m {
		if value == "" {
			return check, fmt.Sprintf("%s masih kosong", key)
		} else {
			check = true
		}
	}
	return check, ""
}

func ExecOrFail(query string) {
	if _, err := db.Exec(query); err != nil {
		log.Fatal(err)
	}
}
