-- +goose Up
-- +goose StatementBegin
create view displayable_users as
SELECT 
  id, name, email, date_of_birth
FROM users;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop view if exists displayable_users;
-- +goose StatementEnd
