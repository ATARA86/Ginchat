// Package service
package service

import (
	"fmt"
	"ginchat/models"
	"ginchat/utils"
	"math/rand"
	"net/http"
	"strconv"

	//"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/asaskevich/govalidator" //这是一个用于检查格式的包
	"github.com/gin-gonic/gin"
	//"github.com/gorilla/websocket"
)

// GetUserList 获取用户列表
// @Summary 获取用户列表
// @Description 返回所有用户的信息
// @Tags 用户
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /users [get]
func GetUserList(c *gin.Context) {
	data := models.GetUserList()
	utils.RespSuccess(c, data)
}

// gin-jwt只返回了token和expire（过期时间），没有用户信息

func GetCurrentUser(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	identity := claims["identity"].(string)
	user := models.FindUserbyname(identity)
	if user == nil {
		utils.RespFail(c, "用户不存在")
		return
	}
	utils.RespSuccess(c, user)
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户
// @Accept json
// @Produce json
// @Param name query string false "用户名"
// @Param password query string false "密码"
// @Param repassword query string false "确认密码"
// @Param phone query string false "电话"
// @Param email query string false "邮箱"
// @Success 200 {object} map[string]interface{}
// @Router /users [post]
func CreateUser(c *gin.Context) {
	user := models.UserBasic{}

	//创建一个结构体来存储前端读到的JSON BODY数据
	var reqData struct {
		Name       string `form:"name"`
		Password   string `form:"password"`
		Repassword string `form:"repassword"`
		Phone      string `form:"phone"`
		Email      string `form:"email"`
	}
	c.ShouldBind(&reqData)
	// 	c.ShouldBind(&reqData) 可以自动识别：
	// - URL 参数 ?name=xxx
	// - Form 表单数据
	// - JSON body

	user.Name = reqData.Name
	password := reqData.Password
	repassword := reqData.Repassword
	user.Phone = reqData.Phone
	user.Email = reqData.Email

	salt := fmt.Sprintf("%06d", rand.Int31())

	// 重要！先把所有值都赋给 user，再验证
	user.Name = reqData.Name
	user.Password = utils.MakePassword(password, salt) // 加密密码
	user.Salt = salt                                   // 保存盐值到数据库，记住了！
	user.Phone = reqData.Phone
	user.Email = reqData.Email

	_, err := govalidator.ValidateStruct(user)
	if err != nil {
		fmt.Println(err)
		utils.RespFail(c, "格式不正确！")
		return
	}

	if models.FindUserbyname(user.Name) != nil {
		utils.RespFail(c, "用户名已存在！")
		return
	}

	if user.Phone != "" {
		if models.FindUserByPhone(user.Phone) != nil {
			utils.RespFail(c, "手机号已被注册！")
			return
		}
	}

	if user.Email != "" {
		if models.FindUserByEmail(user.Email) != nil {
			utils.RespFail(c, "邮箱已被注册！")
			return
		}
	}

	if password == "" {
		utils.RespFail(c, "密码不能为空")
		return
	}
	if password != repassword {
		utils.RespFail(c, "两次密码不一致")
		return
	}

	err = models.CreateUser(user)
	if err != nil {
		utils.RespFail(c, "创建用户失败: "+err.Error())
		return
	}

	utils.RespSuccess(c, nil)
}

// UserLogin 用户登录
// @Summary 用户登录
// @Description 通过用户名和密码登录
// @Tags 用户
// @Accept json
// @Produce json
// @Param name query string true "用户名"
// @Param password query string true "密码"
// @Success 200 {object} map[string]interface{}
// @Router /users/login [post]
func UserLogin(c *gin.Context) {
	name := c.Query("name")
	password := c.Query("password")

	if name == "" || password == "" {
		utils.RespFail(c, "用户名或密码不能为空")
		return
	}

	user := models.FindUserByNameAndPwd(name, password)
	if user == nil {
		utils.RespFail(c, "用户名或密码错误")
		return
	}

	utils.RespSuccess(c, user)
}

// GetUserByID 获取单个用户
// @Summary 获取单个用户
// @Description 根据ID获取用户信息
// @Tags 用户
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} map[string]interface{}
// @Router /users/{id} [get]
func GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.RespFail(c, "无效的用户ID")
		return
	}

	user, err := models.GetUserByID(uint(id))
	if err != nil {
		utils.RespFail(c, "用户不存在")
		return
	}

	utils.RespSuccess(c, user)
}

// UpdateUser 更新用户
// @Summary 更新用户
// @Description 根据ID更新用户信息
// @Tags 用户
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param name query string false "用户名"
// @Param password query string false "密码"
// @Param phone query string false "电话"
// @Param email query string false "邮箱"
// @Success 200 {object} map[string]interface{}
// @Router /users/{id} [put]
func UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.RespFail(c, "无效的用户ID")
		return
	}

	user, err := models.GetUserByID(uint(id))
	if err != nil {
		utils.RespFail(c, "用户不存在")
		return
	}

	name := c.Query("name")
	password := c.Query("password")
	phone := c.Query("phone")
	email := c.Query("email")

	_, err = govalidator.ValidateStruct(user)
	if err != nil {
		fmt.Println(err)
		utils.RespFail(c, "修改的格式不正确！重新输入！")
		return
	}

	if name != "" {
		user.Name = name
	}
	if password != "" {
		user.Password = password
	}
	if phone != "" {
		user.Phone = phone
	}
	if email != "" {
		user.Email = email
	}

	err = models.UpdateUser(*user)
	if err != nil {
		utils.RespFail(c, "更新用户失败: "+err.Error())
		return
	}

	utils.RespSuccess(c, nil)
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 根据ID删除用户
// @Tags 用户
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} map[string]interface{}
// @Router /users/{id} [delete]
func DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.RespFail(c, "无效的用户ID")
		return
	}

	user := models.UserBasic{}
	user.ID = uint(id)

	err = models.DeleteUser(user)
	if err != nil {
		utils.RespFail(c, "删除用户失败: "+err.Error())
		return
	}

	utils.RespSuccess(c, nil)
}

// SearchFriends 搜索好友列表
// @Summary 搜索好友列表
// @Description 获取当前用户的好友列表
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /users/auth/friends [get]
func SearchFriends(c *gin.Context) {
	//从jwt token提取用户claim
	claims := jwt.ExtractClaims(c)
	//获取身份标识
	identity := claims["identity"].(string)
	//查询用户信息
	user := models.FindUserbyname(identity)
	if user == nil {
		utils.RespFail(c, "用户不存在")
		return
	}

	//查询好友列表
	friends := models.SearchFriend(user.ID)
	utils.RespSuccess(c, friends) //返回
}

// AddFriend 添加好友
// @Summary 添加好友
// @Description 添加指定用户为好友
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param target_id query int true "目标用户ID"
// @Success 200 {object} map[string]interface{}
// @Router /users/auth/add-friend [post]
func AddFriend(c *gin.Context) {
	//先获取当前登录用户
	claims := jwt.ExtractClaims(c)
	identity := claims["identity"].(string)
	user := models.FindUserbyname(identity)
	if user == nil {
		utils.RespFail(c, "用户不存在")
		return
	}

	//获取目标用户id
	targetIDStr := c.Query("target_id")
	if targetIDStr == "" {
		utils.RespFail(c, "目标用户ID不能为空")
		return
	}

	//解析id
	targetID, err := strconv.ParseUint(targetIDStr, 10, 32)
	if err != nil {
		utils.RespFail(c, "无效的目标用户ID")
		return
	}

	//检查是否存在
	targetUser, err := models.GetUserByID(uint(targetID))
	if err != nil {
		utils.RespFail(c, "目标用户不存在")
		return
	}

	if user.ID == targetUser.ID {
		utils.RespFail(c, "不能添加自己为好友")
		return
	}

	if models.IsFriend(user.ID, targetUser.ID) {
		utils.RespFail(c, "你们已经是好友了")
		return
	}

	err = models.AddFriend(user.ID, targetUser.ID)
	if err != nil {
		utils.RespFail(c, "添加好友失败: "+err.Error())
		return
	}

	utils.RespSuccess(c, nil)
}

// http是一次性链接，ws是持久性链接，http只有客户端可以发消息，ws双向都可以发，且实时推送
// Gorilla WebSocket 库的 升级器 ，用于将 HTTP 连接升级为 WebSocket 连接。
// var upGrade = websocket.Upgrader{
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true
// 	},
// }

// func SendMsg(c *gin.Context) {
// 	//使用升级器，将http升级为ws链接
// 	ws, err := upGrade.Upgrade(c.Writer, c.Request, nil)
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	defer ws.Close()  //最后执行关闭操作
// 	MsgHandler(ws, c) //处理消息
// }

// //消息处理函数
// //在这里的redis只管进行消息的转发

// func MsgHandler(ws *websocket.Conn, c *gin.Context) {
// 	//订阅函数，订阅redis的ws频道
// 	ch, err := utils.Subscribe(c.Request.Context(), utils.PublishKey)
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}

// 	//循环监听redis消息
// 	for msg := range ch {
// 		//格式化时间
// 		tm := time.Now().Format("2006-01-02 15:04:05")
// 		m := fmt.Sprintf("[%s]:%s", tm, msg.Payload)

// 		//发送给ws客户端
// 		err = ws.WriteMessage(1, []byte(m))
// 		if err != nil {
// 			fmt.Println(err)
// 			return
// 		}
// 	}
// }

func SendUserMsg(w http.ResponseWriter, r *http.Request) {
	models.Chat(w, r)
}




//群聊功能

func CreateGroup(c *gin.Context) {
	//解析请求参数
	var req struct {
		Name  string `json:"name"`
		Desc  string `json:"desc"`
		Owner uint   `json:"owner"`//获取群主id
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespFail(c, "参数错误")
		return
	}

	//调用函数，创建群聊
	group, err := models.CreateGroup(req.Owner, req.Name, req.Desc)
	if err != nil {
		utils.RespFail(c, "创建群失败")
		return
	}
	utils.RespSuccess(c, group)
}

func AddGroup(c *gin.Context) {//加群
	var req struct {
		GroupID uint `json:"group_id"`
		UserID  uint `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespFail(c, "参数错误")
		return
	}

	err := models.AddGroupMember(req.GroupID, req.UserID)
	if err != nil {
		utils.RespFail(c, "加入群失败")
		return
	}
	utils.RespSuccess(c, nil)
}

func GetGroupMembers(c *gin.Context) {
	groupIDStr := c.Query("group_id")
	groupID, _ := strconv.ParseUint(groupIDStr, 10, 64)

	members := models.GetGroupMembers(uint(groupID))//获取群友信息
	utils.RespSuccess(c, members)
}

func GetUserGroups(c *gin.Context) {//获取用户列表
	userIDStr := c.Query("user_id")
	userID, _ := strconv.ParseUint(userIDStr, 10, 64)

	groups := models.GetUserGroups(uint(userID))
	utils.RespSuccess(c, groups)
}

func GetGroupMessages(c *gin.Context) {//获取群信息
	groupIDStr := c.Query("group_id")
	groupID, _ := strconv.ParseUint(groupIDStr, 10, 64)

	messages := models.GetGroupMessages(uint(groupID), 20)
	utils.RespSuccess(c, messages)
}
