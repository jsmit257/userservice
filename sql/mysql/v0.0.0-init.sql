create table users (
  uuid          varchar(36)   not null primary key,
  name          varchar(128)  not null unique,
  password      char(15)      not null,
  salt          char(4)       not null,
  loginsuccess  datetime      null,
  loginfailure  datetime      null,
  failurecount  unsigned int  not null default 0,
  mtime         datetime      not null default current_timestamp,
  ctime         datetime      not null default current_timestamp,
  dtime         datetime      null
);

create table addresses (
  uuid     varchar(36)   not null primary key,
  street1  varchar(128)  not null,
  street2  varchar(128)  null,
  city     varchar(64)   not null,
  state    varchar(32)   not null,
  country  varchar(128)  not null,
  zip      varchar(10)   null,
  mtime    datetime      not null,
  ctime    datetime      not null,
);

create table contacts (
  uuid         varchar(36)   not null primary key,
  firstname    varchar(128)  not null,
  lastname     varchar(128)  not null,
  user_uuid    varchar(36)   not null foreign key refrences users(uuid),
  billto_uuid  varchar(36)   null     foreign key refrences addreses(uuid),
  shipto_uuid  varchar(36)   null     foreign key refrences addreses(uuid),
  mtime        datetime      not null,
  ctime        datetime      not null,
  dtime        datetime      null
);