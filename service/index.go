// Package service 1
package service

import "github.com/gin-gonic/gin"

// GetIndex 首页
// @Summary 首页
// @Description 返回欢迎信息
// @Tags index
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /index [get]
func GetIndex(c *gin.Context) {
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "welcome !",
		"data": nil,
	})
}
