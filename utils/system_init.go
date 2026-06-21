// Package utils （工具包）是 Go 项目中存放 通用工具函数 的地方，类似于"工具箱"。
package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var RedisConn *redis.Client //设定的全局变量，和db差不多

//db是gorm的核心对象，代表了数据库链接的抽象，它用于管理和维护数据库的链接
//同样它也是所有orm对数据库操作的入口

func GetDB() *gorm.DB {
	return DB
}

func InitConfig() {
	//viper这个包是用来读取配置文件来简化代码的一个包
	//告诉viper去哪里拿配置文件
	viper.SetConfigName("app")
	viper.AddConfigPath("config") //配置文件路径
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
	}
	//viper.Get() 是 Viper 配置管理库中最核心、最基础的方法，
	//它的作用是根据指定的键（Key）从配置中获取对应的值（Value）
	fmt.Println("config app", viper.Get("app"))
}

func InitMysql() (*gorm.DB, error) {
	//自定义日志模板，打印sql语句
	newlogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		//创建一个日志输出到控制台log.LstdFlags是标准时间格式
		logger.Config{
			SlowThreshold: time.Second, //设定慢sql阈值，超过1秒算慢sql
			LogLevel:      logger.Info, //日志级别，记录info以上级别日志
			Colorful:      true,        //彩色输出，方便调试
		},
	)

	dsn := viper.GetString("mysql.dsn")
	if dsn == "" {
		return nil, fmt.Errorf("mysql dsn is empty")
	}

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: newlogger})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)           //最大空闲链接数
	sqlDB.SetMaxOpenConns(100)          //最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) //链接最大生命周期

	fmt.Println("MySQL数据库连接成功")
	return DB, nil
}

func InitRedis() (*redis.Client, error) {
	//从配置文件中读取redis地址
	addr := viper.GetString("redis.addr")
	if addr == "" {
		return nil, fmt.Errorf("redis addr is empty")
	}

	//创建redis客户端（链接池）
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
		//一个 Redis 服务可以创建 16 个"数据库"（db0-db15）默认用 db0
		PoolSize:     viper.GetInt("redis.pool_size"),
		MinIdleConns: viper.GetInt("redis.min_idle_conn"),
	})

	//检测是否链接成功
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("redis连接失败: %w", err)
	}

	RedisConn = client //保存全局变量，通过它可以调用redis的方法
	fmt.Println("Redis连接成功")
	return client, nil
}

// 定义频道key，websocket表示频道名称
const (
	PublishKey = "websocket"
)

//redis的发布，订阅功能测试，发布/订阅 = 消息广播
//发布消息到redis上

func Publish(ctx context.Context, channel string, msg string) error {
	//cxt是上下文，用于控制超时与取消，channel则是频道名
	//与线程通信那个无关
	fmt.Println("publish", msg)
	err := RedisConn.Publish(ctx, channel, msg).Err() //这个是redis包的一个函数
	//用于发布消息到指定的频道
	return err
}

// 订阅消息到redis上

func Subscribe(ctx context.Context, channel string) (<-chan *redis.Message, error) {
	pubsub := RedisConn.Subscribe(ctx, channel)
	ch := pubsub.Channel()
	return ch, nil
}





const (
	UserCachePrefix   = "user:"          // 用户缓存前缀，拼接用户缓存 key，如 user:1
	FriendCachePrefix = "friend:"        // 好友缓存前缀
	CacheExpireTime   = 30 * time.Minute // 默认缓存时间，防止缓存无限增长
)

func SetCache(key string, value interface{}) error {
	//检查redis链接
	if RedisConn == nil {
		return fmt.Errorf("redis connection is nil")
	}
	//设定5秒超时防止阻塞
	// 1. context.Background() 创建一个"根上下文"（空的context）
	// 2. WithTimeout() 基于根上下文，创建带超时功能的子上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// 作用：告诉系统，这个操作最多等5秒
	// cancel - 一个函数，用于"提前取消"操作
	defer cancel()
	//go对象序列化JSON，这样方便写入redis
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	//写入redis
	return RedisConn.Set(ctx, key, data, CacheExpireTime).Err()
}


//读取缓存

func GetCache(key string, dest interface{}) error {
	//依旧检查链接
	if RedisConn == nil {
		return fmt.Errorf("redis connection is nil")
	}
	//防止阻塞
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//从缓存获取数据
	val, err := RedisConn.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	//反序列化得到目标对象
	return json.Unmarshal(val, dest)
}

func DelCache(key string) error {
	//删除缓存
	if RedisConn == nil {
		return fmt.Errorf("redis connection is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	
	defer cancel()
	//删除
	return RedisConn.Del(ctx, key).Err()
}

func DelCacheByPattern(pattern string) error {
	//按模式删除，用途： 批量删除，比如 DelCacheByPattern("friend:*") 删除所有好友缓存
	if RedisConn == nil {
		return fmt.Errorf("redis connection is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//扫描所有匹配的key
	iter := RedisConn.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		//逐个删除
		if err := RedisConn.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// 限流常量
const (
	RateLimitMax  = 30              // 单用户每周期最大请求数
	RateLimitSecs = 60              // 限流周期（秒）
)

// RedisRateLimit 基于Redis的请求限流
// 返回 true = 通过，false = 被限流
func RedisRateLimit(key string) bool {
	if RedisConn == nil {
		return true // Redis不可用时放行
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rateKey := "ratelimit:" + key

	count, err := RedisConn.Get(ctx, rateKey).Int()
	if err != nil && err != redis.Nil {
		return true // Redis错误时放行
	}

	if count >= RateLimitMax {
		return false // 被限流
	}

	if count == 0 {
		RedisConn.Set(ctx, rateKey, 1, RateLimitSecs*time.Second)
	} else {
		RedisConn.Incr(ctx, rateKey)
	}

	return true
}

// CheckUserMessageRate 用户消息限流
func CheckUserMessageRate(userID int64) bool {
	return RedisRateLimit(fmt.Sprintf("user:%d", userID))
}

// CheckIPRate IP限流
func CheckIPRate(ip string) bool {
	return RedisRateLimit(fmt.Sprintf("ip:%s", ip))
}

// CheckMessageSize 检查消息大小是否合法
// 返回 true = 合法，false = 过大
func CheckMessageSize(size int64) bool {
	return size <= 1024*1024 // 1MB
}

// CheckImageSize 检查图片大小是否合法
// 返回 true = 合法，false = 过大
func CheckImageSize(size int64) bool {
	return size <= 5*1024*1024 // 5MB
}
