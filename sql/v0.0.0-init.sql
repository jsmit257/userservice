create table users (
  id varchar(*) not null primary key
  name varchar not null unique,
  password varchar not null,
  salt varchar(*) not null,
  mtime datetime not null,
  ctime datetime not null default current_timestamp
)
