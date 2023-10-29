-- +goose Up
-- +goose StatementBegin
create table if not exists users (
  id serial primary key,
  email varchar(255) unique not null,
  created_at timestamp default now() not null,
  date_of_birth date,
  name varchar(255)
);
-- +goose StatementEnd
--
-- +goose Down
-- +goose StatementBegin
drop table if exists users;
-- +goose StatementEnd
