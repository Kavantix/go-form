package database

import (
	"database/sql"
	"fmt"
	"log"
)

type AssignmentRow struct {
	Id    int32  `db:"id"`
	Name  string `db:"name"`
	Type  string `db:"type"`
	Order int32  `db:"order"`
}

func GetAssignments(page, pageSize int) ([]AssignmentRow, error) {
	assignments := []AssignmentRow{}
	err := db.Select(&assignments,
		`select id, name, "order", "type" from assignments order by "order" limit $1 offset $2`,
		pageSize, page,
	)
	if err != nil {
		log.Fatal(err)
		return nil, fmt.Errorf("failed to query assignments: %w", err)
	}
	return assignments, nil
}

func GetAssignment(id int) (*AssignmentRow, error) {
	assignments := []AssignmentRow{}
	err := db.Select(&assignments,
		`select id, name, "order", "type" from assignments where id = $1`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query assignments: %w", err)
	}
	if len(assignments) != 1 {
		return nil, ErrNotFound
	}
	return &assignments[0], nil
}

func CreateAssignment(name, Type string) (int32, error) {
	maxOrder := 0
	{
		result := struct {
			Order sql.NullInt32 `db:"order"`
		}{}
		err := db.Get(&result, `select max("order") as "order" from assignments`)
		if err != nil {
			return 0, fmt.Errorf("failed to insert assignment: %w", err)
		}
		maxOrder = int(result.Order.Int32)
	}
	result := struct {
		Id int32 `db:"id"`
	}{}
	err := db.Get(
		&result,
		`insert into assignments(name, "type", "order") values ($1, $2, $3) returning id`,
		name, Type, maxOrder+1,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert assignment: %w", err)
	}
	return result.Id, nil
}

func UpdateAssignment(id int32, name, Type string) error {
	_, err := db.Exec(
		`update assignments set name=$1, "type"=$2 where id = $3`,
		name, Type, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update assignment: %w", err)
	}
	return nil
}
