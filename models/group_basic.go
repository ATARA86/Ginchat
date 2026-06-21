// Package models
package models

import (
	"fmt"
	"ginchat/utils"
	"time"

	"gorm.io/gorm"
)

// - CreateGroup - 创建群组（创建者自动成为群成员）
// - AddGroupMember - 添加群成员
// - RemoveGroupMember - 移除群成员
// - GetGroupMembers - 获取群成员列表
// - GetUserGroups - 获取用户加入的群列表
// - GetGroupMessages - 获取群消息历史
// - sendGroupMsg - 群发消息给所有成员

//群聊信息表

type GroupBasic struct {
	gorm.Model
	Name    string
	OwnerID uint   //群主id
	Icon    string //群图标
	Type    int    //群类型
	Desc    string //群描述
}

func (table *GroupBasic) TableName() string {
	return "group_basic"
}


//创建群逻辑

func CreateGroup(ownerID uint, name, desc string) (*GroupBasic, error) {
	group := &GroupBasic{
		Name:    name,
		OwnerID: ownerID,
		Desc:    desc,
		Type:    1,//普通群类型
	}
	//写入数据库
	result := utils.DB.Create(group)
	if result.Error != nil {
		return nil, result.Error
	}

	//自动执行，将创建者加入群里
	err := AddGroupMember(group.ID, ownerID)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func AddGroupMember(groupID, userID uint) error {//添加成员逻辑
	//关键点 ： contacts 是一个 切片 ，包含了 所有 群成员关系记录，不是一个！
	contact := Contact{
		OwnerID:  groupID,//群id
		TargetID: userID,//成员id
		Type:     2,//成员类型
	}
	return utils.DB.Create(&contact).Error//对这个成员执行添加
}

func RemoveGroupMember(groupID, userID uint) error {//移除
	//这其实就是两个查询语句吧
	return utils.DB.Where("owner_id = ? AND target_id = ? AND type = 2", groupID, userID).Delete(&Contact{}).Error
}


func GetGroupMembers(groupID uint) []UserBasic {
	//获取该群所有成员关系
	contacts := make([]Contact, 0)
	utils.DB.Where("owner_id = ? AND type = 2", groupID).Find(&contacts)

	//提取用户id
	userIDs := make([]uint64, 0)
	for _, c := range contacts {
		userIDs = append(userIDs, uint64(c.TargetID))
	}

	//批量查询用户信息
	users := make([]UserBasic, 0)
	if len(userIDs) > 0 {
		utils.DB.Where("id IN ?", userIDs).Find(&users)
	}
	return users
}

func GetUserGroups(userID uint) []GroupBasic {
	//查找用户在哪些群里
	contacts := make([]Contact, 0)
	utils.DB.Where("target_id = ? AND type = 2", userID).Find(&contacts)

	groupIDs := make([]uint64, 0)
	for _, c := range contacts {
		groupIDs = append(groupIDs, uint64(c.OwnerID))
	}

	groups := make([]GroupBasic, 0)
	if len(groupIDs) > 0 {
		utils.DB.Where("id IN ?", groupIDs).Find(&groups)
	}
	return groups
}

func GetGroupByID(id uint) (*GroupBasic, error) {
	//按照id获取群信息
	var group GroupBasic
	result := utils.DB.First(&group, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &group, nil
}

func SaveGroupMessage(formID, targetID int64, msgType, media int, content string) error {
	//保存群聊信息
	msg := Message{
		FormID:   formID,
		TargetID: targetID,
		Type:     msgType,
		Media:    media,
		Content:  content,
	}
	return utils.DB.Create(&msg).Error
}

func GetGroupMessages(groupID uint, limit int) []Message {
	if limit <= 0 {
		limit = 20
	}
	msgs := make([]Message, 0)
	utils.DB.Where("target_id = ? AND type = 2", groupID).Order("created_at DESC").Limit(limit).Find(&msgs)
	return msgs
}

func GetGroupMessageCacheKey(groupID uint) string {//生成缓存key
	return fmt.Sprintf("group:%d:messages", groupID)
}

func SaveGroupMessageCache(groupID uint, msgs []Message) error {//保存群消息到缓存
	key := GetGroupMessageCacheKey(groupID)
	return utils.SetCache(key, msgs)
}

func GetGroupMessageCache(groupID uint) ([]Message, error) {//从缓存获取群消息
	key := GetGroupMessageCacheKey(groupID)
	var msgs []Message
	err := utils.GetCache(key, &msgs)
	return msgs, err
}

var LastGroupMsgTime = make(map[uint]time.Time)//记录最后消息的时间

func ShouldSaveGroupMessage(groupID uint) bool {//检查是否要存到数据库
	lastTime, exists := LastGroupMsgTime[groupID]
	if !exists {
		return true
	}
	return time.Since(lastTime) > 5*time.Minute
}

func UpdateLastGroupMsgTime(groupID uint) {
	LastGroupMsgTime[groupID] = time.Now()
}
