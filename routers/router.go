// @APIVersion 1.0.0
// @Title go-client-api Test API
// @Description go-client-api is tron-java grpc client
// @TermsOfServiceUrl https://tron.network/
package routers

import (
	"github.com/astaxie/beego"
	"github.com/sasaxie/go-client-api/controllers"
)

func init() {
	ns := beego.NewNamespace("/v1",
		beego.NSNamespace("/account",
			beego.NSInclude(
				&controllers.AccountController{},
			),
		),
		beego.NSNamespace("/witness",
			beego.NSInclude(
				&controllers.WitnessController{},
			),
		),
		beego.NSNamespace("/node",
			beego.NSInclude(
				&controllers.NodeController{},
			),
		),
	)
	beego.AddNamespace(ns)
}
