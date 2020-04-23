package controllers

import (
	"github.com/fbsobreira/gotron/models"

	"github.com/astaxie/beego"
)

// Operations about Number Message
type NumberMessageController struct {
	beego.Controller
}

// @Title Get next maintenance time
// @Description Get next maintenance time
// @Success 200 {nextmaintenancetime} models.NumberMessage
// @router /next-maintenance-time [get]
func (n *NumberMessageController) NextMaintenanceTime() {
	nextMaintenanceTime, err := models.GetNextMaintenanceTime()
	if err != nil {
		n.Abort("500")
	}
	n.Data["json"] = nextMaintenanceTime
	n.ServeJSON()
}

// @Title Get total transaction
// @Description Get total transaction
// @Success 200 {totaltransaction} models.NumberMessage
// @router /total-transaction [get]
func (n *NumberMessageController) TotalTransaction() {
	totalTransaction := models.GetTotalTransaction()
	n.Data["json"] = totalTransaction
	n.ServeJSON()
}
