package database

import (
	"context"
	"fmt"
	"github.com/Kavantix/go-form/newdatabase"
	"strings"
	"time"
)

type UserRow = newdatabase.DisplayableUser

type ReloginTokenRow struct {
	Id        int32     `db:"id"`
	UserId    int32     `db:"user_id"`
	Token     string    `db:"token"`
	CreatedAt time.Time `db:"created_at"`
}

func GetUser(ctx context.Context, id int) (UserRow, error) {
	return queries.GetUser(ctx, int32(id))
}

func GetUserByEmail(ctx context.Context, email string) (UserRow, error) {
	return queries.GetUserByEmail(ctx, email)
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

func GetUsers(ctx context.Context, page, pageSize int) ([]UserRow, error) {
	return queries.GetUsersPage(ctx, int32(pageSize), int32(page*pageSize))
}

func IsEmailInUse(ctx context.Context, email string, excludeUserId int32) (bool, error) {
	return queries.UserWithEmailExists(ctx, email, excludeUserId)
}

func checkDuplicateEmailErr(err error) error {
	if strings.Contains(err.Error(), `unique constraint "users_email_key"`) {
		return ErrDuplicateEmail
	} else {
		return err
	}
}

func CreateUser(ctx context.Context, name, email string, dateOfBirth time.Time) (int32, error) {
	id, err := queries.InsertUser(ctx, name, email, dateOfBirth)
	return id, checkDuplicateEmailErr(err)
}

func UpdateUser(ctx context.Context, id int32, name, email string, dateOfBirth time.Time) error {
	err := queries.UpdateUser(ctx, newdatabase.UpdateUserParams{
		Id:          id,
		Name:        name,
		Email:       email,
		DateOfBirth: dateOfBirth,
	})
	return checkDuplicateEmailErr(err)
}
