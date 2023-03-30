package config

type Config struct {
	Server *Server
	Mysql  *MysqlConfig
	Redis  *RedisConfig
}

type Server struct {
	Name string `toml:"name"`
	Addr string `toml:"addr"`
	Env  string `toml:"env"`
}

type MysqlConfig struct {
	Name   string `toml:"name"`
	Master string `toml:"master"`
	Slave  string `toml:"slave"`
}

type RedisConfig struct {
	Name     string `toml:"name"`
	Addr     string `json:"addr"`
	PassWord string `json:"pass_word"`
	DataBase int    `json:"data_base"`
}
