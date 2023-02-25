package main

import (
	"fmt"

	"github.com/jsmit257/userservice/internal/data/mysql"
	"github.com/jsmit257/userservice/internal/router"
)

func main() {
	cfg := NewConfig()

	fmt.Printf("%q\n", cfg)

	mysql, err := mysql.NewInstance(cfg.MySQLHost, cfg.MySQLUser, cfg.MySQLRootPwd, cfg.MySQLPort)
	if err != nil {
		panic(err)
	}

	err = router.NewInstance(&router.UserService{
		User:    mysql,
		Address: mysql,
		Contact: mysql,
	})
	if err != nil {
		panic(err)
	}
}
