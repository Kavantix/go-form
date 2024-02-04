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

type ReloginTokenRow struct {
	Id        int32     `db:"id"`
	UserId    int32     `db:"user_id"`
	Token     string    `db:"token"`
	CreatedAt time.Time `db:"created_at"`
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

func GetUserByEmail(email string) (*UserRow, error) {
	users := []UserRow{}
	err := db.Select(&users, "select id, name, email, date_of_birth from users where email = $1", email)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user by email: %w", err)
	}
	if len(users) != 1 {
		return nil, ErrNotFound
	}

	return &users[0], nil
}

func InsertReloginToken(userId int32, token string) (int, error) {
	result := struct {
		Id int `db:"id"`
	}{}
	err := db.Get(
		&result,
		"insert into relogin_tokens (user_id, token) values ($1, $2) returning id",
		userId, token,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert relogin token: %w", err)
	}
	return result.Id, nil
}

func ConsumeReloginToken(userId int32, token string, createdAfter time.Time) error {
	result, err := db.Exec(
		"delete from relogin_tokens where token = $1 and user_id = $2 and created_at > $3",
		token, userId, createdAfter,
	)
	if err != nil {
		return fmt.Errorf("failed to consume relogin token: %w", err)
	}
	rows, err := result.RowsAffected()
	fmt.Printf("Consumed %d tokens\n", rows)
	if err != nil {
		panic("database driver does not support rows affected")
	}
	if rows <= 0 {
		return ErrNotFound
	}
	return nil
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
