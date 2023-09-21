package main

import (
	"github.com/jxskiss/mcli"
)

type CommonOptions struct {
	Username    string `cli:"#R, -u, --user, Username"`
	Password    string `cli:"#R, -p, --pass, Password"`
	ClientId    string `cli:"#R, -c, --client, Client ID"`
	RedirectUri string `cli:"#O, -r, --redirect, Redirect URI" default:"http://localhost:4200/"`
	UseCache    bool   `cli:"#O, -C, --use-cache, Use cache" default:"false"`
	CachePath   string `cli:"#O, -P, --cache-path, Cache path" default:"/tmp"`
}

type Context struct {
	Username       string
	Password       string
	ClientId       string
	GatewayId      string
	InstallationId string
	DeviceId       string
	RedirectUri    string
	CodeVerifier   string
	Cache          Cache
}

const API_BASE_URL = "https://api.viessmann-platform.io"

func showHelpCommand() {
	mcli.PrintHelp()
}

func showVersionCommand() {
	println(VERSION)
}

func main() {
	app := mcli.App{
		Description: "Viessmann API Query Tool v" + VERSION,
	}

	app.Add("installations", getInstallationsCommand, "Get Installations JSON")
	app.Add("features", getFeaturesCommand, "Get Features JSON")
	app.Add("help", showHelpCommand, "This help message")
	app.Add("version", showVersionCommand, "Returns version number")

	app.Run()
}
