-- +goose Up
create table gauge (
	name text not null primary key,
	value double precision
);

create table counter (
	name text not null primary key,
	value int
);

-- +goose Down
drop table gauge;
drop table counter;
