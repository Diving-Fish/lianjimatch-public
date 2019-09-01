package main

import (
	"github.com/kataras/iris"
)
import "match/controller"

func main() {
	app := iris.Default()
	crs := func(ctx iris.Context) {
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Credentials", "true")
		ctx.Header("Access-Control-Allow-Headers", "Access-Control-Allow-Origin,Content-Type,Authorization")
		if ctx.Method() == "OPTIONS" {
			return
		}
		ctx.Next()
	}
	app.Use(crs)
	admin := app.Party("/api/admin", controller.AdminHandler).AllowMethods(iris.MethodOptions)
	{
		admin.Get("/fetch_all_players", controller.FetchAllPlayers)
		admin.Get("/delete_team", controller.DeleteTeamAdmin)
		admin.Get("/stat", controller.AdminAuth)
		admin.Get("/get_status", controller.GetAllStatus)
	}
	team := app.Party("/api/team", controller.AuthHandler).AllowMethods(iris.MethodOptions)
	{
		team.Post("/add_player", controller.AddPlayer)
		team.Get("/stat", controller.Auth)
		team.Get("/delete", controller.DeleteTeam)
		team.Post("/position", controller.SetPosition)
	}
	public := app.Party("/api/public").AllowMethods(iris.MethodOptions)
	{
		public.Get("/send_process", controller.SendProcess)
		public.Get("/get_groups", controller.GetAllGroups)
		public.Get("/get_stats", controller.GetAllStats)
		public.Get("/get_area_stats", controller.GetStatsByGroup)
		public.Post("/create_team", controller.CreateTeam)
		public.Post("/login", controller.Login)
		public.Get("/get_players", controller.GetPlayers)
		public.Post("/single_sign", controller.SingleSign)
		public.Get("/query_player", controller.QueryPlayer)
	}
	_ = app.Run(iris.Addr(":8088"))
}
