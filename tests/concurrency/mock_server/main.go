package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/noobtrump/hcache/pkg/cache"
)

// 服务器配置
var (
	port        = flag.Int("port", 8080, "HTTP服务器端口")
	cacheSize   = flag.Int("cache-size", 100000, "缓存最大条目数")
	policy      = flag.String("policy", "lru", "缓存淘汰策略 (lru, lfu, fifo, random)")
	ttl         = flag.Duration("ttl", 5*time.Minute, "缓存默认TTL")
	valueSize   = flag.Int("value-size", 1024, "默认值大小（字节）")
	logRequests = flag.Bool("log", false, "是否记录每个请求")
)

// 全局缓存实例
var cacheInstance cache.ICache

// 缓存项请求结构
type CacheRequest struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value,omitempty"`
	TTL   int64       `json:"ttl,omitempty"` // TTL（秒）
}

// 缓存响应结构
type CacheResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message,omitempty"`
	Value   interface{}  `json:"value,omitempty"`
	Exists  bool         `json:"exists,omitempty"`
	Stats   *cache.Stats `json:"stats,omitempty"`
}

func main() {
	flag.Parse()

	// 初始化缓存
	var err error
	cacheInstance = cache.NewMockCache("mock-server-cache", *cacheSize, *policy)
	if err != nil {
		log.Fatalf("初始化缓存失败: %v", err)
	}
	defer cacheInstance.Close()

	// 设置路由
	http.HandleFunc("/cache/get", handleGet)
	http.HandleFunc("/cache/set", handleSet)
	http.HandleFunc("/cache/delete", handleDelete)
	http.HandleFunc("/cache/stats", handleStats)
	http.HandleFunc("/cache/clear", handleClear)

	// 添加健康检查端点
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 启动服务器
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("启动 HTTP 服务器，监听 %s", addr)
	log.Printf("缓存配置: 大小=%d, 策略=%s, TTL=%s", *cacheSize, *policy, *ttl)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// 处理 GET 请求
func handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "仅支持 GET 方法", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "缺少 'key' 参数", http.StatusBadRequest)
		return
	}

	if *logRequests {
		log.Printf("GET 请求: key=%s", key)
	}

	ctx := context.Background()
	value, exists, err := cacheInstance.Get(ctx, key)

	response := CacheResponse{
		Success: err == nil,
		Exists:  exists,
	}

	if err != nil {
		response.Message = fmt.Sprintf("获取缓存失败: %v", err)
	} else if !exists {
		response.Message = "键不存在"
	} else {
		response.Value = value
	}

	sendJSONResponse(w, response)
}

// 处理 SET 请求
func handleSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	var req CacheRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("无效的 JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		http.Error(w, "缺少 'key' 字段", http.StatusBadRequest)
		return
	}

	// 如果未提供值，生成指定大小的随机值
	var value interface{} = req.Value
	if value == nil {
		value = make([]byte, *valueSize)
	}

	// 设置 TTL
	ttlDuration := *ttl
	if req.TTL > 0 {
		ttlDuration = time.Duration(req.TTL) * time.Second
	}

	if *logRequests {
		log.Printf("SET 请求: key=%s, ttl=%s", req.Key, ttlDuration)
	}

	ctx := context.Background()
	err := cacheInstance.Set(ctx, req.Key, value, ttlDuration)

	response := CacheResponse{
		Success: err == nil,
	}

	if err != nil {
		response.Message = fmt.Sprintf("设置缓存失败: %v", err)
	} else {
		response.Message = "设置成功"
	}

	sendJSONResponse(w, response)
}

// 处理 DELETE 请求
func handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodGet {
		http.Error(w, "仅支持 DELETE 或 GET 方法", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "缺少 'key' 参数", http.StatusBadRequest)
		return
	}

	if *logRequests {
		log.Printf("DELETE 请求: key=%s", key)
	}

	ctx := context.Background()
	deleted, err := cacheInstance.Delete(ctx, key)

	response := CacheResponse{
		Success: err == nil,
		Exists:  deleted,
	}

	if err != nil {
		response.Message = fmt.Sprintf("删除缓存失败: %v", err)
	} else if deleted {
		response.Message = "删除成功"
	} else {
		response.Message = "键不存在"
	}

	sendJSONResponse(w, response)
}

// 处理 STATS 请求
func handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "仅支持 GET 方法", http.StatusMethodNotAllowed)
		return
	}

	if *logRequests {
		log.Printf("STATS 请求")
	}

	ctx := context.Background()
	stats, err := cacheInstance.Stats(ctx)

	response := CacheResponse{
		Success: err == nil,
	}

	if err != nil {
		response.Message = fmt.Sprintf("获取统计信息失败: %v", err)
	} else {
		response.Stats = stats
	}

	sendJSONResponse(w, response)
}

// 处理 CLEAR 请求
func handleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "仅支持 POST 或 GET 方法", http.StatusMethodNotAllowed)
		return
	}

	if *logRequests {
		log.Printf("CLEAR 请求")
	}

	ctx := context.Background()
	err := cacheInstance.Clear(ctx)

	response := CacheResponse{
		Success: err == nil,
	}

	if err != nil {
		response.Message = fmt.Sprintf("清空缓存失败: %v", err)
	} else {
		response.Message = "缓存已清空"
	}

	sendJSONResponse(w, response)
}

// 发送 JSON 响应
func sendJSONResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}
