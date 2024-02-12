-- +goose Up
-- +goose StatementBegin
create table if not exists users (
  id serial primary key,
  email varchar(255) unique not null,
  created_at timestamp default now() not null,
  date_of_birth date not null,
  name varchar(255) not null
);

insert into users(email, name, date_of_birth) values('mail@pietervanloon.com', 'Pieter', '1994-12-14');
-- +goose StatementEnd
--
-- +goose Down
-- +goose StatementBegin
drop table if exists users;
-- +goose StatementEnd
