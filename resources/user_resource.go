package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Kavantix/go-form/database"
	. "github.com/Kavantix/go-form/interfaces"
	"github.com/Kavantix/go-form/newdatabase"
	"github.com/Kavantix/go-form/templates/components"
	age "github.com/bearbin/go-age"
)

type userResource struct {
	queries *newdatabase.Queries
}

func NewUserResource(queries *newdatabase.Queries) Resource[newdatabase.DisplayableUser] {
	return userResource{
		queries: queries,
	}
}

func (r userResource) Title() string {
	return "Users"
}

func (r userResource) FetchPage(ctx context.Context, page, pageSize int) ([]newdatabase.DisplayableUser, error) {
	return r.queries.GetUsersPage(ctx, page, pageSize)
}

func (r userResource) FetchRow(ctx context.Context, id int32) (*newdatabase.DisplayableUser, error) {
	user, err := r.queries.GetUser(ctx, id)
	if err != nil {
		return nil, err
	} else {
		return &user, nil
	}
}

func (r userResource) ParseRow(ctx context.Context, id *int, formFields map[string]string) (*newdatabase.DisplayableUser, error) {
	var err error
	user := newdatabase.DisplayableUser{}
	if id != nil {
		user.Id = int32(*id)
	}
	user.Name = formFields["name"]
	user.Email = formFields["email"]
	emailExists, err := r.queries.UserWithEmailExists(ctx, user.Email, user.Id)
	if err != nil {
		return &user, fmt.Errorf("failed to check email for duplicates: %w", err)
	}
	if emailExists {
		return &user, ValidationError{
			FieldName: "email",
			Reason:    database.ErrDuplicateEmail,
			Message:   "Email already used",
		}
	}
	user.DateOfBirth, err = time.Parse("2006-01-02", formFields["date_of_birth"])
	if err != nil {
		return &user, ParsingError{
			FieldName: "date_of_birth",
			Reason:    err,
			Message:   "Invalid date",
		}
	}
	if age.Age(user.DateOfBirth) < 18 {
		return &user, ValidationError{
			FieldName: "date_of_birth",
			Reason:    errors.New("age below 18"),
			Message:   "Minimum age is 18",
		}
	}
	return &user, nil
}

func (r userResource) CreateRow(ctx context.Context, user *newdatabase.DisplayableUser) (int32, error) {
	return r.queries.InsertUser(ctx, user.Name, user.Email, user.DateOfBirth)
}

func (r userResource) UpdateRow(ctx context.Context, user *newdatabase.DisplayableUser) error {
	return r.queries.UpdateUser(ctx, newdatabase.UpdateUserParams{
		Id:          user.Id,
		Name:        user.Name,
		Email:       user.Email,
		DateOfBirth: user.DateOfBirth,
	})
}

func (r userResource) FormConfig() FormConfig[newdatabase.DisplayableUser] {
	return FormConfig[newdatabase.DisplayableUser]{
		SaveUrl: func(row *newdatabase.DisplayableUser) string {
			if row == nil || row.Id == 0 {
				return "/users"
			} else {
				return fmt.Sprintf("/users/%d", row.Id)
			}
		},
		Fields: [](FormField[newdatabase.DisplayableUser]){
			&components.TextFormFieldConfig[newdatabase.DisplayableUser]{
				FieldLabel:  "Name",
				FieldName:   "name",
				Placeholder: "Enter a name",
				FieldValue:  func(row *newdatabase.DisplayableUser) string { return row.Name },
				Required:    true,
			},
			&components.TextFormFieldConfig[newdatabase.DisplayableUser]{
				FieldLabel:  "Email",
				FieldName:   "email",
				Placeholder: "Enter an email",
				Type:        "email",
				FieldValue:  func(row *newdatabase.DisplayableUser) string { return row.Email },
				Required:    true,
			},
			&components.TextFormFieldConfig[newdatabase.DisplayableUser]{
				FieldLabel:  "Birthdate",
				FieldName:   "date_of_birth",
				Placeholder: "Enter the date of birth",
				Type:        "date",
				FieldValue:  func(row *newdatabase.DisplayableUser) string { return row.DateOfBirth.Format("2006-01-02") },
				Required:    true,
			},
		},
	}
}

func (r userResource) Location(row *newdatabase.DisplayableUser) string {
	if row == nil || row.Id == 0 {
		return "/users"
	} else {
		return fmt.Sprintf("/users/%d", row.Id)
	}
}

func (r userResource) TableConfig() [](ColumnConfig[newdatabase.DisplayableUser]) {
	return [](ColumnConfig[newdatabase.DisplayableUser]){
		{Name: "Id", Value: func(user *newdatabase.DisplayableUser) string { return strconv.Itoa(int(user.Id)) }},
		{Name: "Name", Value: func(user *newdatabase.DisplayableUser) string { return user.Name }},
		{Name: "Email", Value: func(user *newdatabase.DisplayableUser) string { return user.Email }},
		{Name: "Age", Value: func(user *newdatabase.DisplayableUser) string {
			return fmt.Sprintf("%d years", age.Age(user.DateOfBirth))
		}},
	}
}
