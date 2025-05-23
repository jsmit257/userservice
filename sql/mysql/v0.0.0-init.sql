start transaction;

set @dbexists=(
   select  1 
     from  information_schema.schemata 
    where  schema_name = 'userservice'
);

create database if not exists userservice;

use userservice;

create table if not exists users(
  uuid          varchar(36)   not null primary key,
  name          varchar(128)  not null unique,
  email         varchar(256)  null,
  cell          varchar(20)   null, -- a phone number for txts and calls
  password      varchar(128)  not null,
  salt          char(4)       not null,
  loginsuccess  datetime      null     default current_timestamp,
  loginfailure  datetime      null,
  failurecount  int unsigned  not null default 0,
  locked        datetime      null,
  mtime         datetime      not null default current_timestamp,
  ctime         datetime      not null default current_timestamp,
  dtime         datetime      null,
  check         (1 or email is not null or cell is not null)
) engine=InnoDB;

create table if not exists  addresses(
  uuid     varchar(36)   not null primary key,
  street1  varchar(128)  not null,
  street2  varchar(128)  null,
  city     varchar(64)   not null,
  state    varchar(32)   not null,
  country  varchar(128)  not null,
  zip      varchar(10)   null,
  mtime    datetime      not null default current_timestamp,
  ctime    datetime      not null default current_timestamp
) engine=InnoDB;

create table if not exists contacts(
  uuid         varchar(36)   not null primary key,
  firstname    varchar(128)  not null,
  lastname     varchar(128)  not null,
  billto_uuid  varchar(36)   null,
  shipto_uuid  varchar(36)   null,
  phone        varchar(20)   null, -- don't send text messages here
  mtime        datetime      not null default current_timestamp,
  ctime        datetime      not null default current_timestamp,
  foreign key (uuid) references users(uuid),
  foreign key (billto_uuid) references addresses(uuid),
  foreign key (shipto_uuid) references addresses(uuid)
) engine=InnoDB;

set @stmt=(select if(@dbexists is not null, 'ROLLBACK', 'COMMIT'));

prepare execution from @stmt;

execute execution;
