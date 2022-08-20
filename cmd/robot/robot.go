package main

import (
	"fmt"
	"github.com/hpb-project/srng-robot/config"
	"github.com/hpb-project/srng-robot/db"
	"github.com/hpb-project/srng-robot/services/product"
	"github.com/hpb-project/srng-robot/services/pullevent"
)

type Robot struct {
	ldb *db.LevelDB
	config config.Config

	pe *pullevent.PullEvent
	ps *product.ProductService
}

func NewRobot(config config.Config) *Robot {
	ldb := db.NewLevelDB(config.DBPath)
	if ldb == nil {
		panic("db create failed")
	}

	pe := pullevent.NewPullEvent(config, ldb)
	if pe == nil {
		panic("create pull event servicce failed")
	}

	ps,err := product.NewProductService(config, ldb)
	if err != nil {
		panic(fmt.Sprintf("new product service failed with error (%s)",err ))
	}

	return &Robot{
		ldb: ldb,
		config: config,
		ps: ps,
	}
}

func (r *Robot) Start() {
	go r.pe.GetLogs()

	r.ps.Run()



	run := make(chan struct{})
	<-run
}

func (r *Robot) Stop() {

}