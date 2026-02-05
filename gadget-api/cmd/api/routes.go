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

type HandlerRegistry struct {
	Auth     *auth.Handler
	Category *category.Handler
	Brand    *brand.Handler
	Product  *product.Handler
	Review   *review.Handler
	Cart     *cart.Handler
	Order    *order.Handler
}

func setupRoutes(r *gin.Engine, reg HandlerRegistry) {
	r.Use(middleware.RequestID())

	v1 := r.Group("/api/v1")
	{
		// ========================
		// AUTH (PUBLIC)
		// ========================

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
			adminCategories.PATCH("/:id", reg.Category.Update)
			adminCategories.DELETE("/:id", reg.Category.Delete)
			adminCategories.PATCH("/:id/restore", reg.Category.Restore)
		}

		// ========================
		// BRANDS
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
			adminBrands.PATCH("/:id", reg.Brand.Update)
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
			adminProducts.PATCH("/:id", reg.Product.Update)
			adminProducts.DELETE("/:id", reg.Product.Delete)
			adminProducts.PATCH("/:id/restore", reg.Product.Restore)
		}

		// ========================
		// REVIEW (AUTH REQUIRED, NON-PRODUCT)
		// ========================
		reviews := v1.Group("")
		reviews.Use(middleware.AuthMiddleware())
		{
			reviews.PATCH("/reviews/:id", reg.Review.UpdateReview)
			reviews.DELETE("/reviews/:id", reg.Review.DeleteReview)
			reviews.GET("/users/:userId/reviews", reg.Review.GetReviewsByUserID)
		}

		// ========================
		// CART (AUTH REQUIRED)
		// ========================
		carts := v1.Group("/carts")
		carts.Use(middleware.AuthMiddleware())
		carts.Use(middleware.ExtractUserID())
		{
			carts.POST("", reg.Cart.Create)
			carts.GET("/detail", reg.Cart.Detail)
			carts.GET("/count", reg.Cart.Count)
			carts.DELETE("", reg.Cart.Delete)

			items := carts.Group("/items/:productId")
			{
				items.POST("", reg.Cart.AddItem)
				items.PATCH("", reg.Cart.UpdateQty)
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

		}
		// Admin Routes (Management)
		adminOrders := v1.Group("/admin/orders")
		adminOrders.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))
		{
			adminOrders.GET("", reg.Order.ListAdmin)
			adminOrders.PATCH("/:id/status", reg.Order.UpdateStatusByAdmin)
		}
	}
}
