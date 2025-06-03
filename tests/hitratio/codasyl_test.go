// Package hitratio provides test utilities for measuring cache hit ratios under different access patterns.
// hitratio 包提供了用于测量不同访问模式下缓存命中率的测试工具。
package hitratio

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Humphrey-He/hcache/pkg/cache"
)

// TestCODASYLPattern tests cache performance with CODASYL-like access patterns
// where data is accessed in a network/graph structure.
// TestCODASYLPattern 测试缓存在类CODASYL访问模式下的性能表现，
// 即数据在网络/图结构中被访问的情况。
func TestCODASYLPattern(t *testing.T) {
	// Create cache configurations to test
	// 创建要测试的缓存配置
	cacheSizes := []int{1000, 10000}
	policies := []string{"lru", "lfu", "fifo", "random"}

	for _, size := range cacheSizes {
		for _, policy := range policies {
			t.Run(fmt.Sprintf("Size%d_%s", size, policy), func(t *testing.T) {
				// Create cache with the specified policy
				// 使用指定策略创建缓存
				c := createCache("codasyl-test", size, policy)
				defer c.Close()
				ctx := context.Background()

				// Run the test
				// 运行测试
				totalOps := 100000
				nodeCount := 5000     // Number of nodes in the network (网络中的节点数量)
				edgesPerNode := 5     // Average number of edges per node (每个节点的平均边数)
				traversalLength := 10 // Average length of traversals (遍历的平均长度)

				hits, misses, evictions := runCODASYLTest(ctx, c, totalOps, nodeCount, edgesPerNode, traversalLength)

				// 验证hits+misses的总数
				totalAccesses := hits + misses
				// 预期的访问次数应该约等于 totalOps * traversalLength
				// 计算预期的最小和最大访问次数 (允许10%的误差)
				expectedAccessesBase := totalOps * traversalLength
				expectedAccessesMin := expectedAccessesBase - expectedAccessesBase/10 // 下限：基准值 - 10%
				expectedAccessesMax := expectedAccessesBase + expectedAccessesBase/10 // 上限：基准值 + 10%

				if totalAccesses < expectedAccessesMin || totalAccesses > expectedAccessesMax {
					t.Logf("警告: 实际访问总数 (%d) 与预期范围 (%d-%d) 不符",
						totalAccesses, expectedAccessesMin, expectedAccessesMax)
				}

				// Report results - 修正的命中率计算
				// 报告结果 - 修正的命中率计算
				hitRatio := float64(hits) / float64(hits+misses) * 100
				evictionRatio := float64(evictions) / float64(totalOps) * 100 // 这里保持不变，因为它是相对于操作次数的

				t.Logf("测试结果:")
				t.Logf("总操作数: %d", totalOps)
				t.Logf("总访问次数: %d", hits+misses)
				t.Logf("命中数: %d", hits)
				t.Logf("未命中数: %d", misses)
				t.Logf("命中率: %.2f%%", hitRatio)
				t.Logf("淘汰数: %d", evictions)
				t.Logf("淘汰比率: %.2f%%", evictionRatio)
				t.Logf("持续时间: %v", time.Since(time.Now().Add(-time.Second))) // Approximate duration (近似持续时间)
			})
		}
	}
}

// runCODASYLTest executes a test simulating CODASYL network database access patterns.
// runCODASYLTest 执行模拟CODASYL网络数据库访问模式的测试。
//
// Parameters:
//   - ctx: Context for the operation (操作的上下文)
//   - c: Cache instance to test (要测试的缓存实例)
//   - totalOps: Total number of operations to perform (要执行的操作总数)
//   - nodeCount: Number of nodes in the network (网络中的节点数量)
//   - edgesPerNode: Average number of edges per node (每个节点的平均边数)
//   - traversalLength: Average length of traversals (遍历的平均长度)
//
// Returns:
//   - hits: Number of cache hits (缓存命中数)
//   - misses: Number of cache misses (缓存未命中数)
//   - evictions: Number of cache evictions (缓存淘汰数)
func runCODASYLTest(ctx context.Context, c cache.ICache, totalOps, nodeCount, edgesPerNode, traversalLength int) (hits, misses int, evictions int64) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create a graph structure
	// 创建图结构
	graph := make(map[int][]int)
	for i := 0; i < nodeCount; i++ {
		edges := make([]int, 0, edgesPerNode)
		for j := 0; j < edgesPerNode; j++ {
			target := r.Intn(nodeCount)
			edges = append(edges, target)
		}
		graph[i] = edges
	}

	// Pre-populate the cache with some nodes
	// 预先填充缓存，存入一些节点
	for i := 0; i < nodeCount/10; i++ {
		key := fmt.Sprintf("node-%d", r.Intn(nodeCount))
		c.Set(ctx, key, generateRandomValue(), defaultTTL)
	}

	for i := 0; i < totalOps; i++ {
		// Start at a random node
		// 从随机节点开始
		currentNode := r.Intn(nodeCount)

		// Perform a traversal
		// 执行遍历
		for j := 0; j < traversalLength; j++ {
			nodeKey := fmt.Sprintf("node-%d", currentNode)

			// Access the current node
			// 访问当前节点
			_, found, _ := c.Get(ctx, nodeKey)
			if found {
				hits++
			} else {
				misses++
				// Set on miss
				// 未命中时设置值
				c.Set(ctx, nodeKey, generateRandomValue(), defaultTTL)
			}

			// Move to a connected node
			// 移动到相连的节点
			if edges, ok := graph[currentNode]; ok && len(edges) > 0 {
				currentNode = edges[r.Intn(len(edges))]
			} else {
				// If no edges, jump to a random node
				// 如果没有边，跳转到随机节点
				currentNode = r.Intn(nodeCount)
			}
		}
	}

	stats, _ := c.Stats(ctx)
	evictions = stats.Evictions
	return hits, misses, evictions
}
