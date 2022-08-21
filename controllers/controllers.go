package controllers

import (
	"encoding/hex"
	"github.com/astaxie/beego"
	"github.com/hpb-project/srng-robot/db"
)

type Controller struct {
	beego.Controller
	ldb *db.LevelDB
}

func NewController(ldb *db.LevelDB) *Controller {
	c := &Controller{}
	c.ldb = ldb
	return c
}

func (d *Controller) ResponseInfo(code int, errMsg interface{}, result interface{}) {
	switch code {
	case 500:
		d.Data["json"] = map[string]interface{}{"error": "500", "err_msg": errMsg, "data": result}
	case 200:
		d.Data["json"] = map[string]interface{}{"error": "200", "err_msg": errMsg, "data": result}
	}
	d.ServeJSON()
}

func (d *Controller) GetSeed() {
	param := d.Ctx.Input.Query("hash")
	hash,_ := hex.DecodeString(param)

	value,exist := db.GetSeedBySeedHash(d.ldb, hash)
	if exist {
		d.ResponseInfo(200, "ok", hex.EncodeToString(value))
	} else {
		d.ResponseInfo(500, "not found seed", nil)
	}
}



