package hitratio

import (
	"fmt"
	"math/rand"
	"time"
)

// DataGenerator 提供用于测试的数据生成功能
type DataGenerator struct {
	// 随机数生成器
	rng *rand.Rand

	// 生成数据的配置
	keySpaceSize int     // 键空间大小
	valueSize    int     // 值大小（字节）
	distribution string  // 分布类型：uniform, zipf
	zipfParam    float64 // Zipf 分布参数

	// Zipf 分布生成器
	zipf *rand.Zipf
}

// NewDataGenerator 创建一个新的数据生成器
func NewDataGenerator(keySpaceSize, valueSize int, distribution string, zipfParam float64) *DataGenerator {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	dg := &DataGenerator{
		rng:          rng,
		keySpaceSize: keySpaceSize,
		valueSize:    valueSize,
		distribution: distribution,
		zipfParam:    zipfParam,
	}

	// 如果是 Zipf 分布，初始化 Zipf 生成器
	if distribution == "zipf" {
		dg.zipf = rand.NewZipf(rng, zipfParam, 1.0, uint64(keySpaceSize-1))
	}

	return dg
}

// NextKey 生成下一个键
func (dg *DataGenerator) NextKey() string {
	var keyID uint64

	switch dg.distribution {
	case "uniform":
		keyID = uint64(dg.rng.Intn(dg.keySpaceSize))
	case "zipf":
		keyID = dg.zipf.Uint64()
	default:
		// 默认使用均匀分布
		keyID = uint64(dg.rng.Intn(dg.keySpaceSize))
	}

	return fmt.Sprintf("key:%d", keyID)
}

// NextValue 生成下一个值
func (dg *DataGenerator) NextValue() []byte {
	value := make([]byte, dg.valueSize)
	dg.rng.Read(value)
	return value
}

// GenerateKeyValuePairs 生成指定数量的键值对
func (dg *DataGenerator) GenerateKeyValuePairs(count int) map[string][]byte {
	result := make(map[string][]byte, count)

	for i := 0; i < count; i++ {
		key := dg.NextKey()
		value := dg.NextValue()
		result[key] = value
	}

	return result
}

// GenerateHotKeys 生成热点键
// 返回按热度排序的键列表
func (dg *DataGenerator) GenerateHotKeys(count int) []string {
	// 创建一个专用的 Zipf 生成器，参数更高以产生更集中的热点
	hotZipf := rand.NewZipf(dg.rng, 1.5, 1.0, uint64(count-1))

	// 使用 map 去重
	keyMap := make(map[uint64]struct{})
	for len(keyMap) < count {
		keyMap[hotZipf.Uint64()] = struct{}{}
	}

	// 转换为字符串切片
	keys := make([]string, 0, count)
	for keyID := range keyMap {
		keys = append(keys, fmt.Sprintf("key:%d", keyID))
	}

	return keys
}

// GenerateAccessSequence 生成访问序列
// 参数:
//   - count: 序列长度
//   - readRatio: 读操作比例 (0.0-1.0)
//   - hotKeyRatio: 热点键访问比例 (0.0-1.0)
//   - hotKeys: 热点键列表
func (dg *DataGenerator) GenerateAccessSequence(count int, readRatio, hotKeyRatio float64, hotKeys []string) []AccessOp {
	sequence := make([]AccessOp, count)

	for i := 0; i < count; i++ {
		// 决定操作类型（读/写）
		isRead := dg.rng.Float64() < readRatio

		// 决定是否访问热点键
		useHotKey := dg.rng.Float64() < hotKeyRatio && len(hotKeys) > 0

		var key string
		if useHotKey {
			// 从热点键中选择
			key = hotKeys[dg.rng.Intn(len(hotKeys))]
		} else {
			// 生成普通键
			key = dg.NextKey()
		}

		var value []byte
		if !isRead {
			value = dg.NextValue()
		}

		sequence[i] = AccessOp{
			IsRead: isRead,
			Key:    key,
			Value:  value,
		}
	}

	return sequence
}

// AccessOp 表示一次缓存访问操作
type AccessOp struct {
	IsRead bool   // true 表示读操作，false 表示写操作
	Key    string // 键
	Value  []byte // 值（仅用于写操作）
}
