package resources

import (
	"fmt"
	"strconv"

	"github.com/Kavantix/go-form/database"
	. "github.com/Kavantix/go-form/interfaces"
	"github.com/Kavantix/go-form/templates/components"
)

type AssignmentResource struct {
}

func (r AssignmentResource) Title() string {
	return "Assignments"
}

func (r AssignmentResource) FetchPage(page, pageSize int) ([]database.AssignmentRow, error) {
	return database.GetAssignments(page, pageSize)
}

func (r AssignmentResource) FetchRow(id int) (*database.AssignmentRow, error) {
	return database.GetAssignment(id)
}

func (r AssignmentResource) ParseRow(id *int, formFields map[string]string) (*database.AssignmentRow, error) {
	assignment := database.AssignmentRow{}
	if id != nil {
		assignment.Id = int32(*id)
	}
	assignment.Name = formFields["name"]
	assignment.Type = formFields["type"]
	return &assignment, nil
}

func (r AssignmentResource) CreateRow(assignment *database.AssignmentRow) (int, error) {
	return database.CreateAssignment(assignment.Name, assignment.Type)
}

func (r AssignmentResource) UpdateRow(assignment *database.AssignmentRow) error {
	return database.UpdateAssignment(assignment.Id, assignment.Name, assignment.Type)
}

func (r AssignmentResource) FormConfig() FormConfig[database.AssignmentRow] {
	return FormConfig[database.AssignmentRow]{
		SaveUrl: func(row *database.AssignmentRow) string {
			if row == nil || row.Id == 0 {
				return "/assignments"
			} else {
				return fmt.Sprintf("/assignments/%d", row.Id)
			}
		},
		Fields: [](FormField[database.AssignmentRow]){
			&components.TextFormFieldConfig[database.AssignmentRow]{
				Label:       "Name",
				FieldName:   "name",
				Placeholder: "Enter a name",
				Required:    true,
				Value:       func(row *database.AssignmentRow) string { return row.Name },
			},
			&components.SelectFormFieldConfig[database.AssignmentRow]{
				Label:       "Type",
				FieldName:   "type",
				Placeholder: "Choose a type",
				Options: []struct{ Label, Value string }{
					{
						Label: "Sound",
						Value: "sound",
					},
					{
						Label: "Text",
						Value: "text",
					},
				},
				Required: true,
				Value:    func(row *database.AssignmentRow) string { return row.Type },
			},
		},
	}
}

func (r AssignmentResource) Location(row *database.AssignmentRow) string {
	if row == nil || row.Id == 0 {
		return "/assignments"
	} else {
		return fmt.Sprintf("/assignments/%d", row.Id)
	}
}

func (r AssignmentResource) TableConfig() [](ColumnConfig[database.AssignmentRow]) {
	return [](ColumnConfig[database.AssignmentRow]){
		{Name: "Id", Value: func(user *database.AssignmentRow) string { return strconv.Itoa(int(user.Id)) }},
		{Name: "Name", Value: func(user *database.AssignmentRow) string { return user.Name }},
		{Name: "Type", Value: func(user *database.AssignmentRow) string { return user.Type }},
	}
}
