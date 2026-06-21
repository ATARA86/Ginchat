// Package utils
package utils

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

//MD5 是一种加密算法，把任意长度的字符串转换成 128位（32个字符） 的固定长度字符串。
// MD5Encode 小写

func MD5Encode(data string) string {
	h := md5.New()//创建哈希对象
	h.Write([]byte(data))//写入要加密的数据
	tempStr := h.Sum(nil)//计算哈希值
	return hex.EncodeToString(tempStr)//转化成16进制字符串
}

// MD5EncodeUpper 大写
func MD5EncodeUpper(data string) string {
	return strings.ToUpper(MD5Encode(data))//调用上面的函数然后转大写
}

//加密

func MakePassword(plainpwd,salt string)string{
	return MD5Encode(plainpwd+salt)//salt是一个随机数
}

//解密

func ValidPassword(plainpwd,salt,password string)bool{
	return MD5Encode(plainpwd+salt)== password
}