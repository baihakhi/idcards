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

	"github.com/jung-kurt/gofpdf"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"

	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type User struct {
	ID    string
	Status string
	NIK string
	Name  string
	Phone string
	Address string
	Rating int
	Notes string
	Photo string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Result struct {
	NIK     string
	Updated bool
	Err     error
}

var (
	db *sql.DB
	tmpl *template.Template
)

const (
	pathToTempl string = "static/assets/"
	pathToCard string = "cards/idcards/"
	pathToFont string = "static/assets/fonts/"
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

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request){
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
			"User": users,
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

	nik		:= r.FormValue("nik")
	name 	:= r.FormValue("name")
	status 	:= r.FormValue("status")
	phone 	:= r.FormValue("phone")
	address := r.FormValue("address")
	rating 	:= r.FormValue("rating")
	if rating == "" {
		rating = "0"
	}
	ratingInt, _ := strconv.Atoi(rating)
	notes 	:= r.FormValue("notes")
	photoData := r.FormValue("photo")
	userID, err := generateUserID(status)
	if err != nil {
		tmpl.ExecuteTemplate(w, "index.html", map[string]interface{}{
			"Error": err.Error(),
		})
		return
	}
	
	check, errLog := completionCheck(name, nik, photoData, userID, status)
	if !check{
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
	_, err = db.Exec("INSERT INTO users (id, nik, status, name, phone, address, rating, notes, photo)" +
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
		ID: userID,
		Status: status,
		NIK: nik,
		Name: name,
		Phone: phone,
		Address: address,
		Rating: ratingInt,
		Notes: notes,
		Photo: webPhotoPath,
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
	nik		:= r.FormValue("nik")
	name 	:= r.FormValue("name")
	status 	:= r.FormValue("status")
	phone 	:= r.FormValue("phone")
	address := r.FormValue("address")
	rating 	:= r.FormValue("rating")
	if rating == "" {
		rating = "1"
	}
	notes 	:= r.FormValue("notes")
	photoData := r.FormValue("photo")


	check, errLog := completionCheck(name, nik, photoData, userID, status)
	if !check{
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

	log.Println("file written",name, userID, webPhotoPath)
	_, err := db.Exec("UPDATE users SET nik=?, status=?, name=?, phone=?, address=?, rating=?, notes=?, photo=? " +
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
				Photo: foto,
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

func parseInt(s string) int {
	i, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return i
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

func generateIDCard(templatePath, photoPath, outputPath, name, userID, alamat string) error {
	roboto200 := pathToFont + "Roboto/static/Roboto-Light.ttf"
	roboto400 := pathToFont + "Roboto/static/Roboto-Regular.ttf"

	bgFile, err := os.Open(templatePath)
	if err != nil {
		return err
	}
	defer bgFile.Close()
	bgImg, _, _ := image.Decode(bgFile)

	photoFile, err := os.Open(photoPath)
	if err != nil {
		return err
	}
	defer photoFile.Close()
	photoImg, _, _ := image.Decode(photoFile)

	// TODO:Resize photo 

	card := image.NewRGBA(bgImg.Bounds())
	draw.Draw(card, bgImg.Bounds(), bgImg, image.Point{}, draw.Src)

	photoPosition := image.Rect(155, 320, 490, 770) 
	draw.Draw(card, photoPosition, photoImg, image.Point{}, draw.Over)

	err = drawText(card, name, 240, 818, 32, roboto400, color.Black)
	if err != nil {
		return err
	}
	err = drawText(card, "SIK-"+userID, 240, 862, 28, roboto400, color.Black)
	if err != nil {
		return err
	}

	arr := strings.SplitN(alamat, ",", 2)
	_ = drawText(card, arr[0], 240, 910, 24, roboto200, color.Black)
	if len(arr) == 2 {
		_ = drawText(card, strings.TrimLeft(arr[1], " "), 240, 945, 24, roboto200, color.Black)
	}

	outFile, _ := os.Create(outputPath)
	defer outFile.Close()
	return png.Encode(outFile, card)
}

func drawText(img *image.RGBA, text string, x, y int, fontSize float64, fontPath string, col color.Color) error {
	// Load TTF font
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return err
	}

	ft, err := opentype.Parse(fontBytes)
	if err != nil {
		return err
	}

	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer face.Close()

	// Set up drawer
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col), // e.g. colornames.Black
		Face: face,
		Dot:  fixed.P(x, y),
	}

	d.DrawString(text)
	return nil
}

func normalizeName(nama string) string {
	maxChar := 20
	if len(nama) <= maxChar {
		return nama
	}
	fN := strings.Split(nama, " ")
	if len(fN) == 1 {
		return fN[0][:maxChar]
	}
	if len(fN[0])>=7 && fN[0][:1] == "M" && fN[0][len(fN[0])-2:] == "AD" {
		fN[0] = "M"
	}
	if len(strings.Join(fN[0:3], " ")) >= maxChar-2 {
		fN[2] = fN[2][:1]
	}
	res := strings.Join(fN, " ")
	if len(res) < maxChar {
		maxChar = len(res)
	}
	
	return res[:maxChar]
}

func completionCheck(name, nik, photoData, userID, status string) (bool,string) {
	check := false
	if name == "" {
		return check, "nama masih kosong"
	} else if nik == "" {
		return check,"nik masih kosong"
	} else if photoData == "" {
		return check,"foto masih kosong"
	} else if status == "" {
		return check, "status masih kosong"
	} else if userID == "" {
		return check, "ID gagal dibuat"
	} else {
		check = true
	}
	return check, ""
}

func ExecOrFail(query string) {
	if _, err := db.Exec(query); err != nil {
		log.Fatal(err)
	}
}

func UpdateExcel(u User) error {
	baseDir := "./data/"
	filePath := filepath.Join(baseDir, "data.xlsx")

	// Open the existing file
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		log.Printf("Error opening %s: %v\n", filePath, err)
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Println("Failed to close Excel file:", cerr)
		}
	}()

	sheet := "data"
	rows, err := f.GetRows(sheet)
	if err != nil {
		log.Printf("Error reading rows in %s: %v\n", filePath, err)
		return err
	}

	// Append to next row
	rowIndex := len(rows) + 1

	f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIndex), u.ID)
	f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), u.Status)
	f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIndex), u.NIK)
	f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIndex), u.Name)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIndex), u.Phone)
	f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIndex), u.Address)
	f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIndex), u.Rating)
	f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIndex), u.Notes)
	f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIndex), u.Photo)

	if err := f.Save(); err != nil {
		log.Printf("Failed to save file: %v\n", err)
		return err
	}

	return nil
}

func printPDF(user User) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Formulir Pendaftaran Penyetor Afval", false)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "Formulir Pendaftaran Penyetor Afval", "", 1, "C", false, 0, "")
	pdf.Ln(0)
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 10, "PT. Sinar Indah Kertas", "", 1, "C", false, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 10, "Identitas Penyetor Afval")
	pdf.Ln(8)
	pdf.Cell(0, 8, fmt.Sprintf("Nama     : %s", user.Name))
	pdf.Ln(8)
	pdf.Cell(0, 8, fmt.Sprintf("NIK         : %s", user.NIK))
	pdf.Ln(8)
	pdf.Cell(0, 8, fmt.Sprintf("No. Telp  : %s", user.Phone))
	pdf.Ln(8)
	pdf.MultiCell(0, 8, fmt.Sprintf("Alamat    : %s", user.Address), "", "", false)
	pdf.Ln(8)

	// Declaration statements
	statements := []string{
		"Dengan menandatangani formulir ini saya menyatakan:",
		"    1. Saya mengajukan/mendaftar sebagai penyetor afval PT. Sinar Indah Kertas.",
		"    2. Afval yang saya setor adalah hasil kegiatan yang sah dan tidak melanggar hukum.",
		"    3. Saya berkomitmen untuk menjaga kualitas dan kejujuran dalam setiap setoran.",
		"    4. Saya bersedia menjalani proses inspeksi & verifikasi sesuai sistem QC yang diterapkan.",
		"  5. PT. Sinar Indah Kertas berhak menolak apabila kualitas afval tidak memenuhi standar      perusahaan.",
		"    6. Saya bersedia mengikuti tata tertib yang berlaku, di antaranya:",
		"       a. Tidak mengambil gambar/foto/video di area perusahaan.",
		"       b. Tidak merokok di area perusahaan.",
		"       c. Tidak melanggar batas kecepatan kendaraan di area perusahaan.",
		"   7. Saya menyadari bahwa pelanggaran terhadap komitmen dapat berdampak pada pemutusan      kerjasama.",
		"  8. Saya menyatakan bahwa data & pernyataan yang saya berikan adalah benar dan dapat       dipertanggungjawabkan.",
	}

	for _, line := range statements {
		pdf.MultiCell(0, 8, line, "", "", false)
	}
	pdf.Ln(10)

	// Signature block
	pdf.CellFormat(0, 6, "Kudus, 24 Juli 2025", "", 1, "R", false, 0, "")
	pdf.Ln(18)
	pdf.CellFormat(0, 6, user.Name, "", 1, "R", false, 0, "")

	// Output
	if err := os.MkdirAll("output", os.ModePerm); err != nil {
		return err
	}
	filename := fmt.Sprintf("pdf/form_%s.pdf", user.ID)
	return pdf.OutputFileAndClose(filename)
}
