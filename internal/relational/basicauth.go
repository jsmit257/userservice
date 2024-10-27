package data

// func (db *Conn) BasicAuth(ctx context.Context, login *sharedv1.BasicAuth, cid sharedv1.CID) (*sharedv1.User, error) {
// 	m := mtrcs.MustCurryWith(prometheus.Labels{"method": "BasicAuth"})

// 	var id sharedv1.UUID
// 	var pass, salt string
// 	var loginSuccess, loginFailure *time.Time
// 	var failureCount uint8

// 	err := db.QueryRowContext(ctx, selectBasicAuth, login.Name).Scan(
// 		&id,
// 		&pass,
// 		&salt,
// 		&loginSuccess,
// 		&loginFailure,
// 		&failureCount)
// 	if err != nil {
// 		return nil, trackError(cid, db.logger, m, err, err.Error()) // FIXME: fmt.Errorf("internal server error")
// 	} else if id == "" {
// 		return nil, trackError(cid, db.logger, m, fmt.Errorf("bad username or password"), "bad_username")
// 	} else if failureCount > maxFailure {
// 		return nil, trackError(cid, db.logger, m, fmt.Errorf("too many failed login attempts"), "password_lockout")
// 	}

// 	now := time.Now().UTC()
// 	if hash(login.Pass, salt) != pass {
// 		if err := db.updateBasicAuth(ctx, id, loginSuccess, &now, failureCount+1); err != nil {
// 			return nil, trackError(cid, db.logger, m, err, err.Error()) // fmt.Errorf("internal server error"), err.Error())
// 		}
// 		return nil, trackError(cid, db.logger, m, fmt.Errorf("bad username or password"), "bad_password")
// 	}

// 	loginFailure = nil
// 	if err := db.updateBasicAuth(ctx, id, &now, nil, 0); err != nil {
// 		return nil, err
// 	}

// 	user, err := db.GetUser(ctx, id, cid)
// 	if err == nil {
// 		// for all intents and purposes, BasicAuth is successful (i.e "err":"none" in the metric) here,
// 		// GetUser may still fail with a separate metric; the context is the key to correlating the
// 		// nested calls, but i haven't figured that part out yet
// 		m.WithLabelValues("none").Inc()
// 	}

// 	// when authing, the returned user has the login_success from the previous successful logged-in session,
// 	// not this one, for anyone who actually pays attention
// 	user.LoginSuccess = loginSuccess
// 	return user, err
// }

// func (db *Conn) updateBasicAuth(ctx context.Context, id sharedv1.UUID, loginSuccess, loginFailure *time.Time, failureCount uint8) error {
// 	var err error
// 	var result sql.Result
// 	var updateCount int64

// 	result, err = db.ExecContext(ctx, updateBasicAuth, loginSuccess, loginFailure, failureCount, id)
// 	if err == nil {
// 		updateCount, err = result.RowsAffected()
// 		if err == nil && updateCount != 1 {
// 			return fmt.Errorf("basicAuth not updated")
// 		}
// 	}
// 	return err
// }
