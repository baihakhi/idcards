package service

import (
	"context"
	"database/sql"
	"fmt"
	"idcard/internal/model"
	"idcard/internal/repository"
	"idcard/internal/util"
	"io"
	"log"
	"strconv"
	"sync"
)

type (
	UserService interface {
		CreateUserAction(ctx context.Context, u *model.User) error
		GenerateUserID(ctx context.Context, status string) (string, error)
		GetUserList(ctx context.Context, limit uint8) (*[]model.User, error)
		GetUserByNik(ctx context.Context, nik string) (*model.User, error)
		UpdateUserAction(ctx context.Context, user *model.User) error
		BulkUpsertUser(ctx context.Context, file io.Reader) (int, error)
	}
	userServ struct {
		repo     repository.UserRepository
		pdfSvc   PdfService
		excelSvc ExcelService
	}

	Result struct {
		NIK      string
		Affected bool
		Err      error
	}
)

const (
	templatePath = util.PathToTempl + "kartu.png"
)

func NewUserService(repo repository.UserRepository, pdf PdfService, excel ExcelService) UserService {
	return &userServ{repo: repo, pdfSvc: pdf, excelSvc: excel}
}

func (s *userServ) CreateUserAction(ctx context.Context, u *model.User) error {
	if err := s.repo.Create(ctx, u); err != nil {
		return err
	}

	outputPath := util.PathToCard + u.ID + ".png"
	if err := util.GenerateIDCard(templatePath, u.Photo, outputPath, util.NormalizeName(u.Name), u.ID, u.Address); err != nil {
		return err
	}

	if err := s.excelSvc.UpdateExcel(u); err != nil {
		return err
	}

	if err := s.pdfSvc.PrintPDF(u); err != nil {
		return err
	}

	return nil
}

func (s *userServ) GenerateUserID(ctx context.Context, status string) (string, error) {
	res, err := s.repo.GetLastUserId(ctx, status)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	} else if err == sql.ErrNoRows {
		return fmt.Sprintf("%s%03d", status, 1), nil
	}

	userCount, err := strconv.Atoi(res[1:])
	return fmt.Sprintf("%s%03d", status, userCount+1), err
}

func (s *userServ) GetUserList(ctx context.Context, limit uint8) (*[]model.User, error) {
	users, err := s.repo.GetList(ctx, limit)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(*users); i++ {
		(*users)[i].Name = util.NormalizeName((*users)[i].Name)
	}

	return users, nil
}

func (s *userServ) GetUserByNik(ctx context.Context, nik string) (*model.User, error) {
	var u *model.User
	u, err := s.repo.GetUserByNik(ctx, nik)

	return u, err
}

func (s *userServ) UpdateUserAction(ctx context.Context, user *model.User) error {
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		log.Print("db: ", err)
		return err
	}

	outputPath := util.PathToCard + user.ID + ".png"
	if err := util.GenerateIDCard(templatePath, user.Photo, outputPath, util.NormalizeName(user.Name), user.ID, user.Address); err != nil {
		log.Print("id generator: ", err)
		return err
	}

	if err := s.excelSvc.UpdateExcel(user); err != nil {
		log.Print("excel: ", err)
		return err
	}

	if err := s.pdfSvc.PrintPDF(user); err != nil {
		log.Print("pdf: ", err)
		return err
	}

	return nil
}

func (s *userServ) BulkUpsertUser(ctx context.Context, file io.Reader) (int, error) {

	jobs := make(chan model.User)
	results := make(chan Result)
	var affected int
	var wg sync.WaitGroup

	numWorkers := 4

	rows, err := s.excelSvc.ParseExcel(file)
	if err != nil {
		return 0, fmt.Errorf("parse error: %w", err)
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.userWorker(ctx, tx, jobs, results)
		}()
	}

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
			u := model.User{
				ID:      row[0],
				Status:  row[1],
				NIK:     row[2],
				Name:    row[3],
				Phone:   row[4],
				Address: row[5],
				Rating:  util.ParseInt(row[6]),
				Notes:   ket,
				Photo:   foto,
			}
			jobs <- u
			log.Println("job sent: ", u.NIK)
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		if res.Err != nil {
			return 0, fmt.Errorf("bulk update failed: %w", res.Err)
		}
		if res.Affected {
			affected++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return affected, nil
}

func (s *userServ) userWorker(ctx context.Context, tx *sql.Tx, jobs <-chan model.User, results chan<- Result) {
	for u := range jobs {
		res := Result{NIK: u.NIK}
		log.Println("upsert: ", u.NIK)
		r, err := s.repo.UpsertUser(ctx, tx, u)
		if err != nil {
			res.Err = fmt.Errorf("upsert NIK %s: %w", u.NIK, err)
			results <- res
			continue
		}
		if r > 1 {
			res.Affected = true
		}
		log.Println("upsert done: ", u.NIK)
		results <- res
	}
}
