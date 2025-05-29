package model

import (
	"time"
)

// Product represents a product in our e-commerce system
type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProductList represents a list of products with pagination info
type ProductList struct {
	Products []Product `json:"products"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}

// ProductFilter represents the filter criteria for products
type ProductFilter struct {
	Category string  `form:"category"`
	MinPrice float64 `form:"min_price"`
	MaxPrice float64 `form:"max_price"`
	Page     int     `form:"page,default=1"`
	PageSize int     `form:"page_size,default=20"`
}
