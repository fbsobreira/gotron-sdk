package controllers

import (
	"github.com/sasaxie/go-client-api/models"

	"github.com/astaxie/beego"
)

// Operations about Node
type NodeController struct {
	beego.Controller
}

// @Title Get node list
// @Description get node list
// @Success 200 {nodeList} []models.Node
// @router /list [get]
func (n *NodeController) List() {
	nodes := models.GetNodeList()
	n.Data["json"] = nodes
	n.ServeJSON()
}
