// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: assignments.sql

package database

import (
	"context"
)

const getAssignment = `-- name: GetAssignment :one
select
  id, name, "order", created_at, updated_at, type
from assignments
where id = $1
`

func (q *Queries) GetAssignment(ctx context.Context, id int32) (Assignment, error) {
	row := q.db.QueryRow(ctx, getAssignment, id)
	var i Assignment
	err := row.Scan(
		&i.Id,
		&i.Name,
		&i.Order,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Type,
	)
	return i, err
}

const insertAssignment = `-- name: InsertAssignment :one
with max_order as (
  select case 
    when max("order") is null then 0
    else max("order")
  end as "order"
  from assignments
) insert into assignments(
  name,
  "type",
  "order"
) values ($1, $2, (select "order" + 1 from max_order)) returning id
`

func (q *Queries) InsertAssignment(ctx context.Context, name string, type_ string) (int32, error) {
	row := q.db.QueryRow(ctx, insertAssignment, name, type_)
	var id int32
	err := row.Scan(&id)
	return id, err
}

const updateAssignment = `-- name: UpdateAssignment :exec
update assignments set
  name = $2,
  "type" = $3,
  "order" = case when $3 is not null then $4 else assignments."order" end
where id = $1
`

type UpdateAssignmentParams struct {
	Id    int32  `db:"id"`
	Name  string `db:"name"`
	Type  string `db:"type"`
	Order int32  `db:"order"`
}

func (q *Queries) UpdateAssignment(ctx context.Context, arg UpdateAssignmentParams) error {
	_, err := q.db.Exec(ctx, updateAssignment,
		arg.Id,
		arg.Name,
		arg.Type,
		arg.Order,
	)
	return err
}

const getAssignmentsPage = `-- name: getAssignmentsPage :many
select
  id, name, "order", created_at, updated_at, type
from assignments
limit $1 offset $2
`

func (q *Queries) getAssignmentsPage(ctx context.Context, limit int32, offset int32) ([]Assignment, error) {
	rows, err := q.db.Query(ctx, getAssignmentsPage, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Assignment{}
	for rows.Next() {
		var i Assignment
		if err := rows.Scan(
			&i.Id,
			&i.Name,
			&i.Order,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Type,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}