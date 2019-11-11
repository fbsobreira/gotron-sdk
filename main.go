package main

import (
	"github.com/astaxie/beego"
	"github.com/fbsobreira/go-client-api/common/global"
	_ "github.com/fbsobreira/go-client-api/routers"
	"github.com/fbsobreira/go-client-api/service"
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
