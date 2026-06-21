// Package models 1
package models

import (
	"fmt"
	"ginchat/utils"
	"time"

	"gorm.io/gorm"
)

type UserBasic struct {
	gorm.Model
	Name          string `valid:"required,length(3|20)"`            // 必填，3-20字符
	Password      string `valid:"required,length(6|50)"`            // 必填，6-50字符
	Phone         string `valid:"optional,matches(^1[3-9]\\d{9}$)"` // 可选，11位手机号
	Email         string `valid:"optional,email"`                   // 可选，邮箱格式
	Identity      string
	ClientIP      string
	ClientPort    string
	Salt          string
	Logintime     *time.Time
	Heartbeattime *time.Time
	LoginOutTime  *time.Time `gorm:"column:login_out_time" json:"login_out_time"`
	Islogout      bool
	Deviceinfo    string
}

// gorm的表名方法，为了告诉gorm在操作数据库的时候使用哪个表名
//Go 的零值时间 time.Time{} 转换到 MySQL 时变成了 '0000-00-00' ，MySQL 不接受这个值。
//用*time.Time可以解决这个问题

func (table *UserBasic) TableName() string {
	return "user_basic"
}

func GetUserList() []*UserBasic {
	var data []*UserBasic //这时还是一个空的切片
	utils.DB.Find(&data)  //用于查询数据库并并将数据填充到data中
	//通过 &data 告诉 GORM："请把结果存到 data 这个变量的内存地址里"
	for _, i := range data {
		fmt.Println(i)
	}
	return data
}

func CreateUser(user UserBasic) error {
	result := utils.DB.Create(&user)
	return result.Error
}

func FindUserbyname(name string) *UserBasic {
	user := UserBasic{}
	result := utils.DB.Where("name = ?", name).First(&user)
	if result.Error != nil {
		return nil // 用户不存在
	}
	return &user // 用户存在
}

func FindUserByPhone(phone string) *UserBasic {
	user := UserBasic{}
	result := utils.DB.Where("phone = ?", phone).First(&user)
	if result.Error != nil {
		return nil // 手机号不存在
	}
	return &user
}

func FindUserByEmail(email string) *UserBasic {
	user := UserBasic{}
	result := utils.DB.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil // 邮箱不存在
	}
	return &user
}

func FindUserByNameAndPwd(name, password string) *UserBasic {
	user := UserBasic{}
	// 1. 先根据用户名查找用户
	result := utils.DB.Where("name = ?", name).First(&user)
	if result.Error != nil {
		return nil // 用户不存在
	}

	// 2. 用盐值加密输入的密码，与数据库比对
	if !utils.ValidPassword(password, user.Salt, user.Password) {
		return nil // 密码错误
	}

	return &user
}

func GetUserByID(id uint) (*UserBasic, error) {
	cacheKey := utils.UserCachePrefix + fmt.Sprintf("%d", id)
	var cachedUser UserBasic
	//获取用户，先查缓存
	err := utils.GetCache(cacheKey, &cachedUser)
	if err == nil {
		return &cachedUser, nil
	}

	//缓存没命中，查数据库
	var user UserBasic
	result := utils.DB.First(&user, id)
	if result.Error != nil {
		return nil, result.Error //返回查询到的第一个结构体与错误
	}

	//数据库找到了，记得写入缓存
	utils.SetCache(cacheKey, &user)
	return &user, nil
}

func UpdateUser(user UserBasic) error { //参数用户更新的值，返回可能出现的错误
	//更新数据库
	result := utils.DB.Model(&user).Updates(user)
	//删除缓存，因为更新了，下次获取要在数据库中重新加载了
	utils.DelCache(utils.UserCachePrefix + fmt.Sprintf("%d", user.ID))
	return result.Error
}

func DeleteUser(user UserBasic) error {
	utils.DelCache(utils.UserCachePrefix + fmt.Sprintf("%d", user.ID))
	result := utils.DB.Delete(&user)
	return result.Error
}
