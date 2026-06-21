package main

// @title GinChat API
// @version 1.0
// @description GinChat 项目 API 文档
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host 127.0.0.1:8081
// @BasePath /
// Package main ginchat项目程序入口

import (
	"log"

	"ginchat/models"
	app "ginchat/router" //main函数引用的包的包名不应该为main
	"ginchat/utils"
)

// 为什么是 router.Router() 而不是 app.Router()？
// 答案： 这取决于你导入的包名。你写的是 router "ginchat/router" ，
// 所以必须用 router.Router() 。如果你想用 app.Router() ，应该写成 app "ginchat/router" 。

func main() {
	//初始化配置文件以及初始化，链接数据库
	utils.InitConfig()

	db, err := utils.InitMysql()
	if err != nil {
		panic("MySQL初始化失败: " + err.Error())
	}
	// 在删除表之后gorm自动创建表结构，方便调试
	db.AutoMigrate(&models.Message{})
	db.AutoMigrate(&models.GroupBasic{})
	db.AutoMigrate(&models.Contact{})

	_, err = utils.InitRedis()
	if err != nil {
		log.Println("Redis初始化失败: " + err.Error())
	}

	r := app.Router()
	r.Run(":8081")
}
