-- name: getUsersPage :many
select 
  *
from displayable_users
order by id
limit $1 offset $2;

-- name: GetUser :one
select 
  *
from displayable_users
where id = @id
limit 1;

-- name: GetUserByEmail :one
select 
  *
from displayable_users
where email = @email
limit 1;


-- name: UserWithEmailExists :one
select exists(
  select
  from users
  where email = $1
    and id != sqlc.arg(excluding_id)
);

-- name: insertUser :one
insert into users(
  name,
  email,
  date_of_birth
) values ($1, $2, $3) returning id;

-- name: updateUser :exec
update users set
  name=$2,
  email=$3,
  date_of_birth=$4
where id = $1;

-- name: InsertReloginToken :one
insert into relogin_tokens (
  user_id,
  token
) values ($1, $2) returning id;

-- name: consumeReloginToken :execrows
delete from relogin_tokens
where token = $1
  and user_id = $2
  and created_at > sqlc.arg(created_after);
