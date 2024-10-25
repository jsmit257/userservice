insert 
  into users(
         id, 
         name, 
         password, 
         salt,
         mtime,
         ctime,
         dtime,
         login_success,
         login_failure,
         failure_count
      )
values ("00000000-0000-0000-0000-000000000001", "testuser1", "bogus", "salt", "0001-01-01", "0001-01-01", null, null, null, 0),
       ("00000000-0000-0000-0000-000000000002", "testuser2", "bogus", "salt", "0001-01-01", "0001-01-01", null, null, null, 0),
       ("00000000-0000-0000-0000-000000000003", "testuser3", "bogus", "salt", "0001-01-01", "0001-01-01", null, null, null, 0);
       