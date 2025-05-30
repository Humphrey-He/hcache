// Package service implements the business logic for the product API.
// It serves as an intermediary between the HTTP handlers and the data storage,
// providing caching functionality to improve performance and reduce database load.
//
// Package service 实现产品API的业务逻辑。
// 它作为HTTP处理程序和数据存储之间的中介，提供缓存功能以提高性能并减少数据库负载。
package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/noobtrump/hcache/examples/http_server/model"
	"github.com/noobtrump/hcache/examples/http_server/storage"
	"github.com/noobtrump/hcache/pkg/cache"
)

// ProductService handles product business logic with caching.
// It implements a cache-aside pattern where data is first looked up in the cache,
// and if not found, retrieved from the storage and then added to the cache.
//
// ProductService 处理带有缓存的产品业务逻辑。
// 它实现了缓存旁路模式，首先在缓存中查找数据，如果未找到，则从存储中检索数据并添加到缓存中。
type ProductService struct {
	cache   cache.ICache            // Cache instance / 缓存实例
	storage *storage.ProductStorage // Data storage / 数据存储
}

// NewProductService creates a new product service with cache and storage.
//
// Parameters:
//   - cache: The cache instance to use
//   - storage: The product storage to use for persistence
//
// Returns:
//   - *ProductService: A new product service instance
//
// NewProductService 创建一个带有缓存和存储的新产品服务。
//
// 参数:
//   - cache: 要使用的缓存实例
//   - storage: 用于持久化的产品存储
//
// 返回:
//   - *ProductService: 一个新的产品服务实例
func NewProductService(cache cache.ICache, storage *storage.ProductStorage) *ProductService {
	return &ProductService{
		cache:   cache,
		storage: storage,
	}
}

// GetProduct retrieves a product by ID, using cache if available.
// It implements the cache-aside pattern: first check cache, if miss then
// fetch from storage and update cache for future requests.
//
// Parameters:
//   - ctx: The context for the operation
//   - id: The product ID to retrieve
//
// Returns:
//   - model.Product: The retrieved product
//   - error: An error if the product could not be retrieved
//
// GetProduct 通过ID检索产品，如果可用则使用缓存。
// 它实现了缓存旁路模式：首先检查缓存，如果未命中，则从存储中获取并更新缓存以供将来请求使用。
//
// 参数:
//   - ctx: 操作的上下文
//   - id: 要检索的产品ID
//
// 返回:
//   - model.Product: 检索到的产品
//   - error: 如果无法检索产品则返回错误
func (s *ProductService) GetProduct(ctx context.Context, id int) (model.Product, error) {
	// Generate cache key
	// 生成缓存键
	cacheKey := fmt.Sprintf("product:%d", id)

	// Try to get from cache first
	// 首先尝试从缓存获取
	value, exists, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		// Log cache error but continue to fetch from storage
		// 记录缓存错误但继续从存储中获取
		log.Printf("Cache error: %v", err)
	} else if exists {
		// Cache hit
		// 缓存命中
		log.Printf("[CACHE HIT] Product ID: %d", id)
		return value.(model.Product), nil
	}

	// Cache miss, get from storage
	// 缓存未命中，从存储中获取
	log.Printf("[CACHE MISS] Product ID: %d", id)
	product, err := s.storage.GetProduct(ctx, id)
	if err != nil {
		return model.Product{}, err
	}

	// Store in cache for future requests
	// 存储在缓存中以供将来请求使用
	err = s.cache.Set(ctx, cacheKey, product, 5*time.Minute)
	if err != nil {
		// Log cache error but return the product anyway
		// 记录缓存错误但仍然返回产品
		log.Printf("Failed to cache product: %v", err)
	}

	return product, nil
}

// GetProducts retrieves products based on filter criteria.
// It caches the filtered product lists with a shorter TTL than individual products
// since lists are more likely to change frequently.
//
// Parameters:
//   - ctx: The context for the operation
//   - filter: The filter criteria for the products
//
// Returns:
//   - model.ProductList: The list of products matching the filter
//   - error: An error if the products could not be retrieved
//
// GetProducts 根据过滤条件检索产品。
// 它缓存过滤后的产品列表，其TTL比单个产品短，因为列表更可能频繁变化。
//
// 参数:
//   - ctx: 操作的上下文
//   - filter: 产品的过滤条件
//
// 返回:
//   - model.ProductList: 匹配过滤条件的产品列表
//   - error: 如果无法检索产品则返回错误
func (s *ProductService) GetProducts(ctx context.Context, filter model.ProductFilter) (model.ProductList, error) {
	// Generate cache key based on filter parameters
	// 基于过滤参数生成缓存键
	cacheKey := fmt.Sprintf("products:category=%s:min=%.2f:max=%.2f:page=%d:size=%d",
		filter.Category, filter.MinPrice, filter.MaxPrice, filter.Page, filter.PageSize)

	// Try to get from cache first
	// 首先尝试从缓存获取
	value, exists, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		// Log cache error but continue to fetch from storage
		// 记录缓存错误但继续从存储中获取
		log.Printf("Cache error: %v", err)
	} else if exists {
		// Cache hit
		// 缓存命中
		log.Printf("[CACHE HIT] Products list: %s", cacheKey)
		return value.(model.ProductList), nil
	}

	// Cache miss, get from storage
	// 缓存未命中，从存储中获取
	log.Printf("[CACHE MISS] Products list: %s", cacheKey)
	productList, err := s.storage.GetProducts(ctx, filter)
	if err != nil {
		return model.ProductList{}, err
	}

	// Store in cache for future requests (shorter TTL for lists)
	// 存储在缓存中以供将来请求使用（列表的TTL较短）
	err = s.cache.Set(ctx, cacheKey, productList, 2*time.Minute)
	if err != nil {
		// Log cache error but return the products anyway
		// 记录缓存错误但仍然返回产品
		log.Printf("Failed to cache product list: %v", err)
	}

	return productList, nil
}

// CreateProduct creates a new product.
// After creating the product in storage, it caches the new product and
// invalidates any cached product lists to ensure consistency.
//
// Parameters:
//   - ctx: The context for the operation
//   - product: The product to create
//
// Returns:
//   - model.Product: The created product with assigned ID
//   - error: An error if the product could not be created
//
// CreateProduct 创建一个新产品。
// 在存储中创建产品后，它会缓存新产品并使任何缓存的产品列表失效以确保一致性。
//
// 参数:
//   - ctx: 操作的上下文
//   - product: 要创建的产品
//
// 返回:
//   - model.Product: 创建的产品（带有分配的ID）
//   - error: 如果无法创建产品则返回错误
func (s *ProductService) CreateProduct(ctx context.Context, product model.Product) (model.Product, error) {
	// Create in storage
	// 在存储中创建
	newProduct, err := s.storage.CreateProduct(ctx, product)
	if err != nil {
		return model.Product{}, err
	}

	// Cache the new product
	// 缓存新产品
	cacheKey := fmt.Sprintf("product:%d", newProduct.ID)
	err = s.cache.Set(ctx, cacheKey, newProduct, 5*time.Minute)
	if err != nil {
		// Log cache error but return the product anyway
		// 记录缓存错误但仍然返回产品
		log.Printf("Failed to cache new product: %v", err)
	}

	// Invalidate any product lists in cache
	// In a real system, you might want to be more selective about which lists to invalidate
	// 使缓存中的任何产品列表失效
	// 在实际系统中，您可能希望更有选择性地决定使哪些列表失效
	s.invalidateProductLists(ctx)

	return newProduct, nil
}

// UpdateProduct updates an existing product.
// After updating the product in storage, it updates the cache and
// invalidates any cached product lists to ensure consistency.
//
// Parameters:
//   - ctx: The context for the operation
//   - id: The ID of the product to update
//   - product: The updated product data
//
// Returns:
//   - model.Product: The updated product
//   - error: An error if the product could not be updated
//
// UpdateProduct 更新现有产品。
// 在存储中更新产品后，它会更新缓存并使任何缓存的产品列表失效以确保一致性。
//
// 参数:
//   - ctx: 操作的上下文
//   - id: 要更新的产品的ID
//   - product: 更新的产品数据
//
// 返回:
//   - model.Product: 更新后的产品
//   - error: 如果无法更新产品则返回错误
func (s *ProductService) UpdateProduct(ctx context.Context, id int, product model.Product) (model.Product, error) {
	// Update in storage
	// 在存储中更新
	updatedProduct, err := s.storage.UpdateProduct(ctx, id, product)
	if err != nil {
		return model.Product{}, err
	}

	// Update in cache
	// 在缓存中更新
	cacheKey := fmt.Sprintf("product:%d", id)
	err = s.cache.Set(ctx, cacheKey, updatedProduct, 5*time.Minute)
	if err != nil {
		// Log cache error but return the product anyway
		// 记录缓存错误但仍然返回产品
		log.Printf("Failed to update product in cache: %v", err)
	}

	// Invalidate any product lists in cache
	// 使缓存中的任何产品列表失效
	s.invalidateProductLists(ctx)

	return updatedProduct, nil
}

// DeleteProduct deletes a product by ID.
// After deleting the product from storage, it removes the product from cache and
// invalidates any cached product lists to ensure consistency.
//
// Parameters:
//   - ctx: The context for the operation
//   - id: The ID of the product to delete
//
// Returns:
//   - error: An error if the product could not be deleted
//
// DeleteProduct 通过ID删除产品。
// 从存储中删除产品后，它会从缓存中移除产品并使任何缓存的产品列表失效以确保一致性。
//
// 参数:
//   - ctx: 操作的上下文
//   - id: 要删除的产品的ID
//
// 返回:
//   - error: 如果无法删除产品则返回错误
func (s *ProductService) DeleteProduct(ctx context.Context, id int) error {
	// Delete from storage
	// 从存储中删除
	err := s.storage.DeleteProduct(ctx, id)
	if err != nil {
		return err
	}

	// Delete from cache
	// 从缓存中删除
	cacheKey := fmt.Sprintf("product:%d", id)
	_, err = s.cache.Delete(ctx, cacheKey)
	if err != nil {
		// Log cache error but continue
		// 记录缓存错误但继续
		log.Printf("Failed to delete product from cache: %v", err)
	}

	// Invalidate any product lists in cache
	// 使缓存中的任何产品列表失效
	s.invalidateProductLists(ctx)

	return nil
}

// PreloadPopularProducts preloads popular products into the cache.
// This is typically called during application startup to warm the cache
// with frequently accessed products, improving initial response times.
//
// Parameters:
//   - ctx: The context for the operation
//
// Returns:
//   - error: An error if the preloading failed
//
// PreloadPopularProducts 预加载热门产品到缓存中。
// 这通常在应用程序启动期间调用，以便用经常访问的产品预热缓存，从而提高初始响应时间。
//
// 参数:
//   - ctx: 操作的上下文
//
// 返回:
//   - error: 如果预加载失败则返回错误
func (s *ProductService) PreloadPopularProducts(ctx context.Context) error {
	log.Println("Preloading popular products into cache...")

	// Get popular product IDs
	// 获取热门产品ID
	ids, err := s.storage.GetPopularProductIDs(ctx)
	if err != nil {
		return err
	}

	// Load each product into cache with a longer TTL than regular requests
	// 将每个产品加载到缓存中，TTL比常规请求更长
	for _, id := range ids {
		product, err := s.storage.GetProduct(ctx, id)
		if err != nil {
			log.Printf("Failed to load product %d: %v", id, err)
			continue
		}

		cacheKey := fmt.Sprintf("product:%d", id)
		err = s.cache.Set(ctx, cacheKey, product, 10*time.Minute)
		if err != nil {
			log.Printf("Failed to cache popular product %d: %v", id, err)
		} else {
			log.Printf("Preloaded product %d: %s", id, product.Name)
		}
	}

	log.Printf("Preloaded %d popular products", len(ids))
	return nil
}

// invalidateProductLists removes all product list entries from the cache.
// This is a helper method called after product mutations (create/update/delete)
// to ensure cache consistency.
//
// Parameters:
//   - ctx: The context for the operation
//
// invalidateProductLists 从缓存中移除所有产品列表条目。
// 这是在产品变更（创建/更新/删除）后调用的辅助方法，以确保缓存一致性。
//
// 参数:
//   - ctx: 操作的上下文
func (s *ProductService) invalidateProductLists(ctx context.Context) {
	// In a real system, you might want to use a pattern-based deletion
	// or maintain a registry of list cache keys to invalidate
	// For this example, we'll just log that we would do this
	// 在实际系统中，您可能希望使用基于模式的删除
	// 或维护一个要使其失效的列表缓存键的注册表
	// 对于此示例，我们只记录我们会这样做
	log.Println("Invalidating product lists in cache")

	// For a real implementation, you might do something like:
	// keys, _ := s.cache.Keys(ctx)
	// for _, key := range keys {
	//     if strings.HasPrefix(key, "products:") {
	//         s.cache.Delete(ctx, key)
	//     }
	// }
}

// GetProductByStringID is a convenience method that converts a string ID to int
// and then retrieves the product.
//
// Parameters:
//   - ctx: The context for the operation
//   - idStr: The product ID as a string
//
// Returns:
//   - model.Product: The retrieved product
//   - error: An error if the product could not be retrieved or the ID is invalid
//
// GetProductByStringID 是一个便捷方法，它将字符串ID转换为整数，然后检索产品。
//
// 参数:
//   - ctx: 操作的上下文
//   - idStr: 作为字符串的产品ID
//
// 返回:
//   - model.Product: 检索到的产品
//   - error: 如果无法检索产品或ID无效则返回错误
func (s *ProductService) GetProductByStringID(ctx context.Context, idStr string) (model.Product, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return model.Product{}, fmt.Errorf("invalid product ID: %s", idStr)
	}
	return s.GetProduct(ctx, id)
}
