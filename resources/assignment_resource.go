package resources

import (
	"fmt"
	"strconv"

	"github.com/Kavantix/go-form/database"
	. "github.com/Kavantix/go-form/interfaces"
)

type AssignmentResource struct {
}

func (r AssignmentResource) Title() string {
	return "Assignments"
}

func (r AssignmentResource) FormConfig() FormConfig[database.AssignmentRow] {
	return FormConfig[database.AssignmentRow]{}
}

func (r AssignmentResource) Location(row *database.AssignmentRow) string {
	return fmt.Sprintf("/assignments/%d", row.Id)
}

func (r AssignmentResource) TableConfig() [](ColumnConfig[database.AssignmentRow]) {
	return [](ColumnConfig[database.AssignmentRow]){
		{Name: "Id", Value: func(user *database.AssignmentRow) string { return strconv.Itoa(int(user.Id)) }},
		{Name: "Name", Value: func(user *database.AssignmentRow) string { return user.Name }},
		{Name: "Type", Value: func(user *database.AssignmentRow) string { return user.Type }},
	}
}
