package repository

import (
	"context"
	"database/sql"
	"idcard/internal/model"
	"time"
)

type (
	UserRepository interface {
		Begin(ctx context.Context) (*sql.Tx, error)
		Create(ctx context.Context, user *model.User) error
		GetList(ctx context.Context, limit uint8) (*[]model.User, error)
		GetLastUserId(ctx context.Context, status string) (string, error)
		GetUserByNik(ctx context.Context, nik string) (user *model.User, err error)
		UpdateUser(ctx context.Context, u *model.User) error
		UpsertUser(ctx context.Context, tx *sql.Tx, u model.User) (int64, error)
	}
	userRepo struct {
		db *sql.DB
	}
)

func NewUserRepository(database *sql.DB) UserRepository {
	return &userRepo{db: database}
}

func (r *userRepo) Begin(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *userRepo) Create(ctx context.Context, u *model.User) error {
	query := `INSERT INTO users (id, nik, status, name, phone, address, rating, notes, photo) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query, u.ID, u.NIK, u.Status, u.Name, u.Phone, u.Address, u.Rating, u.Notes, u.Photo)

	return err
}

func (r *userRepo) GetList(ctx context.Context, limit uint8) (*[]model.User, error) {
	users := []model.User{}
	rows, err := r.db.Query("SELECT id, nik, status, name, phone, address, rating, notes, photo, created_at, updated_at FROM users ORDER BY updated_at DESC LIMIT ?", limit)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var u model.User
		err := rows.Scan(&u.ID, &u.NIK, &u.Status, &u.Name, &u.Phone, &u.Address, &u.Rating, &u.Notes, &u.Photo, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return &users, nil
}

func (r *userRepo) GetLastUserId(ctx context.Context, status string) (string, error) {
	var (
		uID string
		tgl time.Time
	)

	err := r.db.QueryRow("SELECT ID, created_at FROM users where status=? ORDER BY created_at DESC LIMIT 1", status).Scan(&uID, &tgl)
	if err == sql.ErrNoRows {
		uID = "S001"
		return uID, nil
	}

	return uID, err
}

func (r *userRepo) GetUserByNik(ctx context.Context, nik string) (user *model.User, err error) {
	var u model.User
	err = r.db.QueryRow("SELECT * FROM users WHERE nik = ?", nik).Scan(&u.ID, &u.NIK, &u.Status, &u.Name, &u.Phone, &u.Address, &u.Rating, &u.Notes, &u.Photo, &u.CreatedAt, &u.UpdatedAt)

	return &u, err
}

func (r *userRepo) UpdateUser(ctx context.Context, u *model.User) error {
	_, err := r.db.Exec("UPDATE users SET nik=?, status=?, name=?, phone=?, address=?, rating=?, notes=?, photo=? WHERE users.id=?",
		u.NIK, u.Status, u.Name, u.Phone, u.Address, u.Rating, u.Notes, u.Photo, u.ID)
	return err
}

func (r *userRepo) UpsertUser(ctx context.Context, tx *sql.Tx, u model.User) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO users (id, status, nik, name, phone, address, rating, notes, photo)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (id) DO UPDATE SET 
			name = EXCLUDED.name,
			nik =  EXCLUDED.nik,
			phone = EXCLUDED.phone,
			address = EXCLUDED.address,
			rating = EXCLUDED.rating,
			notes = EXCLUDED.notes,
			photo = EXCLUDED.photo`, u.ID, u.Status, u.NIK, u.Name, u.Phone, u.Address, u.Rating, u.Notes, u.Photo)
	affected, _ := res.RowsAffected()
	return affected, err
}
