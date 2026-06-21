package utils

import (
	"net/http"
	"github.com/gin-gonic/gin"
)


//json.Marshal 是 Go 语言标准库 encoding/json 包中的函数
//用于将 Go 数据结构转换为 JSON 格式的字节切片（[]byte）。

type H struct {
	Code  int         `json:"code"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data,omitempty"`
	Row   interface{} `json:"row,omitempty"`
	Total interface{} `json:"total,omitempty"`
}

//之前用的gin.H是一个map，返回什么是什么
//这里的H是一个结构体

func Resp(c *gin.Context, code int, data interface{}, msg string) {
	c.JSON(http.StatusOK, H{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

func RespSuccess(c *gin.Context, data interface{}) {
	Resp(c, 200, data, "操作成功")
}

func RespSuccessWithMsg(c *gin.Context, data interface{}, msg string) {
	//成功响应，自定义 msg
	Resp(c, 200, data, msg)
}

func RespFail(c *gin.Context, msg string) {
	Resp(c, -1, nil, msg)
}

func RespFailWithCode(c *gin.Context, code int, msg string) {
	Resp(c, code, nil, msg)
}

func RespError(c *gin.Context, code int, msg string) {
	Resp(c, code, nil, msg)
}

func RespPage(c *gin.Context, data interface{}, total interface{}) {
	c.JSON(http.StatusOK, H{
		Code:  200,
		Msg:   "查询成功",
		Data:  data,
		Total: total,
	})
}
