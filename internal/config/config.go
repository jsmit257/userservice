package config

import (
	"encoding/json"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	MySQLHost string `envconfig:"MYSQL_HOST" required:"true" json:"mysql_host"`
	MySQLPort uint16 `envconfig:"MYSQL_PORT" required:"true" json:"mysql_port"`
	MySQLPwd  string `envconfig:"MYSQL_PASSWORD" required:"true" json:"-"`
	MySQLUser string `envconfig:"MYSQL_USER" required:"true" json:"mysql_user"`

	RedisUser string `envconfig:"REDIS_USER" json:"redis_user"`
	RedisPass string `envconfig:"REDIS_PASS" json:"-"`
	RedisHost string `envconfig:"REDIS_HOST" default:"redis" json:"redis_host"`
	RedisPort int16  `envconfig:"REDIS_PORT" default:"6379" json:"redis_port"`

	MaildHost   string `envconfig:"MAILD_HOST" default:"mail.google.com" json:"maild_host,omitempty"`
	MaildPort   uint16 `envconfig:"MAILD_PORT" default:"587" json:"maild_port,omitempty"`
	MaildUser   string `envconfig:"MAILD_USER" default:"svc" json:"maild_user,omitempty"`
	MaildPass   string `envconfig:"MAILD_PASS" default:"snakeoil" json:"-"`
	MaildSender string `envconfig:"MAILD_SENDER" default:"no-reply@cffc.io" json:"maild_sender"`

	SmsAccountID string `envconfig:"SMS_ACCT_ID" json:"sms_acct_id,omitempty"`
	SmsAuthToken string `envconfig:"SMS_AUTH_TOKEN" json:"sms_auth_token,omitempty"`
	SmsSender    string `envconfig:"SMS_SENDER" json:"sms_sender,omitempty"`

	AuthnTimeout int64  `envconfig:"AUTHN_TIMEOUT" default:"15" json:"authn_timeout"`
	MaxLogins    int    `envconfig:"MAX_LOGINS" default:"5" json:"max_logins"`
	CookieName   string `envconfig:"AUTHN_COOKIE" default:"us-authn" json:"authn_cookie"`

	LogonURL   string `envconfig:"LOGIN_URL" default:"/authnz/login.html" json:"logon_url"`
	ResetURL   string `envconfig:"RESET_URL" default:"/authnz/login.html?reset" json:"reset_url"`
	SuccessURL string `envconfig:"REDIR_SUCCESS" default:"/" json:"success_url"`

	ServerHost string `envconfig:"HTTP_HOST" default:"0.0.0.0" json:"server_host"`
	ServerPort uint16 `envconfig:"HTTP_PORT" default:"3000" json:"server_port"`

	LogLevel      string `envconfig:"LOG_LEVEL" default:"INFO" json:"min_log_level"`
	EmailTestMode bool   `envconfig:"EMAIL_TEST_MODE" default:"false" json:"email_test_mode"`
	SmsTestMode   bool   `envconfig:"SMS_TEST_MODE" default:"false" json:"sms_test_mode"`
}

func NewConfig() *Config {
	result := &Config{}
	err := envconfig.Process("US", result)
	if err != nil {
		panic(err)
	}
	return result
}

func (c *Config) JSON() string {
	result, _ := json.Marshal(c)
	return string(result)
}
