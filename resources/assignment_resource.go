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
	queries *database.Queries
}

func NewAssignmentResource(queries *database.Queries) Resource[database.Assignment] {
	return assignmentResource{
		queries: queries,
	}
}

func (r assignmentResource) Title() string {
	return "Assignments"
}

func (r assignmentResource) FetchPage(ctx context.Context, page, pageSize int) ([]database.Assignment, error) {
	return r.queries.GetAssignmentsPage(ctx, page, pageSize)
}

func (r assignmentResource) FetchRow(ctx context.Context, id int32) (*database.Assignment, error) {
	assignment, err := r.queries.GetAssignment(ctx, id)
	if err != nil {
		return nil, err
	} else {
		return &assignment, nil
	}
}

func (r assignmentResource) ParseRow(ctx context.Context, id *int, formFields map[string]string) (*database.Assignment, error) {
	assignment := database.Assignment{}
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

func (r assignmentResource) CreateRow(ctx context.Context, assignment *database.Assignment) (int32, error) {
	return r.queries.InsertAssignment(ctx, assignment.Name, assignment.Type)
}

func (r assignmentResource) UpdateRow(ctx context.Context, assignment *database.Assignment) error {
	return r.queries.UpdateAssignment(ctx, database.UpdateAssignmentParams{
		Id:    assignment.Id,
		Name:  assignment.Name,
		Type:  assignment.Type,
		Order: assignment.Order,
	})
}

func (r assignmentResource) FormConfig() FormConfig[database.Assignment] {
	return FormConfig[database.Assignment]{
		SaveUrl: func(row *database.Assignment) string {
			if row == nil || row.Id == 0 {
				return "/assignments"
			} else {
				return fmt.Sprintf("/assignments/%d", row.Id)
			}
		},
		Fields: [](FormField[database.Assignment]){
			&components.TextFormFieldConfig[database.Assignment]{
				FieldLabel:  "Name",
				FieldName:   "name",
				Placeholder: "Enter a name",
				Required:    true,
				FieldValue:  func(row *database.Assignment) string { return row.Name },
			},
			&components.SelectFormFieldConfig[database.Assignment]{
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
				FieldValue: func(row *database.Assignment) string { return row.Type },
			},
		},
	}
}

func (r assignmentResource) Location(row *database.Assignment) string {
	if row == nil || row.Id == 0 {
		return "/assignments"
	} else {
		return fmt.Sprintf("/assignments/%d", row.Id)
	}
}

func (r assignmentResource) TableConfig() [](ColumnConfig[database.Assignment]) {
	return [](ColumnConfig[database.Assignment]){
		{Name: "Id", Value: func(user *database.Assignment) string { return strconv.Itoa(int(user.Id)) }},
		{Name: "Name", Value: func(user *database.Assignment) string { return user.Name }},
		{Name: "Type", Value: func(user *database.Assignment) string { return user.Type }},
		{Name: "Order", Value: func(row *database.Assignment) string { return strconv.Itoa(int(row.Order)) }},
	}
}
