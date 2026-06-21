// Package models
package models

import (
	"encoding/json"
	"fmt"
	"ginchat/utils"
	"net/http"
	"strconv"
	"sync" //这是go语言的并发编程包
	"time"

	"github.com/gorilla/websocket"
	"gopkg.in/fatih/set.v0" //就是redis里的那个
	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	FormID   int64  //发送者
	TargetID int64  //接收者
	Type     int    //消息类型 群聊，私聊，广播
	Media    int    //消息类型  图片 文字 音频
	Content  string //内容
	Pic      string
	URL      string
	Desc     string
	Amount   int //其他数字统计
}

func (table *Message) TableName() string {
	return "message"
}

//发送消息，需要发送者id，接收者id，消息类型，发送内容，发送的类型
//还要校验token ，关系

//用户链接节点

type Node struct {
	Conn      *websocket.Conn //ws链接
	DataQueue chan []byte     //消息发送队列
	GroupSets set.Interface   //用户加入的群组
}

// 映射关系
var clientMap map[int64]*Node = make(map[int64]*Node, 0)

//用来存储所有在线用户

// 思考，用户A向map中添加数据时b也向其中添加数据，c在这时去读取数据，就会发生冲突
// 读写锁
var rwLocker sync.RWMutex //常用的第一个包，读写锁
//读时可以同时读，但是写的时候只能一人写，这时不能有人读

func init() { //init的特性，里面的函数会自动执行
	// UDP相关已移除，直接消息转发不需要广播
}

func Chat(writer http.ResponseWriter, request *http.Request) {
	//这个函数用于校验token的合法性
	query := request.URL.Query()
	userIDStr := query.Get("userID")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64) //解析userid
	token := query.Get("token")
	_ = token
	// TargetID := query.Get("targetID")
	// context := query.Get("context")
	// msgType := query.Get("type")
	isvalida := true
	conn, err := (&websocket.Upgrader{ //升级为ws链接
		CheckOrigin: func(r *http.Request) bool { //检查请求来源，返回true表示接受所有来源
			return isvalida
		},
	}).Upgrade(writer, request, nil) //升级
	if err != nil {
		fmt.Println(err)
		return
	}

	//获取conn
	node := &Node{
		Conn:      conn,                  //ws连接对象
		DataQueue: make(chan []byte, 50), //初始化消息队列，容量为50
		//这个是每个用户的私有通道，点对点进行发送消息
		GroupSets: set.New(set.ThreadSafe), //群组集合（线程安全）
		//对所有操作加锁
	}

	//用户关系
	//userid和node绑定并加锁
	rwLocker.Lock()          //加写锁
	clientMap[userID] = node //存入会话，加锁防止同时读写发生冲突
	rwLocker.Unlock()        //解锁

	//完成发送逻辑
	go sendProc(node) //发送协程

	//完成接收逻辑
	go recvProc(node) //接受协程
	sendMsg(userID, []byte("欢迎进入聊天系统！"))
}

func sendProc(node *Node) {
	for { //一直循环
		data := <-node.DataQueue //一直阻塞等待，消息队列的消息
		err := node.Conn.WriteMessage(websocket.TextMessage, data)
		//发送消息给客户端
		if err != nil { //链接断开退出循环
			fmt.Println(err)
			return
		}
	}
}

func recvProc(node *Node) {
	for { //阻塞等待消息
		_, data, err := node.Conn.ReadMessage() //等待用户消息
		if err != nil {
			fmt.Println(err)
			return
		}
		dispatch(data) //直接调度处理，不再走UDP广播
		fmt.Println("[ws]<<<<<", data)
	}
}

var udpsendChan chan []byte //一个udp发送通道

func init() { //init的特性，里面的函数会自动执行
	udpsendChan = make(chan []byte, 200) //初始化通道容量为200
	//go udpSendProc()                     //启动 UDP发送协程（已禁用）
	//go udpRecvProc()                     //启动 UDP接收协程（已禁用）
}

// 完成udp发送协程
// func udpSendProc() {
// 	//建立udp链接
// 	con, err := net.DialUDP("udp", nil, &net.UDPAddr{
// 		//拨号udp链接到目标
// 		IP:   net.IPv4(192, 168, 80, 255), //广播地址
// 		Port: 3000,                        //udp端口号
// 	})
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	defer con.Close() //关闭链接
// 	//循环发送
// 	for {
// 		data := <-udpsendChan
// 		_, err := con.Write(data) //发送数据
// 		if err != nil {
// 			fmt.Println(err)
// 			return
// 		}
// 	}
// }

//这里用广播的好处：
// 发送一次，所有在线用户都能收到，在线服务器也能收到
// 收到后再根据 TargetID 过滤给特定用户

// 完成udp接收协程
// func udpRecvProc() {
// 	//监听UDP端口
// 	con, err := net.ListenUDP("udp", &net.UDPAddr{
// 		IP:   net.IPv4zero, //监听所有网卡
// 		Port: 3000,
// 	})
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	defer con.Close()
// 	for { //循环接收
// 		var buf [512]byte           //缓冲区
// 		n, err := con.Read(buf[0:]) //阻塞等待接收
// 		//把数据接受到缓冲区，返回读取缓冲区中的字符数
// 		if err != nil {
// 			fmt.Println(err)
// 			return
// 		}
// 		dispatch(buf[0:n]) //调用分发函数，只发送实际读取的字节
// 		//否则会读到多余空字节，或者是缓冲区中尚未清0的部分
// 	}
// }

// 后端调度逻辑
// 解析JSON消息
func dispatch(data []byte) {
	// 1. 消息大小限制 1MB
	if !utils.CheckMessageSize(int64(len(data))) {
		fmt.Println("消息过大，丢弃")
		return
	}

	msg := Message{}
	//把JSON数据解析为message结构体
	// 2. 解析消息
	err := json.Unmarshal(data, &msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 3. 图片消息大小限制 5MB
	if msg.Media == 2 && len(msg.Pic) > 5*1024*1024 {
		fmt.Println("图片过大，丢弃")
		return
	}

	// 4. 消息频率限制（1分钟内最多30条）
	if !utils.CheckUserMessageRate(msg.FormID) {
		fmt.Println("用户消息过于频繁，丢弃")
		return
	}

	saveMessage(&msg) //保存记录

	switch msg.Type {
	case 1:
		sendMsg(msg.TargetID, data)
	case 2:
		sendGroupMsg(msg.TargetID, data)
	}
}

func sendGroupMsg(groupID int64, data []byte) {
	members := GetGroupMembers(uint(groupID))
	for _, member := range members {
		sendMsg(int64(member.ID), data)
	}
}

// saveMessage 私有消息存入Redis缓存，并定期写入MySQL持久化
// 流程：1.写入Redis缓存 2.判断是否该写入数据库 3.批量写入MySQL
func saveMessage(msg *Message) {
	SavePrivateMessageCache(msg)
	if ShouldSavePrivateMessage(msg.FormID, msg.TargetID) {
		FlushPrivateMessagesToDB(msg.FormID, msg.TargetID)
	}
}

// 私聊消息缓存相关

// GetPrivateMessageCacheKey 生成私聊消息缓存的Redis key
// userID大小比较保证两人聊天的key唯一（无论谁先查询）
func GetPrivateMessageCacheKey(userID1, userID2 int64) string {
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}
	return fmt.Sprintf("private:%d:%d:messages", userID1, userID2)
}

// SavePrivateMessageCache 将消息存入Redis缓存
// 缓存结构为列表，最多缓存100条消息防止Redis内存溢出
func SavePrivateMessageCache(msg *Message) error {
	key := GetPrivateMessageCacheKey(msg.FormID, msg.TargetID)
	var msgs []Message
	utils.GetCache(key, &msgs)
	msgs = append(msgs, *msg)
	if len(msgs) > 100 {
		msgs = msgs[len(msgs)-100:]
	}
	return utils.SetCache(key, msgs)
}

// GetPrivateMessageCache 获取私聊消息缓存
func GetPrivateMessageCache(userID1, userID2 int64) ([]Message, error) {
	return GetPrivateMessages(userID1, userID2)
}

// lastPrivateMsgTime 记录每对用户上次写入数据库的时间（内存Map）
var lastPrivateMsgTime = make(map[string]time.Time)

// GetPrivateMsgKey 生成用户对的时间记录key
func GetPrivateMsgKey(userID1, userID2 int64) string {
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}
	return fmt.Sprintf("%d:%d", userID1, userID2)
}

// ShouldSavePrivateMessage 判断是否该将缓存写入数据库
// 距离上次写入超过5分钟则返回true，触发批量写入
func ShouldSavePrivateMessage(userID1, userID2 int64) bool {
	key := GetPrivateMsgKey(userID1, userID2)
	lastTime, exists := lastPrivateMsgTime[key]
	if !exists {
		return true
	}
	return time.Since(lastTime) > 5*time.Minute
}

// UpdateLastPrivateMsgTime 更新最后写入数据库的时间
func UpdateLastPrivateMsgTime(userID1, userID2 int64) {
	key := GetPrivateMsgKey(userID1, userID2)
	lastPrivateMsgTime[key] = time.Now()
}

// FlushPrivateMessagesToDB 将Redis缓存中的消息批量写入MySQL
// 写入完成后清空缓存并更新时间戳
func FlushPrivateMessagesToDB(userID1, userID2 int64) {
	key := GetPrivateMessageCacheKey(userID1, userID2)
	var msgs []Message
	err := utils.GetCache(key, &msgs)
	if err != nil || len(msgs) == 0 {
		return
	}
	for _, msg := range msgs {
		utils.DB.Create(&msg)
	}
	UpdateLastPrivateMsgTime(userID1, userID2)
	utils.DelCache(key)
}

// GetPrivateMessages 获取私聊消息（优先Redis缓存，没有则查MySQL）
func GetPrivateMessages(userID1, userID2 int64) ([]Message, error) {
	var msgs []Message
	key := GetPrivateMessageCacheKey(userID1, userID2)
	err := utils.GetCache(key, &msgs)
	if err == nil && len(msgs) > 0 {
		return msgs, nil
	}
	utils.DB.Where("(form_id = ? AND target_id = ?) OR (form_id = ? AND target_id = ?)",
		userID1, userID2, userID2, userID1).Order("created_at DESC").Limit(20).Find(&msgs)
	return msgs, nil
}

func sendMsg(userID int64, msg []byte) {
	rwLocker.RLock()              //读锁
	node, ok := clientMap[userID] //在map中找用户
	rwLocker.RUnlock()            //解锁
	if ok {
		node.DataQueue <- msg //放入用户发送队列
	}
}

// 用户A发送消息
//     │
//     ▼
// recvProc() ────── 读取 WebSocket 消息
//     │
//     ▼
// broadMsg() ───── 放入 udpsendChan 通道
//     │
//     ▼
// udpSendProc() ─── 从通道取消息，UDP 广播到 192.168.80.255:3000
//     │
//     ▼
// udpRecvProc() ─── 监听 3000 端口，收到消息
//     │
//     ▼
// dispatch() ────── 解析消息，根据 Type 分发
//     │
//     ▼
// sendMsg() ─────── 找到目标用户的 Node
//     │
//     ▼
// sendProc() ────── 从 DataQueue 取消息，WebSocket 发送
//     │
//     ▼
// 用户B收到消息！
