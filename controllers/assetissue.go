package controllers

import (
	"github.com/sasaxie/go-client-api/models"

	"github.com/astaxie/beego"
)

// Operations about asset issue
type AssetIssueController struct {
	beego.Controller
}

// @Title Get asset issue list
// @Description get asset issue list by account
// @Param	address		path 	string	true
// @Success 200 {assetissuelist} models.AssetIssueList
// @Failure 403 :address is empty
// @router /:address [get]
func (i *AssetIssueController) Address() {
	address := i.GetString(":address")
	if address != "" {
		assetIssueList := models.GetAssetIssueAccount(address)
		i.Data["json"] = assetIssueList
	}
	i.ServeJSON()
}
