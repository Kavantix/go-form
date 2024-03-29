// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package database

import (
	"time"
)

type Assignment struct {
	Id        int32     `db:"id"`
	Name      string    `db:"name"`
	Order     int32     `db:"order"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Type      string    `db:"type"`
}

type DisplayableUser struct {
	Id          int32     `db:"id"`
	Name        string    `db:"name"`
	Email       string    `db:"email"`
	DateOfBirth time.Time `db:"date_of_birth"`
}

type ReloginToken struct {
	Id        int32     `db:"id"`
	Token     string    `db:"token"`
	UserID    int32     `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
}

type User struct {
	Id          int32     `db:"id"`
	Email       string    `db:"email"`
	CreatedAt   time.Time `db:"created_at"`
	DateOfBirth time.Time `db:"date_of_birth"`
	Name        string    `db:"name"`
	UpdatedAt   time.Time `db:"updated_at"`
}
