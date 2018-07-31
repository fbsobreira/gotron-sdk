package controllers

import (
	"github.com/sasaxie/go-client-api/models"

	"encoding/json"
	"github.com/astaxie/beego"
	"github.com/sasaxie/go-client-api/models/contract"
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
// @router /address/:address [get]
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

// @Title Get account net message
// @Description get account net message by address
// @Param	address		path 	string	true
// @Success 200 {accountnetmessage} models.AccountNetMessage
// @Failure 403 :address is empty
// @router /net-message/address/:address [get]
func (a *AccountController) NetMessage() {
	address := a.GetString(":address")
	if address != "" {
		accountNetMessage := models.GetAccountNet(address)
		a.Data["json"] = accountNetMessage
	}
	a.ServeJSON()
}

// @Title Create account
// @Param owneraddress body string true
// @Param accountaddress body string true
// @router /create [post]
func (a *AccountController) CreateAccount() {
	var accountCreateContract contract.AccountCreateContract
	err := json.Unmarshal(a.Ctx.Input.RequestBody, &accountCreateContract)

	if err != nil {
		a.Data["json"] = err.Error()
	} else {
		transaction, err := contract.CreateAccount(accountCreateContract)

		if err != nil {
			a.Data["json"] = err.Error()
		} else {
			a.Data["json"] = transaction
		}
	}

	a.ServeJSON()
}
