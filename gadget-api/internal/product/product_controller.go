package product

import (
	"gadget-api/internal/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service *Service
}

func NewController(s *Service) *Controller {
	return &Controller{service: s}
}

func (ctrl *Controller) GetAll(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	categoryID := c.Query("category_id")

	// Mendukung route /products/category/:categoryId
	if c.Param("categoryId") != "" {
		categoryID = c.Param("categoryId")
	}

	data, total, err := ctrl.service.GetAll(c.Request.Context(), page, limit, search, categoryID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FETCH_ERROR", "Failed to fetch products", err.Error())
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pag := response.Pagination{
		Page: page, PageSize: limit, TotalItems: total, TotalPages: totalPages,
		HasNextPage: page < totalPages, HasPreviousPage: page > 1,
	}
	response.SuccessWithPagination(c, http.StatusOK, "Products retrieved", data, pag)
}

func (ctrl *Controller) Create(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid input", err.Error())
		return
	}
	res, err := ctrl.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "CREATE_ERROR", "Failed to create product", err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Product created", res)
}

func (ctrl *Controller) GetByID(c *gin.Context) {
	res, err := ctrl.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "Product not found", nil)
		return
	}
	response.Success(c, http.StatusOK, "Product found", res)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	ctrl.service.Delete(c.Request.Context(), c.Param("id"))
	response.Success(c, http.StatusOK, "Product deleted (soft delete)", nil)
}

func (ctrl *Controller) Restore(c *gin.Context) {
	res, _ := ctrl.service.Restore(c.Request.Context(), c.Param("id"))
	response.Success(c, http.StatusOK, "Product restored", res)
}
