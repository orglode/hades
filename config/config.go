package config

import "github.com/BurntSushi/toml"

func InitNavigator(confPath string) *Config {
	cfg := &Config{}
	_, err := toml.DecodeFile(confPath, &cfg)
	if err != nil {
		panic("config.toml is err !!")
	}
	return cfg
}
