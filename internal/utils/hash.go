// Package utils 提供HCache内部使用的通用工具函数
// 这些函数无业务含义，可被任何内部模块安全使用
package utils

import (
	"hash/fnv"
	"math/bits"
	"unsafe"
)

// Hash64 使用FNV-1a算法计算字符串的64位哈希值
// 适用于需要高效字符串哈希的场景
func Hash64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// Hash64Bytes 使用FNV-1a算法计算字节切片的64位哈希值
func Hash64Bytes(b []byte) uint64 {
	h := fnv.New64a()
	h.Write(b)
	return h.Sum64()
}

// MurmurHash3 实现MurmurHash3算法的64位变种
// 比标准库的FNV算法更快，但同样具有良好的分布性
// 参考：https://github.com/spaolacci/murmur3
func MurmurHash3(data []byte, seed uint64) uint64 {
	const (
		c1 = uint64(0x87c37b91114253d5)
		c2 = uint64(0x4cf5ad432745937f)
		r1 = 31
		r2 = 27
		m  = uint64(5)
		n  = uint64(0x52dce729)
	)

	h1 := seed
	h2 := seed

	// 处理8字节块
	nblocks := len(data) / 16
	for i := 0; i < nblocks; i++ {
		i16 := i * 16
		k1 := *(*uint64)(unsafe.Pointer(&data[i16]))
		k2 := *(*uint64)(unsafe.Pointer(&data[i16+8]))

		k1 *= c1
		k1 = bits.RotateLeft64(k1, r1)
		k1 *= c2
		h1 ^= k1

		h1 = bits.RotateLeft64(h1, r2)
		h1 = h1*m + n

		k2 *= c2
		k2 = bits.RotateLeft64(k2, r2)
		k2 *= c1
		h2 ^= k2

		h2 = bits.RotateLeft64(h2, r1)
		h2 = h2*m + n
	}

	// 处理剩余字节
	tail := data[nblocks*16:]
	var k1, k2 uint64
	switch len(tail) & 15 {
	case 15:
		k2 ^= uint64(tail[14]) << 48
		fallthrough
	case 14:
		k2 ^= uint64(tail[13]) << 40
		fallthrough
	case 13:
		k2 ^= uint64(tail[12]) << 32
		fallthrough
	case 12:
		k2 ^= uint64(tail[11]) << 24
		fallthrough
	case 11:
		k2 ^= uint64(tail[10]) << 16
		fallthrough
	case 10:
		k2 ^= uint64(tail[9]) << 8
		fallthrough
	case 9:
		k2 ^= uint64(tail[8])
		k2 *= c2
		k2 = bits.RotateLeft64(k2, r2)
		k2 *= c1
		h2 ^= k2
		fallthrough
	case 8:
		k1 ^= uint64(tail[7]) << 56
		fallthrough
	case 7:
		k1 ^= uint64(tail[6]) << 48
		fallthrough
	case 6:
		k1 ^= uint64(tail[5]) << 40
		fallthrough
	case 5:
		k1 ^= uint64(tail[4]) << 32
		fallthrough
	case 4:
		k1 ^= uint64(tail[3]) << 24
		fallthrough
	case 3:
		k1 ^= uint64(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint64(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint64(tail[0])
		k1 *= c1
		k1 = bits.RotateLeft64(k1, r1)
		k1 *= c2
		h1 ^= k1
	}

	// 最终混合
	h1 ^= uint64(len(data))
	h2 ^= uint64(len(data))

	h1 += h2
	h2 += h1

	h1 = fmix64(h1)
	h2 = fmix64(h2)

	h1 += h2
	// h2 += h1 // 不需要，我们只返回h1

	return h1
}

// fmix64 是MurmurHash3的64位混合函数
func fmix64(k uint64) uint64 {
	k ^= k >> 33
	k *= 0xff51afd7ed558ccd
	k ^= k >> 33
	k *= 0xc4ceb9fe1a85ec53
	k ^= k >> 33
	return k
}

// XXHash64 实现XXHash算法的64位变种
// 这是一个极快的非加密哈希算法，适用于大量数据的哈希
// 参考：https://github.com/cespare/xxhash
func XXHash64(data []byte, seed uint64) uint64 {
	const (
		prime1 = 11400714785074694791
		prime2 = 14029467366897019727
		prime3 = 1609587929392839161
		prime4 = 9650029242287828579
		prime5 = 2870177450012600261
	)

	var h uint64
	n := len(data)

	if n >= 32 {
		v1 := seed + prime1 + prime2
		v2 := seed + prime2
		v3 := seed
		v4 := seed - prime1

		// 处理32字节块
		for len(data) >= 32 {
			v1 = round(v1, *(*uint64)(unsafe.Pointer(&data[0])))
			v2 = round(v2, *(*uint64)(unsafe.Pointer(&data[8])))
			v3 = round(v3, *(*uint64)(unsafe.Pointer(&data[16])))
			v4 = round(v4, *(*uint64)(unsafe.Pointer(&data[24])))
			data = data[32:]
		}

		h = bits.RotateLeft64(v1, 1) + bits.RotateLeft64(v2, 7) + bits.RotateLeft64(v3, 12) + bits.RotateLeft64(v4, 18)
		h = mergeRound(h, v1)
		h = mergeRound(h, v2)
		h = mergeRound(h, v3)
		h = mergeRound(h, v4)
	} else {
		h = seed + prime5
	}

	h += uint64(n)

	// 处理剩余字节
	for len(data) >= 8 {
		k := *(*uint64)(unsafe.Pointer(&data[0]))
		h ^= round(0, k)
		h = bits.RotateLeft64(h, 27)*prime1 + prime4
		data = data[8:]
	}

	if len(data) >= 4 {
		h ^= uint64(*(*uint32)(unsafe.Pointer(&data[0]))) * prime1
		h = bits.RotateLeft64(h, 23)*prime2 + prime3
		data = data[4:]
	}

	for _, b := range data {
		h ^= uint64(b) * prime5
		h = bits.RotateLeft64(h, 11) * prime1
	}

	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	h *= prime3
	h ^= h >> 32

	return h
}

// round 是XXHash64的轮函数
func round(acc, input uint64) uint64 {
	acc += input * 0x9e3779b97f4a7c15
	acc = bits.RotateLeft64(acc, 31)
	acc *= 0xbf58476d1ce4e5b9
	return acc
}

// mergeRound 是XXHash64的合并轮函数
func mergeRound(acc, val uint64) uint64 {
	val = round(0, val)
	acc ^= val
	acc = acc*0x9e3779b97f4a7c15 + 0x52dce729
	return acc
}

// JumpConsistentHash 实现Jump Consistent Hash算法
// 这是一个快速、空间效率高的一致性哈希算法，适用于分片
// 参考：http://arxiv.org/abs/1406.2294
func JumpConsistentHash(key uint64, numBuckets int) int {
	var b, j int64 = -1, 0
	for j < int64(numBuckets) {
		b = j
		key = key*2862933555777941757 + 1
		j = int64(float64(b+1) * (float64(int64(1)<<31) / float64((key>>33)+1)))
	}
	return int(b)
}

// HashUint64 计算uint64类型值的哈希
// 使用简单但高效的混合函数
func HashUint64(key uint64) uint64 {
	key = (^key) + (key << 21) // key = (key << 21) - key - 1
	key = key ^ (key >> 24)
	key = (key + (key << 3)) + (key << 8) // key * 265
	key = key ^ (key >> 14)
	key = (key + (key << 2)) + (key << 4) // key * 21
	key = key ^ (key >> 28)
	key = key + (key << 31)
	return key
}

// NextPowerOfTwo 返回大于等于n的最小2的幂
// 用于优化取模运算（使用位与替代取模）
func NextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return n
}

// IsPowerOfTwo 检查n是否为2的幂
func IsPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// FastRangeUint32 将32位哈希值映射到[0, n)范围内
// 避免使用取模操作，提高性能
func FastRangeUint32(hash uint32, n uint32) uint32 {
	return uint32((uint64(hash) * uint64(n)) >> 32)
}

// FastRangeUint64 将64位哈希值映射到[0, n)范围内
func FastRangeUint64(hash, n uint64) uint64 {
	// 使用128位乘法的近似模拟
	hi, lo := bits.Mul64(hash, n)
	if lo < n {
		rem := n - 1 - lo
		if rem > hi {
			return hi
		}
		return rem
	}
	return hi
}
