package database

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound       = pgx.ErrNoRows
	ErrDuplicateEmail = errors.New("email already exists")
)

func (q *Queries) ConsumeReloginToken(ctx context.Context, userId int32, token string, createdAfter time.Time) error {
	deletedIds, err := q.consumeReloginToken(ctx, token, userId, createdAfter)
	if err != nil {
		return err
	}
	if deletedIds <= 0 {
		return ErrNotFound
	}
	return nil
}

func checkDuplicateEmailErr(err error) error {
	if err != nil && strings.Contains(err.Error(), `unique constraint "users_email_key"`) {
		return ErrDuplicateEmail
	} else {
		return err
	}
}

func (q *Queries) InsertUser(ctx context.Context, name, email string, dateOfBirth time.Time) (int32, error) {
	id, err := q.insertUser(ctx, name, email, dateOfBirth)
	return id, checkDuplicateEmailErr(err)
}

type UpdateUserParams = updateUserParams

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) error {
	err := q.UpdateUser(ctx, arg)
	return checkDuplicateEmailErr(err)
}

func (q *Queries) GetUsersPage(ctx context.Context, page, pageSize int) ([]DisplayableUser, error) {
	return q.getUsersPage(ctx, int32(pageSize), int32(page*pageSize))
}
