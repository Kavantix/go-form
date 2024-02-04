-- +goose Up
-- +goose StatementBegin
create table if not exists relogin_tokens (
  id serial primary key,
  token varchar(6),
  user_id serial references users(id),
  created_at timestamp default now() not null
);
create index relogin_tokens_user_id_token on relogin_tokens (user_id, token, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index relogin_tokens_user_id_token;
drop table if exists relogin_tokens;
-- +goose StatementEnd
