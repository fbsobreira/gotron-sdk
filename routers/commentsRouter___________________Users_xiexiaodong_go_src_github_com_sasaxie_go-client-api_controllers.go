package routers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context/param"
)

func init() {

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AccountController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AccountController"],
		beego.ControllerComments{
			Method: "Get",
			Router: `/address/:address`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AccountController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AccountController"],
		beego.ControllerComments{
			Method: "CreateAccount",
			Router: `/create`,
			AllowHTTPMethods: []string{"post"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AccountController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AccountController"],
		beego.ControllerComments{
			Method: "NetMessage",
			Router: `/net-message/address/:address`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AssetIssueController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AssetIssueController"],
		beego.ControllerComments{
			Method: "Address",
			Router: `/address/:address`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AssetIssueController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AssetIssueController"],
		beego.ControllerComments{
			Method: "List",
			Router: `/list`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AssetIssueController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:AssetIssueController"],
		beego.ControllerComments{
			Method: "Name",
			Router: `/name/:name`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:BlockController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:BlockController"],
		beego.ControllerComments{
			Method: "Id",
			Router: `/id/:id`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:BlockController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:BlockController"],
		beego.ControllerComments{
			Method: "LatestNum",
			Router: `/latest-num/:num`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:BlockController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:BlockController"],
		beego.ControllerComments{
			Method: "Now",
			Router: `/now`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:BlockController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:BlockController"],
		beego.ControllerComments{
			Method: "Num",
			Router: `/num/:num`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:BlockController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:BlockController"],
		beego.ControllerComments{
			Method: "GetBlockByLimit",
			Router: `/start/:start/end/:end`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:NodeController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:NodeController"],
		beego.ControllerComments{
			Method: "List",
			Router: `/list`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:NumberMessageController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:NumberMessageController"],
		beego.ControllerComments{
			Method: "NextMaintenanceTime",
			Router: `/next-maintenance-time`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:NumberMessageController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:NumberMessageController"],
		beego.ControllerComments{
			Method: "TotalTransaction",
			Router: `/total-transaction`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:ObjectController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:ObjectController"],
		beego.ControllerComments{
			Method: "Post",
			Router: `/`,
			AllowHTTPMethods: []string{"post"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:ObjectController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:ObjectController"],
		beego.ControllerComments{
			Method: "GetAll",
			Router: `/`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:ObjectController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:ObjectController"],
		beego.ControllerComments{
			Method: "Get",
			Router: `/:objectId`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:ObjectController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:ObjectController"],
		beego.ControllerComments{
			Method: "Put",
			Router: `/:objectId`,
			AllowHTTPMethods: []string{"put"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:ObjectController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:ObjectController"],
		beego.ControllerComments{
			Method: "Delete",
			Router: `/:objectId`,
			AllowHTTPMethods: []string{"delete"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:TransactionController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:TransactionController"],
		beego.ControllerComments{
			Method: "Id",
			Router: `/id/:id`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"],
		beego.ControllerComments{
			Method: "Post",
			Router: `/`,
			AllowHTTPMethods: []string{"post"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"],
		beego.ControllerComments{
			Method: "GetAll",
			Router: `/`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"],
		beego.ControllerComments{
			Method: "Get",
			Router: `/:uid`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"],
		beego.ControllerComments{
			Method: "Put",
			Router: `/:uid`,
			AllowHTTPMethods: []string{"put"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"],
		beego.ControllerComments{
			Method: "Delete",
			Router: `/:uid`,
			AllowHTTPMethods: []string{"delete"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"],
		beego.ControllerComments{
			Method: "Login",
			Router: `/login`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:UserController"],
		beego.ControllerComments{
			Method: "Logout",
			Router: `/logout`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

	beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:WitnessController"] = append(beego.GlobalControllerRouter["github.com/sasaxie/go-client-api/controllers:WitnessController"],
		beego.ControllerComments{
			Method: "List",
			Router: `/list`,
			AllowHTTPMethods: []string{"get"},
			MethodParams: param.Make(),
			Params: nil})

}
