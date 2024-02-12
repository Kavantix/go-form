-- +goose Up
-- +goose StatementBegin
create table if not exists assignments (
  id serial primary key,
  name varchar(255) not null,
  created_at timestamp default now() not null,
  updated_at timestamp default now() not null,
  "type" varchar(50) not null
);
CREATE FUNCTION updated_at_trigger() RETURNS trigger
   LANGUAGE plpgsql AS
$$BEGIN
   NEW.updated_at := current_timestamp;
   RETURN NEW;
END;$$;
CREATE TRIGGER assignments_updated_at
    BEFORE UPDATE ON assignments
    FOR EACH ROW
    EXECUTE PROCEDURE updated_at_trigger();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop trigger if exists assignments_updated_at;
drop function if exists updated_at_trigger;
drop table if exists assignments;
-- +goose StatementEnd
