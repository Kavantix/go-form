package database

import (
	"fmt"
	"strings"
	"time"
)

type UserRow struct {
	Id          int32     `db:"id"`
	Name        string    `db:"name"`
	Email       string    `db:"email"`
	DateOfBirth time.Time `db:"date_of_birth"`
}

func GetUser(id int) (*UserRow, error) {
	users := []UserRow{}
	err := db.Select(&users, "select id, name, email, date_of_birth from users where id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	if len(users) != 1 {
		return nil, ErrNotFound
	}

	return &users[0], nil
}

func GetUsers(page, pageSize int) ([]UserRow, error) {
	users := []UserRow{}
	err := db.Select(&users,
		"select id, name, email, date_of_birth from users order by id limit $1 offset $2",
		pageSize, page,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	return users, nil
}

func IsEmailInUse(email string, excludeUserId int32) (bool, error) {
	result := CountResult{}
	err := db.Get(&result, "select count(*) as count from users where email = $1 and id != $2", email, excludeUserId)
	if err != nil {
		return true, err
	}
	return result.Count > 0, nil
}

func CreateUser(name, email string, dateOfBirth time.Time) (int, error) {
	row := db.QueryRowx(
		"insert into users(name, email, date_of_birth) values ($1, $2, $3) returning id",
		name, email, dateOfBirth,
	)
	result := struct {
		Id int `db:"id"`
	}{}
	err := row.StructScan(&result)
	if err != nil {
		if strings.Contains(err.Error(), `unique constraint "users_email_key"`) {
			return 0, ErrDuplicateEmail
		}
		return 0, fmt.Errorf("failed to insert user: %w", err)
	}
	return result.Id, nil
}

func UpdateUser(id int32, name, email string, dateOfBirth time.Time) error {
	_, err := db.Exec(
		"update users set name=$1, email=$2, date_of_birth=$3 where id = $4",
		name, email, dateOfBirth, id,
	)
	if err != nil {
		if strings.Contains(err.Error(), `unique constraint "users_email_key"`) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}
