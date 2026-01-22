package main

import (
	"gadget-api/internal/auth"
	"gadget-api/internal/brand"
	"gadget-api/internal/cart"
	"gadget-api/internal/category"
	"gadget-api/internal/middleware"
	"gadget-api/internal/order"
	"gadget-api/internal/product"
	"gadget-api/internal/review"

	"github.com/gin-gonic/gin"
)

type ControllerRegistry struct {
	Auth     *auth.Controller
	Category *category.Controller
	Brand    *brand.Controller
	Product  *product.Controller
	Review   *review.Controller
	Cart     *cart.Controller
	Order    *order.Controller
}

func setupRoutes(r *gin.Engine, reg ControllerRegistry) {
	r.Use(middleware.RequestID())

	v1 := r.Group("/api/v1")
	{
		// ========================
		// AUTH (PUBLIC)
		// ========================
		auth := v1.Group("/auth")
		{
			auth.GET("/me", middleware.AuthMiddleware(), reg.Auth.Me)
			auth.POST("/login", reg.Auth.Login)
			auth.POST("/refresh", reg.Auth.RefreshToken)
			auth.POST("/logout", reg.Auth.Logout)
			auth.POST("/register", reg.Auth.Register)
		}

		// ========================
		// CATEGORY
		// ========================
		categories := v1.Group("/categories")
		{
			categories.GET("", reg.Category.ListPublic)
			categories.GET("/:id", reg.Category.GetByID)
		}

		adminCategories := v1.Group("/admin/categories")
		adminCategories.Use(
			middleware.AuthMiddleware(),
			middleware.RoleMiddleware("ADMIN", "SUPERADMIN"),
		)
		{
			adminCategories.GET("", reg.Category.ListAdmin)
			adminCategories.POST("", reg.Category.Create)
			adminCategories.PUT("/:id", reg.Category.Update)
			adminCategories.DELETE("/:id", reg.Category.Delete)
			adminCategories.PATCH("/:id/restore", reg.Category.Restore)
		}

		// ========================
		// CATEGORY
		// ========================
		brands := v1.Group("/brands")
		{
			brands.GET("", reg.Brand.ListPublic)
			brands.GET("/:id", reg.Brand.GetByID)
		}

		adminBrands := v1.Group("/admin/brands")
		adminBrands.Use(
			middleware.AuthMiddleware(),
			middleware.RoleMiddleware("ADMIN", "SUPERADMIN"),
		)
		{
			adminBrands.GET("", reg.Brand.ListAdmin)
			adminBrands.POST("", reg.Brand.Create)
			adminBrands.PUT("/:id", reg.Brand.Update)
			adminBrands.DELETE("/:id", reg.Brand.Delete)
			adminBrands.PATCH("/:id/restore", reg.Brand.Restore)
		}

		// ========================
		// PRODUCT
		// ========================
		products := v1.Group("/products")
		{
			products.GET("", reg.Product.GetPublicList)
			products.GET("/:slug", reg.Product.GetBySlug)
		}
		optional := products.Group("")
		optional.Use(middleware.OptionalAuthMiddleware())
		{
			optional.GET(
				"/:slug/reviews/eligibility",
				reg.Review.CheckReviewEligibility,
			)
		}

		adminProducts := v1.Group("/admin/products")
		adminProducts.Use(middleware.AuthMiddleware())
		adminProducts.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))
		{
			adminProducts.GET("", reg.Product.GetAdminList)
			adminProducts.POST("", reg.Product.Create)
			adminProducts.PUT("/:id", reg.Product.Update)
			adminProducts.DELETE("/:id", reg.Product.Delete)
			adminProducts.PATCH("/:id/restore", reg.Product.Restore)
		}

		// ========================
		// REVIEW (AUTH REQUIRED, NON-PRODUCT)
		// ========================
		reviews := v1.Group("")
		reviews.Use(middleware.AuthMiddleware())
		{
			reviews.PUT("/reviews/:id", reg.Review.UpdateReview)
			reviews.DELETE("/reviews/:id", reg.Review.DeleteReview)
			reviews.GET("/users/:userId/reviews", reg.Review.GetReviewsByUserID)
		}

		// ========================
		// CART (AUTH REQUIRED)
		// ========================
		cart := v1.Group("/cart/:userId")
		cart.Use(middleware.AuthMiddleware())
		{
			cart.POST("", reg.Cart.Create)
			cart.GET("", reg.Cart.Detail)
			cart.GET("/count", reg.Cart.Count)
			cart.DELETE("", reg.Cart.Delete)

			items := cart.Group("/items/:productId")
			{
				items.PUT("", reg.Cart.UpdateQty)
				items.POST("/increment", reg.Cart.Increment)
				items.POST("/decrement", reg.Cart.Decrement)
				items.DELETE("", reg.Cart.DeleteItem)
			}
		}

		// ========================
		// ORDER
		// ========================
		orders := v1.Group("/orders")
		orders.Use(middleware.AuthMiddleware()) // Semua route order butuh login
		{
			// Customer Routes
			orders.POST("/checkout", reg.Order.Checkout)
			orders.GET("", reg.Order.List)
			orders.GET("/:id", reg.Order.Detail)
			orders.PATCH("/:id/cancel", reg.Order.Cancel)
			orders.PATCH("/:id/status", reg.Order.UpdateStatusByCustomer)

			// Admin Routes (Management)
			adminOrders := orders.Group("/admin")
			adminOrders.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))
			{
				adminOrders.GET("", reg.Order.ListAdmin)
				adminOrders.PATCH("/:id/status", reg.Order.UpdateStatusByAdmin)
			}
		}
	}
}
