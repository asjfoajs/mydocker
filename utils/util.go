package utils

import (
	"math/rand"
	"time"
)

// RanStringBytes 生成n位随机字符串
func RanStringBytes(n int) string {
	letterBytes := "abcdefghijklmnopqrstuvwxyz1234567890"
	rand.Seed(time.Now().UnixMilli())

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
