// Package main implements an HTTP server example that demonstrates how to use HCache
// in a web application. It shows proper cache initialization, integration with
// a Gin web server, and common caching patterns for a REST API.
//
// Package main 实现了一个HTTP服务器示例，展示了如何在Web应用程序中使用HCache。
// 它演示了正确的缓存初始化、与Gin Web服务器的集成，以及REST API的常见缓存模式。
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/hcache/examples/http_server/handler"
	"github.com/yourusername/hcache/examples/http_server/middleware"
	"github.com/yourusername/hcache/examples/http_server/service"
	"github.com/yourusername/hcache/examples/http_server/storage"
	"github.com/yourusername/hcache/pkg/cache"
)

// main is the entry point for the HTTP server example.
// It initializes the cache, sets up the service layers,
// configures the HTTP routes, and starts the server.
//
// main 是HTTP服务器示例的入口点。
// 它初始化缓存，设置服务层，配置HTTP路由，并启动服务器。
func main() {
	// Parse command line flags
	// 解析命令行参数
	port := flag.String("port", "8080", "HTTP server port")
	cacheSize := flag.Int("cache-size", 1000, "Maximum number of cache entries")
	ttl := flag.Duration("ttl", 1*time.Minute, "Default TTL for cache entries")
	shards := flag.Int("shards", 16, "Number of cache shards")
	flag.Parse()

	// Initialize cache with options
	// 使用选项初始化缓存
	cacheInstance, err := cache.NewWithOptions("product-cache",
		cache.WithMaxEntryCount(*cacheSize),
		cache.WithTTL(*ttl),
		cache.WithShards(*shards),
		cache.WithMetricsEnabled(true),
	)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheInstance.Close()

	// Initialize storage layer (simulates a database)
	// 初始化存储层（模拟数据库）
	productStorage := storage.NewProductStorage()

	// Initialize service layer with cache and storage
	// 使用缓存和存储初始化服务层
	productService := service.NewProductService(cacheInstance, productStorage)

	// Preload some products into the cache for better initial performance
	// 预加载一些产品到缓存中以提高初始性能
	ctx := context.Background()
	if err := productService.PreloadPopularProducts(ctx); err != nil {
		log.Printf("Warning: Failed to preload products: %v", err)
	}

	// Setup Gin router
	// 设置Gin路由器
	router := gin.Default()

	// Register middleware for request logging and cache metrics
	// 注册请求日志和缓存指标的中间件
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CacheMetrics(cacheInstance))

	// Register product API handlers
	// 注册产品API处理程序
	productHandler := handler.NewProductHandler(productService)
	router.GET("/products/:id", productHandler.GetProduct)
	router.GET("/products", productHandler.ListProducts)
	router.POST("/products", productHandler.CreateProduct)
	router.PUT("/products/:id", productHandler.UpdateProduct)
	router.DELETE("/products/:id", productHandler.DeleteProduct)

	// Add cache stats endpoint for monitoring
	// 添加缓存统计端点用于监控
	router.GET("/cache/stats", func(c *gin.Context) {
		stats, err := cacheInstance.Stats(c)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get cache stats"})
			return
		}
		c.JSON(200, stats)
	})

	// Start HTTP server
	// 启动HTTP服务器
	log.Printf("Starting server on port %s", *port)
	if err := router.Run(":" + *port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
