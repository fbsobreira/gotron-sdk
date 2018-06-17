package controllers

import (
	"github.com/sasaxie/go-client-api/models"

	"github.com/astaxie/beego"
)

// Operations about Block
type BlockController struct {
	beego.Controller
}

// @Title Get now block
// @Description get now block account
// @Success 200 {block} models.Block
// @router /now [get]
func (b *BlockController) Now() {
	nowBlock := models.GetNowBlock()
	b.Data["json"] = nowBlock
	b.ServeJSON()
}
