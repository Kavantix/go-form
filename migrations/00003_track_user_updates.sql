-- +goose Up
-- +goose StatementBegin
alter table users
  add updated_at timestamp default now() not null;
CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE PROCEDURE updated_at_trigger();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table users
  drop column updated_at;
delete trigger if exists users_updated_at;
-- +goose StatementEnd
