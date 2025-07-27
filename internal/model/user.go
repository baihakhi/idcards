package model

import "time"

type User struct {
	ID        string
	Status    string
	NIK       string
	Name      string
	Phone     string
	Address   string
	Rating    int
	Notes     string
	Photo     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
