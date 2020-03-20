package main

import (
	"github.com/astaxie/beego"
	"github.com/fbsobreira/gotron/common/global"
	_ "github.com/fbsobreira/gotron/routers"
	"github.com/fbsobreira/gotron/service"
)

func main() {
	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}

	grpcAddress := beego.AppConfig.String("grpcaddress")

	global.TronClient = service.NewGrpcClient(grpcAddress)
	global.TronClient.Start()
	defer global.TronClient.Conn.Close()

	beego.Run()
}
