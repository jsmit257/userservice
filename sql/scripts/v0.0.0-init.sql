create table users (
  id varchar(1) not null primary key,
  name varchar(1) not null unique,
  password varchar(1) not null,
  salt varchar(1) not null,
  mtime datetime not null default current_timestamp,
  ctime datetime not null default current_timestamp
)
