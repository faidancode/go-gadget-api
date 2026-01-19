package main

import (
	"gadget-api/internal/auth"
	"gadget-api/internal/cart"
	"gadget-api/internal/category"
	"gadget-api/internal/middleware"
	"gadget-api/internal/product"
	"gadget-api/internal/review"

	"github.com/gin-gonic/gin"
)

type ControllerRegistry struct {
	Auth     *auth.Controller
	Category *category.Controller
	Product  *product.Controller
	Review   *review.Controller
	Cart     *cart.Controller
	// Order    *order.Controller
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
			auth.POST("/login", reg.Auth.Login)
			auth.POST("/logout", reg.Auth.Logout)
			auth.POST("/register", reg.Auth.Register)
		}

		// ========================
		// CATEGORY
		// ========================
		cat := v1.Group("/categories")
		{
			// Public
			cat.GET("", reg.Category.GetAll)
			cat.GET("/:id", reg.Category.GetByID)

			// Admin only
			admin := cat.Group("")
			admin.Use(
				middleware.AuthMiddleware(),
				middleware.RoleMiddleware("ADMIN", "SUPERADMIN"),
			)
			{
				admin.POST("", reg.Category.Create)
				admin.PUT("/:id", reg.Category.Update)
				admin.DELETE("/:id", reg.Category.Delete)
				admin.PATCH("/:id/restore", reg.Category.Restore)
			}
		}

		// ========================
		// PRODUCT
		// ========================
		prod := v1.Group("/products")
		{
			// Public
			prod.GET("", reg.Product.GetPublicList)
			prod.GET("/:slug", reg.Product.GetBySlug)
			prod.GET("/:slug/reviews", reg.Review.GetReviewsByProductSlug)

			// Optional Auth (Guest / Logged-in)
			optional := prod.Group("")
			optional.Use(middleware.OptionalAuthMiddleware())
			{
				optional.GET(
					"/:slug/reviews/eligibility",
					reg.Review.CheckReviewEligibility,
				)
			}

			// Auth Required (User)
			authUser := prod.Group("")
			authUser.Use(middleware.AuthMiddleware())
			{
				authUser.POST("/:slug/reviews", reg.Review.CreateReview)

				// Admin only (inherit auth)
				admin := authUser.Group("")
				admin.Use(middleware.RoleMiddleware("ADMIN", "SUPERADMIN"))
				{
					admin.GET("/admin/list", reg.Product.GetAdminList)
					admin.POST("", reg.Product.Create)
					admin.PUT("/:id", reg.Product.Update)
					admin.DELETE("/:id", reg.Product.Delete)
					admin.PATCH("/:id/restore", reg.Product.Restore)
				}
			}
		}

		// ========================
		// REVIEW (AUTH REQUIRED, NON-PRODUCT)
		// ========================
		review := v1.Group("")
		review.Use(middleware.AuthMiddleware())
		{
			review.PUT("/reviews/:id", reg.Review.UpdateReview)
			review.DELETE("/reviews/:id", reg.Review.DeleteReview)
			review.GET("/users/:userId/reviews", reg.Review.GetReviewsByUserID)
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
	}
}
