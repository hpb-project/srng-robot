package routers

import (
	"github.com/astaxie/beego"
	"github.com/hpb-project/srng-robot/controllers"
	"github.com/hpb-project/srng-robot/db"
)

func Init(ldb *db.LevelDB) {
	ctl := controllers.NewController(ldb)
	ns := beego.NewNamespace("/robot",
		beego.NSNamespace("api",
			beego.NSRouter("/getseed", ctl, "get:GetSeed"),
			//beego.NSRouter("/reveal", ctl, "post:ConsumedOneDay"),
		),
	)
	beego.AddNamespace(ns)
}

