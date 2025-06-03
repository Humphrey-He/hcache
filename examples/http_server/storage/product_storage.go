package storage

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/Humphrey-He/hcache/examples/http_server/model"
)

// ProductStorage simulates a database for product data
type ProductStorage struct {
	products      map[int]model.Product
	mu            sync.RWMutex
	accessCount   int
	accessLatency time.Duration
}

// NewProductStorage creates a new product storage with sample data
func NewProductStorage() *ProductStorage {
	ps := &ProductStorage{
		products:      make(map[int]model.Product),
		accessLatency: 100 * time.Millisecond, // Simulate DB latency
	}

	// Add some sample products
	now := time.Now()
	for i := 1; i <= 100; i++ {
		ps.products[i] = model.Product{
			ID:          i,
			Name:        fmt.Sprintf("Product %d", i),
			Description: fmt.Sprintf("Description for product %d", i),
			Price:       float64(i) * 10.99,
			Category:    fmt.Sprintf("Category %d", (i%5)+1),
			Stock:       i * 5,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	return ps
}

// GetProduct retrieves a product by ID
func (s *ProductStorage) GetProduct(ctx context.Context, id int) (model.Product, error) {
	// Simulate database access latency
	time.Sleep(s.accessLatency)

	s.mu.RLock()
	defer s.mu.RUnlock()

	s.accessCount++
	if s.accessCount%10 == 0 {
		fmt.Printf("[DB] Access count: %d\n", s.accessCount)
	}

	product, exists := s.products[id]
	if !exists {
		return model.Product{}, errors.New("product not found")
	}

	return product, nil
}

// GetProducts retrieves products based on filter criteria
func (s *ProductStorage) GetProducts(ctx context.Context, filter model.ProductFilter) (model.ProductList, error) {
	// Simulate database access latency
	time.Sleep(s.accessLatency)

	s.mu.RLock()
	defer s.mu.RUnlock()

	s.accessCount++
	if s.accessCount%10 == 0 {
		fmt.Printf("[DB] Access count: %d\n", s.accessCount)
	}

	var filteredProducts []model.Product

	// Apply filters
	for _, product := range s.products {
		if filter.Category != "" && product.Category != filter.Category {
			continue
		}

		if filter.MinPrice > 0 && product.Price < filter.MinPrice {
			continue
		}

		if filter.MaxPrice > 0 && product.Price > filter.MaxPrice {
			continue
		}

		filteredProducts = append(filteredProducts, product)
	}

	// Calculate pagination
	total := len(filteredProducts)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	start := (filter.Page - 1) * filter.PageSize
	end := start + filter.PageSize
	if start >= total {
		return model.ProductList{
			Products: []model.Product{},
			Total:    total,
			Page:     filter.Page,
			PageSize: filter.PageSize,
		}, nil
	}

	if end > total {
		end = total
	}

	return model.ProductList{
		Products: filteredProducts[start:end],
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

// CreateProduct creates a new product
func (s *ProductStorage) CreateProduct(ctx context.Context, product model.Product) (model.Product, error) {
	// Simulate database access latency
	time.Sleep(s.accessLatency)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.accessCount++
	if s.accessCount%10 == 0 {
		fmt.Printf("[DB] Access count: %d\n", s.accessCount)
	}

	// Generate a new ID
	maxID := 0
	for id := range s.products {
		if id > maxID {
			maxID = id
		}
	}

	product.ID = maxID + 1
	product.CreatedAt = time.Now()
	product.UpdatedAt = product.CreatedAt

	s.products[product.ID] = product
	return product, nil
}

// UpdateProduct updates an existing product
func (s *ProductStorage) UpdateProduct(ctx context.Context, id int, product model.Product) (model.Product, error) {
	// Simulate database access latency
	time.Sleep(s.accessLatency)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.accessCount++
	if s.accessCount%10 == 0 {
		fmt.Printf("[DB] Access count: %d\n", s.accessCount)
	}

	existing, exists := s.products[id]
	if !exists {
		return model.Product{}, errors.New("product not found")
	}

	// Update fields but keep ID and creation time
	product.ID = existing.ID
	product.CreatedAt = existing.CreatedAt
	product.UpdatedAt = time.Now()

	s.products[id] = product
	return product, nil
}

// DeleteProduct deletes a product by ID
func (s *ProductStorage) DeleteProduct(ctx context.Context, id int) error {
	// Simulate database access latency
	time.Sleep(s.accessLatency)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.accessCount++
	if s.accessCount%10 == 0 {
		fmt.Printf("[DB] Access count: %d\n", s.accessCount)
	}

	if _, exists := s.products[id]; !exists {
		return errors.New("product not found")
	}

	delete(s.products, id)
	return nil
}

// GetPopularProductIDs returns a list of popular product IDs (simulated)
func (s *ProductStorage) GetPopularProductIDs(ctx context.Context) ([]int, error) {
	// Simulate database access latency
	time.Sleep(s.accessLatency)

	s.mu.RLock()
	defer s.mu.RUnlock()

	s.accessCount++
	if s.accessCount%10 == 0 {
		fmt.Printf("[DB] Access count: %d\n", s.accessCount)
	}

	// Randomly select 10 products as "popular"
	popularCount := 10
	if len(s.products) < popularCount {
		popularCount = len(s.products)
	}

	// Get all product IDs
	ids := make([]int, 0, len(s.products))
	for id := range s.products {
		ids = append(ids, id)
	}

	// Shuffle the IDs
	rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})

	// Return the first popularCount IDs
	return ids[:popularCount], nil
}
