┌─────────────────────────────────────────────────────────┐
│                gin.Context (请求上下文)                  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  请求信息:                                              │
│  ├─ c.Request.Method      // GET, POST, PUT, DELETE    │
│  ├─ c.Request.URL.Path   // 请求路径                   │
│  ├─ c.Request.Header     // 请求头                     │
│  ├─ c.Query("name")      // URL 查询参数 ?name=xxx    │
│  ├─ c.PostForm("email")  // POST 表单参数              │
│  ├─ c.Param("id")       // 路径参数 /user/:id         │
│  └─ c.Body               // 请求体 JSON                │
│                                                         │
│  响应信息:                                              │
│  ├─ c.JSON()             // 返回 JSON                  │
│  ├─ c.HTML()             // 返回 HTML                  │
│  ├─ c.String()           // 返回字符串                 │
│  └─ c.File()             // 返回文件                   │
│                                                         │
│  其他:                                                  │
│  ├─ c.Keys               // 存储自定义数据              │
│  ├─ c.MustGet()          // 获取中间件传递的数据        │
│  └─ c.Abort()            // 终止请求                    │
│                                                         │
└─────────────────────────────────────────────────────────┘

用于请求上下文

┌─────────────────────────────────────────────────────────┐
│                    HTTP 请求流程                         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│   浏览器                                                │
│      │                                                  │
│      ▼ GET /user/getUserList?name=test                 │
│   ┌─────────────────────────────────────────┐          │
│   │         gin.Engine (路由器)               │          │
│   │                                           │          │
│   │  r.GET("/user/getUserList", handler)    │          │
│   │         │                                │          │
│   │         ▼                                │          │
│   │  ┌──────────────────────────────────┐   │          │
│   │  │   gin.Context (请求上下文)        │   │          │
│   │  │   ──────────────────────────     │   │          │
│   │  │   • 包含请求的所有信息            │   │          │
│   │  │   • 包含响应的所有方法            │   │          │
│   │  │   • 在整个请求过程中传递          │   │          │
│   │  └──────────────────────────────────┘   │          │
│   │         │                                │          │
│   │         ▼                                │          │
│   │  handler(c *gin.Context)                │          │
│   │  {                                      │          │
│   │      c.JSON(200, gin.H{...})           │          │
│   │  }                                      │          │
│   └─────────────────────────────────────────┘          │
│                                                         │
└─────────────────────────────────────────────────────────┘
我的方法要解决什么，要相应什么方法，都在gin.Context中，所以这个可以作为handler函数的参数


## gin.Context 的优势
功能                标准库                          Gin 
获取Query参数   request.URL.Query()              c.Query()
获取Path 参数   手动解析                          c.Param() 
获取JSON       手动 unmarshal                   c.ShouldBindJSON()
返回JSON       手动设置 Header + json marshal      c.JSON() 
中间件         手动写                              c.Use() 
参数绑定        手动写                            c.ShouldBind()

所以 *gin.Context 就是 增强版的 http.ResponseWriter ！
//http.ResponseWriter Go 标准库 net/http HTTP 响应对象
//*gin.Context Gin 框架封装 包含请求+响应的完整上下文

c.Query("name") URL 参数 ?name=xxx 简单参数、GET 请求 
c.ShouldBind(&struct) 请求体（JSON/Form） 复杂数据、POST/PUT

方式                发送格式                     后端获取
 Swagger          URL 参数 ?name=xxx          ✅ c.Query 
 页面表单提交        Form 数据                  ✅ c.Query 
 jQuery AJAX      默认 JSON Body              ❌ c.Query