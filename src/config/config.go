package config

import "gopkg.in/gcfg.v1"

// config 配置文件结构
type config struct {
	API struct {
		Mode string
		Addr string
	}
	Mysql struct {
		Host     string
		Port     int
		Username string
		Password string
		Database string
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
	}
	Elastic struct {
		Addr     string
		Index    string
		Username string
		Password string
	}
	Kimi struct {
		Key    string
		Prompt string
	}
}

var Cfg *config

func init() {
	Cfg = &config{}
	err := gcfg.ReadFileInto(Cfg, "config.toml")
	if err != nil {
		panic(err)
	}
}
