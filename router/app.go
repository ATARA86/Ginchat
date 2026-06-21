// Package app
package app

import (
	"ginchat/docs"
	"ginchat/models"
	"ginchat/service"
	"ginchat/utils"
	"log"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Router() *gin.Engine {

	r := gin.Default()

	// CORS 中间件，解决跨域问题，允许前端访问后端接口。
	//CORS = Cross-Origin Resource Sharing（跨域资源共享）
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		//允许 所有 来源访问（ * 表示任意域名）
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		//允许的 请求方法   GET, POST, PUT, DELETE
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		//允许的 请求头 （Content-Type 用于 JSON，Authorization 用于 Token）
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
	//为什么要用cors，前端的Ajax请求后端，发送JSON数据，携带TOKEN

	//swagger
	docs.SwaggerInfo.BasePath = ""
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// 静态文件服务
	r.Static("/static", "./static")

	// 首页指向 static/index.html
	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.File("./static/index.html")
	})

	// JWT 中间件配置
	//身份标识 证明你是谁（用户 ID）
	//防篡改 改了内容签名就失效
	//无需存储 Token 本身包含信息，不用查数据库
	//有效期 过期了需要重新登录
	//生成的token包含用户信息，直接返回给客户端，不需要存到数据库中，性能更好
	//让各个服务器独立，models里的那个所有都要在本地计算
	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "ginchat",            //名称标识
		Key:         []byte("secret key"), //签名钥匙，类似密码
		Timeout:     time.Hour,            //token有效期
		MaxRefresh:  time.Hour,
		IdentityKey: "identity", //存储用户身份

		//把用户id存入token
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(string); ok {
				return jwt.MapClaims{
					"identity": v,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			//提取用户身份
			claims := jwt.ExtractClaims(c)
			return claims["identity"]
		},

		Authenticator: func(c *gin.Context) (interface{}, error) { //认证函数
			//这里验证用户名，密码
			//接受登录参数
			var loginVals struct {
				Username string `form:"name" json:"name" binding:"required"`
				Password string `form:"password" json:"password" binding:"required"`
			}
			//绑定请求函数
			if err := c.ShouldBind(&loginVals); err != nil {
				return "", err
			}

			//根据用户名查找用户
			user := models.FindUserbyname(loginVals.Username)
			if user == nil {
				return "", jwt.ErrFailedAuthentication //用户不存在
			}

			//验证密码
			if !utils.ValidPassword(loginVals.Password, user.Salt, user.Password) {
				return "", jwt.ErrFailedAuthentication
			}

			//返回用户名
			return loginVals.Username, nil
		},
	})
	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}

	r.NoRoute(authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		utils.RespFail(c, "route not found")
	})

	users := r.Group("/users")
	{
		users.GET("", service.GetUserList)
		users.POST("", service.CreateUser)

		users.POST("/login", authMiddleware.LoginHandler)
		//- 调用你的 Authenticator 验证用户名密码
		// 调用你的 PayloadFunc 获取用户信息
		// 用 Key 签名生成 Token
		// 返回 Token 给前端
		// WebSocket 路由
		//用来发送消息
		//r.GET("/ws", service.SendMsg)
		r.GET("/ws/user", func(c *gin.Context) {
			service.SendUserMsg(c.Writer, c.Request)
		})

		// 需要登录的路由
		auth := users.Group("/auth")
		auth.Use(authMiddleware.MiddlewareFunc())
		{
			auth.GET("/me", service.GetCurrentUser)
			auth.GET("/friends", service.SearchFriends)
			auth.POST("/add-friend", service.AddFriend)
			auth.GET("/:id", service.GetUserByID)
			//这里的id是动态路由参数，表示可以匹配任意值
			auth.PUT("/:id", service.UpdateUser)
			auth.DELETE("/:id", service.DeleteUser)
		}
	}

	groups := r.Group("/groups")
	{
		groups.POST("", service.CreateGroup)
		groups.POST("/add", service.AddGroup)
		groups.GET("/members", service.GetGroupMembers)
		groups.GET("/user/:user_id", service.GetUserGroups)
		groups.GET("/messages", service.GetGroupMessages)
	}

	return r

	//restful原则
	//get用来查询，获取数据，不会修改数据，浏览器可以直接缓存
	//post为创建新数据，每次请求创建新资源
	//put完整替换某个资源，没上传的会被清空/设为默认值
}
