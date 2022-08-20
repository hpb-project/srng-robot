package main

import (
	"github.com/hpb-project/srng-robot/config"
)

func main() {
	robot := NewRobot(config.GetConfig())
	robot.Start()
}
