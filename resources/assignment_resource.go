package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Kavantix/go-form/database"
	. "github.com/Kavantix/go-form/interfaces"
	"github.com/Kavantix/go-form/templates/components"
)

type assignmentResource struct {
}

func NewAssignmentResource() Resource[database.AssignmentRow] {
	return assignmentResource{}
}

func (r assignmentResource) Title() string {
	return "Assignments"
}

func (r assignmentResource) FetchPage(ctx context.Context, page, pageSize int) ([]database.AssignmentRow, error) {
	return database.GetAssignments(page, pageSize)
}

func (r assignmentResource) FetchRow(ctx context.Context, id int32) (*database.AssignmentRow, error) {
	return database.GetAssignment(id)
}

func (r assignmentResource) ParseRow(ctx context.Context, id *int, formFields map[string]string) (*database.AssignmentRow, error) {
	assignment := database.AssignmentRow{}
	if id != nil {
		assignment.Id = int32(*id)
	}
	assignment.Name = formFields["name"]
	assignment.Type = formFields["type"]
	if assignment.Type == "sound" {
		return &assignment, ValidationError{
			FieldName: "type",
			Reason:    errors.New("unsupported type"),
			Message:   "Sound type is not supported yet",
		}
	}
	return &assignment, nil
}

func (r assignmentResource) CreateRow(ctx context.Context, assignment *database.AssignmentRow) (int32, error) {
	return database.CreateAssignment(assignment.Name, assignment.Type)
}

func (r assignmentResource) UpdateRow(ctx context.Context, assignment *database.AssignmentRow) error {
	return database.UpdateAssignment(assignment.Id, assignment.Name, assignment.Type)
}

func (r assignmentResource) FormConfig() FormConfig[database.AssignmentRow] {
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
				FieldLabel:  "Name",
				FieldName:   "name",
				Placeholder: "Enter a name",
				Required:    true,
				FieldValue:  func(row *database.AssignmentRow) string { return row.Name },
			},
			&components.SelectFormFieldConfig[database.AssignmentRow]{
				FieldLabel:  "Type",
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
				Required:   true,
				FieldValue: func(row *database.AssignmentRow) string { return row.Type },
			},
		},
	}
}

func (r assignmentResource) Location(row *database.AssignmentRow) string {
	if row == nil || row.Id == 0 {
		return "/assignments"
	} else {
		return fmt.Sprintf("/assignments/%d", row.Id)
	}
}

func (r assignmentResource) TableConfig() [](ColumnConfig[database.AssignmentRow]) {
	return [](ColumnConfig[database.AssignmentRow]){
		{Name: "Id", Value: func(user *database.AssignmentRow) string { return strconv.Itoa(int(user.Id)) }},
		{Name: "Name", Value: func(user *database.AssignmentRow) string { return user.Name }},
		{Name: "Type", Value: func(user *database.AssignmentRow) string { return user.Type }},
	}
}
