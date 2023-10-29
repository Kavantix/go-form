package database

import (
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
	users := []AssignmentRow{}
	err := db.Select(&users, fmt.Sprintf(
		`select id, name, "order", "type" from assignments order by "order" limit %d offset %d`,
		pageSize, page,
	))
	if err != nil {
		log.Fatal(err)
		return nil, fmt.Errorf("failed to query assignments: %w", err)
	}
	return users, nil
}
