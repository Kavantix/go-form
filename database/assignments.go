package database

import "context"

func (q *Queries) GetAssignmentsPage(ctx context.Context, page, pageSize int) ([]Assignment, error) {
	return q.getAssignmentsPage(ctx, int32(pageSize), int32(page*pageSize))
}
