package controllers

import (
	"github.com/sasaxie/go-client-api/models"

	"github.com/astaxie/beego"
)

// Operations about Account
type AccountController struct {
	beego.Controller
}

// @Title Get account
// @Description get account by address
// @Param	address		path 	string	true		"The key for staticblock"
// @Success 200 {account} models.Account
// @Failure 403 :address is empty
// @router /:address [get]
func (a *AccountController) Get() {
	address := a.GetString(":address")
	if address != "" {
		account, err := models.GetAccountByAddress(address)
		if err != nil {
			a.Data["json"] = err.Error()
		} else {
			a.Data["json"] = account
		}
	}
	a.ServeJSON()
}
