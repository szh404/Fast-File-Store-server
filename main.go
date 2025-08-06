package main

import (
	"file-store/lib"
	"file-store/model/mysql"
	"file-store/router"
	"log"
	"fmt"
)

func main() {
	fmt.Println("Hello Git")
	serverConfig := lib.LoadServerConfig()
	mysql.InitDB(serverConfig)
	defer mysql.DB.Close()

	r := router.SetupRoute()

	r.LoadHTMLGlob("view/*")
	r.Static("/static", "./static")


	if err := r.Run(":8080"); err != nil {
		log.Fatal( "服务器启动失败...")
	}
}
