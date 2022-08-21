package main

import (
	"fmt"
	"github.com/hpb-project/srng-robot/config"
	"github.com/hpb-project/srng-robot/db"
	"github.com/hpb-project/srng-robot/services/monitor"
	"github.com/hpb-project/srng-robot/services/pullevent"
)

type Robot struct {
	ldb *db.LevelDB
	config config.Config

	pe *pullevent.PullEvent
	pm *monitor.MonitorService
}

func NewRobot(config config.Config) *Robot {
	robot := new(Robot)

	ldb := db.NewLevelDB(config.DBPath)
	if ldb == nil {
		panic("db create failed")
	}

	pe := pullevent.NewPullEvent(config, ldb, robot)
	if pe == nil {
		panic("create pull event servicce failed")
	}

	pm,err := monitor.NewMonitorService(config, ldb)
	if err != nil {
		panic(fmt.Sprintf("new monitor service failed with error (%s)",err))
	}

	robot.ldb = ldb
	robot.config = config
	robot.pm = pm
	robot.pe = pe

	return robot
}

func (r *Robot) NewCommit() error {
	return r.pm.DoCommit()
}

func (r *Robot) Reveal(commit []byte) error {
	r.pm.DoReveal(commit)
	return nil
}

func (r *Robot) Start() {
	go r.pe.GetLogs()
	go r.pm.Run()

	run := make(chan struct{})
	<-run
}

func (r *Robot) Stop() {

}