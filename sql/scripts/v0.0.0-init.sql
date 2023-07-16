create table users (
  id varchar(36) not null primary key,
  name varchar(128) not null unique,
  password char(15) not null,
  salt char(4) not null,
  mtime datetime not null default current_timestamp,
  ctime datetime not null default current_timestamp,
  dtime datetime default null
)
;

insert 
  into users(
         id, 
         name, 
         password, 
         salt,
         mtime,
         ctime
      )
values ("00000000-0000-0000-0000-000000000001", "testuser1", "bogus", "salt", "0001-01-01", "0001-01-01"),
       ("00000000-0000-0000-0000-000000000002", "testuser2", "bogus", "salt", "0001-01-01", "0001-01-01"),
       ("00000000-0000-0000-0000-000000000003", "testuser3", "bogus", "salt", "0001-01-01", "0001-01-01")