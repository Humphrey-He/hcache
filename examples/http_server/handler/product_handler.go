// Package handler provides HTTP request handlers for the product API.
// It implements the presentation layer of the application, handling HTTP requests
// and responses while delegating business logic to the service layer.
//
// Package handler 提供产品API的HTTP请求处理程序。
// 它实现了应用程序的表示层，处理HTTP请求和响应，同时将业务逻辑委托给服务层。
package handler

import (
	"net/http"
	"strconv"

	"github.com/Humphrey-He/hcache/examples/http_server/model"
	"github.com/Humphrey-He/hcache/examples/http_server/service"
	"github.com/gin-gonic/gin"
)

// ProductHandler handles HTTP requests for products.
// It acts as an adapter between the HTTP layer and the service layer,
// translating HTTP requests into service calls and formatting responses.
//
// ProductHandler 处理产品的HTTP请求。
// 它充当HTTP层和服务层之间的适配器，将HTTP请求转换为服务调用并格式化响应。
type ProductHandler struct {
	service *service.ProductService
}

// NewProductHandler creates a new product handler with the given service.
//
// Parameters:
//   - service: The product service to use for business logic
//
// Returns:
//   - *ProductHandler: A new product handler instance
//
// NewProductHandler 使用给定的服务创建一个新的产品处理程序。
//
// 参数:
//   - service: 用于业务逻辑的产品服务
//
// 返回:
//   - *ProductHandler: 一个新的产品处理程序实例
func NewProductHandler(service *service.ProductService) *ProductHandler {
	return &ProductHandler{
		service: service,
	}
}

// GetProduct handles GET requests for a single product.
// It extracts the product ID from the URL path parameter and returns
// the product as JSON if found.
//
// Parameters:
//   - c: The Gin context containing the HTTP request and response
//
// GetProduct 处理获取单个产品的GET请求。
// 它从URL路径参数中提取产品ID，并在找到时以JSON格式返回产品。
//
// 参数:
//   - c: 包含HTTP请求和响应的Gin上下文
func (h *ProductHandler) GetProduct(c *gin.Context) {
	// Get product ID from URL
	// 从URL获取产品ID
	id := c.Param("id")

	// Get product from service
	// 从服务获取产品
	product, err := h.service.GetProductByStringID(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

// ListProducts handles GET requests for a list of products.
// It extracts filter criteria from query parameters and returns
// a filtered list of products as JSON.
//
// Parameters:
//   - c: The Gin context containing the HTTP request and response
//
// ListProducts 处理获取产品列表的GET请求。
// 它从查询参数中提取过滤条件，并以JSON格式返回过滤后的产品列表。
//
// 参数:
//   - c: 包含HTTP请求和响应的Gin上下文
func (h *ProductHandler) ListProducts(c *gin.Context) {
	// Parse filter from query parameters
	// 从查询参数解析过滤条件
	var filter model.ProductFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get products from service
	// 从服务获取产品
	products, err := h.service.GetProducts(c, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}

// CreateProduct handles POST requests to create a new product.
// It parses the product data from the request body and returns
// the created product as JSON.
//
// Parameters:
//   - c: The Gin context containing the HTTP request and response
//
// CreateProduct 处理创建新产品的POST请求。
// 它从请求体解析产品数据，并以JSON格式返回创建的产品。
//
// 参数:
//   - c: 包含HTTP请求和响应的Gin上下文
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	// Parse product from request body
	// 从请求体解析产品
	var product model.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create product using service
	// 使用服务创建产品
	newProduct, err := h.service.CreateProduct(c, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newProduct)
}

// UpdateProduct handles PUT requests to update an existing product.
// It extracts the product ID from the URL path parameter, parses the
// updated product data from the request body, and returns the
// updated product as JSON.
//
// Parameters:
//   - c: The Gin context containing the HTTP request and response
//
// UpdateProduct 处理更新现有产品的PUT请求。
// 它从URL路径参数中提取产品ID，从请求体解析更新的产品数据，
// 并以JSON格式返回更新后的产品。
//
// 参数:
//   - c: 包含HTTP请求和响应的Gin上下文
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	// Get product ID from URL
	// 从URL获取产品ID
	idStr := c.Param("id")

	// Parse product from request body
	// 从请求体解析产品
	var product model.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert ID to int
	// 将ID转换为整数
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Update product using service
	// 使用服务更新产品
	updatedProduct, err := h.service.UpdateProduct(c, id, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedProduct)
}

// DeleteProduct handles DELETE requests to delete a product.
// It extracts the product ID from the URL path parameter and
// returns a success message as JSON.
//
// Parameters:
//   - c: The Gin context containing the HTTP request and response
//
// DeleteProduct 处理删除产品的DELETE请求。
// 它从URL路径参数中提取产品ID，并以JSON格式返回成功消息。
//
// 参数:
//   - c: 包含HTTP请求和响应的Gin上下文
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	// Get product ID from URL
	// 从URL获取产品ID
	idStr := c.Param("id")

	// Convert ID to int
	// 将ID转换为整数
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Delete product using service
	// 使用服务删除产品
	err = h.service.DeleteProduct(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}
