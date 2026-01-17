package main

import (
	"gadget-api/internal/auth"
	"gadget-api/internal/cart"
	"gadget-api/internal/category"
	"gadget-api/internal/middleware"
	"gadget-api/internal/product"

	"github.com/gin-gonic/gin"
)

type ControllerRegistry struct {
	Auth     *auth.Controller
	Category *category.Controller
	Product  *product.Controller
	Cart     *cart.Controller
	// Order    *order.Controller
}

func setupRoutes(r *gin.Engine, reg ControllerRegistry) {
	r.Use(middleware.RequestID())

	v1 := r.Group("/api/v1")
	{
		// Auth Routes (Public)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", reg.Auth.Login)
			auth.POST("/logout", reg.Auth.Logout)
			auth.POST("/register", reg.Auth.Register)
		}
		// Modul Category
		// Category Routes
		cat := v1.Group("/categories")
		{
			cat.GET("", reg.Category.GetAll)
			cat.GET("/:id", reg.Category.GetByID)

			// Protected Admin Routes
			adminCat := cat.Group("")
			adminCat.Use(middleware.AuthMiddleware())
			adminCat.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))
			{
				adminCat.POST("", reg.Category.Create)
				adminCat.PUT("/:id", reg.Category.Update)
				adminCat.DELETE("/:id", reg.Category.Delete)
				adminCat.PATCH("/:id/restore", reg.Category.Restore)
			}
		}

		// Product Routes
		prod := v1.Group("/products")
		{
			prod.GET("", reg.Product.GetPublicList)
			prod.GET("/:id", reg.Product.GetByID)

			// Protected Admin Routes
			adminProd := prod.Group("")
			adminProd.Use(middleware.AuthMiddleware())
			adminProd.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))
			{
				prod.GET("", reg.Product.GetAdminList)
				adminProd.POST("", reg.Product.Create)
				adminProd.PUT("/:id", reg.Product.Update)
				adminProd.DELETE("/:id", reg.Product.Delete)
				adminProd.PATCH("/:id/restore", reg.Product.Restore)
			}
		}

		// Cart Routes (Protected)
		// Semua operasi cart membutuhkan user login
		cart := v1.Group("/cart/:userId")
		cart.Use(middleware.AuthMiddleware())
		{
			cart.POST("", reg.Cart.Create)
			cart.GET("", reg.Cart.Detail)
			cart.GET("/count", reg.Cart.Count)
			cart.DELETE("", reg.Cart.Delete)

			// Item management dalam cart
			items := cart.Group("/items/:productId")
			{
				items.PUT("", reg.Cart.UpdateQty)
				items.POST("/increment", reg.Cart.Increment)
				items.POST("/decrement", reg.Cart.Decrement)
				items.DELETE("", reg.Cart.DeleteItem)
			}
		}

	}
}
