package main

import (
	"github.com/astaxie/beego"
	_ "jebeego_api/controllers"
	_ "jebeego_api/routers"
)

func main() {
	beego.Run()
}
