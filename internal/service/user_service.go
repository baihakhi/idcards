package service

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"idcard/internal/config"
	"idcard/internal/model"
	"idcard/internal/repository"
	"idcard/internal/util"
	"io"
	"strconv"
	"sync"
)

type (
	UserService interface {
		CreateUserAction(ctx context.Context, u *model.User, photo []byte) error
		GenerateUserID(ctx context.Context, status string) (string, error)
		GetUserList(ctx context.Context, limit uint8) (*[]model.User, error)
		GetUserByNik(ctx context.Context, nik string) (*model.User, error)
		UpdateUserAction(ctx context.Context, user *model.User, photo []byte) error
		BulkUpsertUser(ctx context.Context, file io.Reader) (int, error)
	}
	userServ struct {
		repo          repository.UserRepository
		storageClient config.Client
		pdfSvc        PdfService
		excelSvc      ExcelService
	}

	Result struct {
		NIK      string
		Affected bool
		Err      error
	}
)

const (
	templatePath = util.PathToAssets + "kartu.png"
)

func NewUserService(repo repository.UserRepository, pdf PdfService, excel ExcelService, storage config.Client) UserService {
	return &userServ{repo: repo, pdfSvc: pdf, excelSvc: excel, storageClient: storage}
}

func (s *userServ) CreateUserAction(ctx context.Context, u *model.User, photo []byte) error {
	if err := s.repo.Create(ctx, u); err != nil {
		return err
	}

	err := s.imageSequenceAction(ctx, u, photo)
	if err != nil {
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

func (s *userServ) UpdateUserAction(ctx context.Context, u *model.User, photo []byte) error {
	if err := s.repo.UpdateUser(ctx, u); err != nil {
		return err
	}

	err := s.imageSequenceAction(ctx, u, photo)
	if err != nil {
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
		r, err := s.repo.UpsertUser(ctx, tx, u)
		if err != nil {
			res.Err = fmt.Errorf("upsert NIK %s: %w", u.NIK, err)
			results <- res
			continue
		}
		if r > 1 {
			res.Affected = true
		}
		results <- res
	}
}

func (s *userServ) imageSequenceAction(ctx context.Context, u *model.User, imgByte []byte) error {
	photoFile := bytes.NewReader(imgByte)
	ext := util.GetFileFormat(u.Photo)
	mime := util.GetMimeType(u.Photo)
	if err := util.GenerateIDCard(templatePath, util.NormalizeName(u.Name), u.ID, u.Address, fmt.Sprintf("%s%s.png", util.PathToCard, u.ID), photoFile); err != nil {
		err = fmt.Errorf("generate ID card: %w", err)
		return err
	}

	if err := s.pdfSvc.PrintPDF(u, fmt.Sprintf("%s%s.pdf", util.PathToContract, u.ID)); err != nil {
		err = fmt.Errorf("generate PDF: %w", err)
		return err
	}

	// Upload user photo to storage
	if err := s.storageClient.Upload(ctx, "images/"+u.ID+"."+ext, mime, photoFile); err != nil {
		err = fmt.Errorf("upload photo: %w", err)
		return err
	}

	// --TO BE REMOVED LATER--
	if err := s.excelSvc.UpdateExcel(u); err != nil {
		err = fmt.Errorf("update excel: %w", err)
		return err
	}

	return nil
}
