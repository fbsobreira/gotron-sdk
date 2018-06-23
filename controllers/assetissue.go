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
// @router /address/:address [get]
func (i *AssetIssueController) Address() {
	address := i.GetString(":address")
	if address != "" {
		assetIssueList := models.GetAssetIssueAccount(address)
		i.Data["json"] = assetIssueList
	}
	i.ServeJSON()
}

// @Title Get asset issue by name
// @Description get asset issue by name
// @Param	name		path 	string	true
// @Success 200 {assetissue} models.AssetIssueContract
// @Failure 403 :name is empty
// @router /name/:name [get]
func (i *AssetIssueController) Name() {
	name := i.GetString(":name")
	if name != "" {
		assetIssue := models.GetAssetIssueByName(name)
		i.Data["json"] = assetIssue
	}
	i.ServeJSON()
}

// @Title Get asset issue list
// @Description get asset issue list
// @Success 200 {assetissuelist} models.AssetIssueList
// @router /list [get]
func (i *AssetIssueController) List() {
	assetIssueList := models.GetAssetIssueList()
	i.Data["json"] = assetIssueList
	i.ServeJSON()
}
