// Package main 1
package main

import (
	"fmt"
	"ginchat/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//orm实质是将go语言转化为sql
func main() {
	// MySQL数据库连接配置
	//dsn := "root:root123@tcp(localhost:3306)/ginchat?charset=utf8mb4&parseTime=True&loc=Local" //时区汇报之类的
	//代码通过dsn配置来定位到数据库中的表
  db, err := gorm.Open(mysql.Open(""), &gorm.Config{})
	if err != nil {
		panic("连接数据库失败")
	}

	// 自动建表
	db.AutoMigrate(&models.UserBasic{}) //这里是传指针的

	// 创建用户
	var user models.UserBasic
	result := db.Create(&models.UserBasic{
		Name:     "testuser",
		Password: "123456",
		Phone:    "13800138000",
		Email:    "test@example.com",
	})
	fmt.Printf("创建用户结果 - Error: %v, RowsAffected: %d\n", result.Error, result.RowsAffected)
	if result.Error != nil {
		panic("创建用户失败")
	}

	// 查询所有用户
	var users []models.UserBasic
	err = db.Where("name = ?", "testuser").Find(&users).Error // 查找 name 为 testuser 的所有用户
	fmt.Printf("查询用户结果 - Error: %v, Found: %d\n", err, len(users))
	if err != nil {
		panic("查询用户失败")
	}

	// 获取第一个查询到的用户进行后续操作
	if len(users) > 0 {
		user = users[0]
		fmt.Printf("获取到用户 - ID: %d, Name: %s\n", user.ID, user.Name)
	} else {
		fmt.Println("没有找到用户")
	}

	// 更新 - 更新用户邮箱
	err = db.Model(&user).Where("id = ?", user.ID).Update("email", "updated@example.com").Error
	if err != nil {
		panic("更新用户邮箱失败")
	}
	// 更新 - 更新多个字段
	err = db.Model(&user).Where("id = ?", user.ID).Updates(models.UserBasic{
		Name:  "updateduser",
		Email: "newemail@example.com",
	}).Error
	if err != nil {
		panic("更新多个字段失败")
	}

	// 删除 - 删除用户（已注释，保留数据）
	// err = db.Where("id = ?", user.ID).Delete(&models.UserBasic{}).Error
	// if err != nil {
	// 	panic("删除用户失败")
	// }
}
