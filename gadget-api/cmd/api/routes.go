package main

import (
	"gadget-api/internal/category"
	"gadget-api/internal/middleware"
	"gadget-api/internal/product"

	"github.com/gin-gonic/gin"
)

// ControllerRegistry mempermudah DI saat jumlah controller bertambah
type ControllerRegistry struct {
	Category *category.Controller
	Product  *product.Controller
	// Nanti tinggal tambah di bawah sini:
	// Auth     *auth.Controller
	// Cart     *cart.Controller
	// Order    *order.Controller
}

func setupRoutes(r *gin.Engine, reg ControllerRegistry) {
	r.Use(middleware.RequestID())

	v1 := r.Group("/api/v1")
	{
		// Modul Category
		catGroup := v1.Group("/categories")
		{
			catGroup.GET("", reg.Category.GetAll)
			catGroup.POST("", reg.Category.Create)
			catGroup.GET("/:id", reg.Category.GetByID)
		}

		// Modul Product
		prodGroup := v1.Group("/products")
		{
			prodGroup.GET("", reg.Product.GetAll)
			prodGroup.POST("", reg.Product.Create)
			prodGroup.GET("/:id", reg.Product.GetByID)
		}

		// Modul lainnya menyusul dengan pola yang sama...
	}
}
