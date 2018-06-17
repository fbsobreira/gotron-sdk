package controllers

import (
	"github.com/sasaxie/go-client-api/models"

	"github.com/astaxie/beego"
)

// Operations about Witness
type WitnessController struct {
	beego.Controller
}

// @Title Get witness list
// @Description get witness list
// @Success 200 {witnessList} []models.Witness
// @router /list [get]
func (w *WitnessController) List() {
	witnesses := models.GetWitnessList()
	w.Data["json"] = witnesses
	w.ServeJSON()
}
