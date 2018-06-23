package controllers

import (
	"github.com/sasaxie/go-client-api/models"

	"github.com/astaxie/beego"
)

// Operations about Transaction
type TransactionController struct {
	beego.Controller
}

// @Title Get transaction by id
// @Description Get transaction by id
// @Param	id		path 	string	true
// @Success 200 {transaction} models.Transaction
// @router /id/:id [get]
func (b *TransactionController) Id() {
	id := b.GetString(":id")
	if id != "" {
		transaction := models.GetTransactionById(id)
		b.Data["json"] = transaction
	}
	b.ServeJSON()
}
