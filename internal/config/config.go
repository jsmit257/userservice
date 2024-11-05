package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	MySQLHost string `envconfig:"MYSQL_HOST" required:"true" json:"mysql_host"`
	MySQLPort uint16 `envconfig:"MYSQL_PORT" required:"true" json:"mysql_port"`
	MySQLPwd  string `envconfig:"MYSQL_PASSWORD" required:"true" json:"-"`
	MySQLUser string `envconfig:"MYSQL_USER" required:"true" json:"mysql_user"`

	AuthzTimeout int64  `envconfig:"AUTHZ_TIMEOUT" default:"15" json:"authz_timeout"`
	CookieName   string `envconfig:"AUTHZ_COOKIE" default:"us-authz" json:"authz_cookie"`

	ServerHost string `envconfig:"HTTP_HOST" default:"0.0.0.0" json:"server_host"`
	ServerPort uint16 `envconfig:"HTTP_PORT" default:"3000" json:"server_port"`

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
