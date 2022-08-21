package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/hpb-project/srng-robot/config"
)

func main() {
	logs.Info("srng robot start")
	robot := NewRobot(config.GetConfig())
	robot.Start()

}
