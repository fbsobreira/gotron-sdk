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

// @Title Get block by num
// @Description Get block by num
// @Param	num		path 	int64	true
// @Success 200 {block} models.Block
// @router /num/:num [get]
func (b *BlockController) Num() {
	num, err := b.GetInt64(":num")
	if err != nil {
		b.Data["json"] = err.Error()
	} else {
		block := models.GetBlockByNum(num)
		b.Data["json"] = block
	}
	b.ServeJSON()
}

// @Title Get block by id
// @Description Get block by id
// @Param	id		path 	string	true
// @Success 200 {block} models.Block
// @router /id/:id [get]
func (b *BlockController) Id() {
	id := b.GetString(":id")
	if id != "" {
		block := models.GetBlockById(id)
		b.Data["json"] = block
	}
	b.ServeJSON()
}
