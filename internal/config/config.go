package config

import (
	"github.com/spf13/viper"
)

type User struct {
	AccessKey string `mapstructure:"access_key"`
	Secret    string
}

type Pubproxy struct {
	UserMap    map[string]User `mapstructure:"user_map"`
	ServerAddr string          `mapstructure:"server_addr"`
	InputAddr  string          `mapstructure:"input_addr"`
}

type Locproxy struct {
	User          *User
	Forward       string
	ServerAddr    string `mapstructure:"server_addr"`
	RegisterRoute string `mapstructure:"register_route"`
}

type Config struct {
	Pubproxy *Pubproxy
	Locproxy *Locproxy
}

var config = &Config{}

func GetConfig() (cnf *Config) {
	return config
}

func ScanConfig() {
	viper.Unmarshal(&config)
}
