// Package models
package models

import (
	"fmt"
	"ginchat/utils"

	"gorm.io/gorm"
)

//人员关系表

type Contact struct {
	gorm.Model
	OwnerID  uint //谁的关系信息
	TargetID uint //对应的是谁
	Type     int  //对应的类型 1好友 2群 3
	Desc     string
}

func (table *Contact) TableName() string {
	return "contact"
}

//这样进行了两次的查询

func SearchFriend(userID uint) []UserBasic {
	cacheKey := utils.FriendCachePrefix + fmt.Sprintf("%d", userID)
	var cachedFriends []UserBasic
	err := utils.GetCache(cacheKey, &cachedFriends)
	if err == nil {
		return cachedFriends
	}

	contacts := make([]Contact, 0)
	objIds := make([]uint64, 0)

	utils.DB.Where("owner_id = ? and type = 1", userID).Find(&contacts)
	//find查询所有

	for _, v := range contacts { //遍历查询结果
		objIds = append(objIds, uint64(v.TargetID))
	}

	// // 第2步：用这些 ID 去 user_basic 表查用户信息
	users := make([]UserBasic, 0)
	if len(objIds) > 0 {
		utils.DB.Where("id IN ?", objIds).Find(&users) //查询匹配切片
	}

	utils.SetCache(cacheKey, users)
	return users
}

// AddFriend 添加好友（双向插入）
func AddFriend(ownerID, targetID uint) error {
	// 构造两条记录：owner->target 和 target->owner
	contacts := []Contact{
		{OwnerID: ownerID, TargetID: targetID, Type: 1},
		{OwnerID: targetID, TargetID: ownerID, Type: 1},
	}
	err := utils.DB.Create(&contacts).Error
	if err != nil {
		return err
	}
	utils.DelCache(utils.FriendCachePrefix + fmt.Sprintf("%d", ownerID))
	utils.DelCache(utils.FriendCachePrefix + fmt.Sprintf("%d", targetID))
	return nil
}

// IsFriend 检查两个用户是否已经是好友
//检查 ownerID 的好友列表里是否已经有 targetID
func IsFriend(ownerID, targetID uint) bool {
	var count int64
	utils.DB.Where("owner_id = ? AND target_id = ? AND type = 1", ownerID, targetID).Find(&Contact{}).Count(&count)
	return count > 0
}
