package utils

const (
	// BitSize 表示一个字节中的比特位数量
	BitSize = 8
)

// BitMap 使用 byte 切片来表示位图
type BitMap struct {
	Bits []byte `json:"bits"`
}

// NewBitmap 创建一个新的位图实例，size 表示位图的大小（比特位数）
func NewBitmap(size uint) *BitMap {
	return &BitMap{
		Bits: make([]byte, (size+BitSize-1)/BitSize),
	}
}

// Set 设置位图中指定位置的比特位为 1 (| 1)
func (b *BitMap) Set(index int) {
	byteIndex := index / BitSize
	bitIndex := index % BitSize
	b.Bits[byteIndex] |= 1 << uint(bitIndex)
}

// Clear 清除位图中指定位置的比特位，将其设置为 0(不直接&0，因为会影响后面，而是与非)
// 就是1&^1 = 0,1&^0 = 1,0&^1 = 1,0&^0 = 1,就是并上 （&1111011111111）
// 先取反，再与1进行与运算，就是清除了
func (b *BitMap) Clear(index int) {
	byteIndex := index / BitSize
	bitIndex := index % BitSize
	b.Bits[byteIndex] &^= 1 << uint(bitIndex)
}

// IsClear 测试位图中指定位置的比特位是否为 0 (0&1 =0,1&1=1)
func (b *BitMap) IsClear(index int) bool {
	byteIndex := index / BitSize
	bitIndex := index % BitSize
	return b.Bits[byteIndex]&(1<<uint(bitIndex)) == 0
}

// Size 返回位图的大小（比特位数）
func (b *BitMap) Size() int {
	return len(b.Bits) * BitSize
}
