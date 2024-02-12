-- name: GetUsersPage :many
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

-- name: InsertUser :one
insert into users(
  name,
  email,
  date_of_birth
) values ($1, $2, $3) returning id;

-- name: UpdateUser :exec
update users set
  name=$1,
  email=$2,
  date_of_birth=$3
where id = $4;
