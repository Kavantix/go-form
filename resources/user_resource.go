package resources

import (
	"fmt"
	"strconv"

	"github.com/Kavantix/go-form/database"
	. "github.com/Kavantix/go-form/interfaces"
	"github.com/Kavantix/go-form/templates/components"
	age "github.com/bearbin/go-age"
)

type UserResource struct {
}

func (r UserResource) Title() string {
	return "Users"
}

func (r UserResource) FormConfig() FormConfig[database.UserRow] {
	return FormConfig[database.UserRow]{
		SaveUrl: func(row *database.UserRow) string {
			if row == nil || row.Id == 0 {
				return "/users"
			} else {
				return fmt.Sprintf("/users/%d", row.Id)
			}
		},
		Fields: [](FormField[database.UserRow]){
			&components.TextFormFieldConfig[database.UserRow]{
				Label:       "Name",
				FieldName:   "name",
				Placeholder: "Enter a name",
				Value:       func(row *database.UserRow) string { return row.Name },
				Required:    true,
			},
			&components.TextFormFieldConfig[database.UserRow]{
				Label:       "Email",
				FieldName:   "email",
				Placeholder: "Enter an email",
				Type:        "email",
				Value:       func(row *database.UserRow) string { return row.Email },
				Required:    true,
			},
			&components.TextFormFieldConfig[database.UserRow]{
				Label:       "Birthdate",
				FieldName:   "date_of_birth",
				Placeholder: "Enter the date of birth",
				Type:        "date",
				Value:       func(row *database.UserRow) string { return row.DateOfBirth.Format("2006-01-02") },
				Required:    true,
			},
		},
	}
}

func (r UserResource) Location(row *database.UserRow) string {
	return fmt.Sprintf("/users/%d", row.Id)
}

func (r UserResource) TableConfig() [](ColumnConfig[database.UserRow]) {
	return [](ColumnConfig[database.UserRow]){
		{Name: "Id", Value: func(user *database.UserRow) string { return strconv.Itoa(int(user.Id)) }},
		{Name: "Name", Value: func(user *database.UserRow) string { return user.Name }},
		{Name: "Email", Value: func(user *database.UserRow) string { return user.Email }},
		{Name: "Age", Value: func(user *database.UserRow) string { return fmt.Sprintf("%d years", age.Age(user.DateOfBirth)) }},
	}
}
