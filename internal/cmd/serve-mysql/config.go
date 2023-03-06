package main

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	MySQLHost    string `envconfig:"MYSQL_HOST" required:"true"`
	MySQLPort    uint16 `envconfig:"MYSQL_PORT" required:"true"`
	MySQLRootPwd string `envconfig:"MYSQL_ROOT_PASSWORD" required:"true" json:"-"`
	MySQLUser    string `envconfig:"MYSQL_USER" required:"true"`
	LogLevel     string `envconfig:"LOG_LEVEL" default:"INFO"`
}

func NewConfig() *Config {
	result := &Config{}
	err := envconfig.Process("US", result)
	if err != nil {
		panic(err)
	}
	return result
}
