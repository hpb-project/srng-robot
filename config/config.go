package config

import "github.com/astaxie/beego"

type Config struct {
	DBPath  string
	Oracle  string
	Token   string
	NodeRPC string
	PrivKey string
	ChainId int
}

var defaultConfig = Config{
	DBPath: "./data/application.db",
}

func GetConfig() Config {
	conf := defaultConfig
	conf.NodeRPC = beego.AppConfig.String("url")
	conf.Oracle = beego.AppConfig.String("oracleAddr")
	conf.Token = beego.AppConfig.String("tokenAddr")
	conf.PrivKey = beego.AppConfig.String("privkey")
	conf.ChainId, _ = beego.AppConfig.Int("chainid")
	return conf
}
