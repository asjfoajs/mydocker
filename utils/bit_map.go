package utils

import (
	"encoding/base64"
	"encoding/json"
)

const (
	// BitSize 表示一个字节中的比特位数量
	BitSize = 8
)

// BitMap 使用 byte 切片来表示位图
type BitMap struct {
	bits []byte
}

// NewBitmap 创建一个新的位图实例，size 表示位图的大小（比特位数）
func NewBitmap(size uint) *BitMap {
	return &BitMap{
		bits: make([]byte, (size+BitSize-1)/BitSize),
	}
}

// Set 设置位图中指定位置的比特位为 1 (| 1)
func (b *BitMap) Set(index int) {
	byteIndex := index / BitSize
	bitIndex := index % BitSize
	b.bits[byteIndex] |= 1 << uint(bitIndex)
}

// Clear 清除位图中指定位置的比特位，将其设置为 0(不直接&0，因为会影响后面，而是与非)
// 就是1&^1 = 0,1&^0 = 1,0&^1 = 1,0&^0 = 1,就是并上 （&1111011111111）
// 先取反，再与1进行与运算，就是清除了
func (b *BitMap) Clear(index int) {
	byteIndex := index / BitSize
	bitIndex := index % BitSize
	b.bits[byteIndex] &^= 1 << uint(bitIndex)
}

// IsClear 测试位图中指定位置的比特位是否为 0 (0&1 =0,1&1=1)
func (b *BitMap) IsClear(index int) bool {
	byteIndex := index / BitSize
	bitIndex := index % BitSize
	return b.bits[byteIndex]&(1<<uint(bitIndex)) == 0
}

// Size 返回位图的大小（比特位数）
func (b *BitMap) Size() int {
	return len(b.bits) * BitSize
}

// MarshalJSON 实现 json.Marshaler 接口
func (bm *BitMap) MarshalJSON() ([]byte, error) {
	// 将字节切片转换为 base64 编码的字符串，便于 JSON 序列化
	return json.Marshal(base64.StdEncoding.EncodeToString(bm.bits))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (bm *BitMap) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	// 将 base64 编码的字符串解码回字节切片
	bytes, err := base64.StdEncoding.DecodeString(str) // 假设 Base64Decode 是一个将 base64 字符串转换为字节切片的函数
	if err != nil {
		return err
	}
	// 将解码后的字节切片赋值给 bm.bits
	bm.bits = bytes
	return nil
}
