package main

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	MySQLHost    string `envconfig:"MYSQL_HOST" required:"true" json:"mysql_host"`
	MySQLPort    uint16 `envconfig:"MYSQL_PORT" required:"true" json:"mysql_port"`
	MySQLRootPwd string `envconfig:"MYSQL_ROOT_PASSWORD" required:"true" json:"-"`
	MySQLUser    string `envconfig:"MYSQL_USER" required:"true" json:"mysql_user"`

	ServerHost string `envconfig:"SERVER_HOST" default:"0.0.0.0" json:"server_host"`
	ServerPort uint16 `envconfig:"SERVER_PORT" default:"3000" json:"server_port"`

	LogLevel string `envconfig:"LOG_LEVEL" default:"INFO" json:"min_log_level"`
}

func NewConfig() *Config {
	result := &Config{}
	err := envconfig.Process("US", result)
	if err != nil {
		panic(err)
	}
	return result
}
