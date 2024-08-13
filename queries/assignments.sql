-- name: GetAssignment :one
select
  *
from assignments
where id = $1;

-- name: getAssignmentsPage :many
select
  *
from assignments
limit $1 offset $2;

-- name: InsertAssignment :one
with max_order as (
  select case 
    when max("order") is null then 0
    else max("order")
  end as "order"
  from assignments
) insert into assignments(
  name,
  "type",
  "order"
) values ($1, $2, (select "order" + 1 from max_order)) returning id;


-- name: UpdateAssignment :exec
update assignments set
  name = coalesce(sqlc.narg('name'), name),
  "type" = coalesce(sqlc.narg('type'), "type"),
  "order" = coalesce(cast(sqlc.narg('order') as int4), "order")
where id = $1;
