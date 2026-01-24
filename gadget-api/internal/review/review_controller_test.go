package review_test

import (
	"bytes"
	"context"
	"encoding/json"
	"gadget-api/internal/review"
	reviewerrors "gadget-api/internal/review/errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ==================== FAKE SERVICE (Mock Manual) ====================

type fakeReviewService struct {
	createFunc           func(ctx context.Context, userID, productSlug string, req review.CreateReviewRequest) (review.ReviewResponse, error)
	getByProductSlugFunc func(ctx context.Context, productSlug string, page, limit int) (review.ReviewListResponse, error)
	getByUserIDFunc      func(ctx context.Context, userID string, page, limit int) (review.UserReviewListResponse, error)
	checkEligibilityFunc func(ctx context.Context, userID, productSlug string) (review.ReviewEligibilityResponse, error)
	updateFunc           func(ctx context.Context, reviewID, userID string, req review.UpdateReviewRequest) (review.ReviewResponse, error)
	deleteFunc           func(ctx context.Context, reviewID, userID string) error
}

func (f *fakeReviewService) Create(ctx context.Context, u, s string, r review.CreateReviewRequest) (review.ReviewResponse, error) {
	return f.createFunc(ctx, u, s, r)
}
func (f *fakeReviewService) GetByProductSlug(ctx context.Context, s string, p, l int) (review.ReviewListResponse, error) {
	return f.getByProductSlugFunc(ctx, s, p, l)
}
func (f *fakeReviewService) GetByUserID(ctx context.Context, u string, p, l int) (review.UserReviewListResponse, error) {
	return f.getByUserIDFunc(ctx, u, p, l)
}
func (f *fakeReviewService) CheckEligibility(ctx context.Context, u, s string) (review.ReviewEligibilityResponse, error) {
	return f.checkEligibilityFunc(ctx, u, s)
}
func (f *fakeReviewService) Update(ctx context.Context, r, u string, req review.UpdateReviewRequest) (review.ReviewResponse, error) {
	return f.updateFunc(ctx, r, u, req)
}
func (f *fakeReviewService) Delete(ctx context.Context, r, u string) error {
	return f.deleteFunc(ctx, r, u)
}

// ==================== REUSABLE HELPERS ====================

type reviewTestDeps struct {
	svc  *fakeReviewService
	ctrl *review.Controller
	w    *httptest.ResponseRecorder
	ctx  *gin.Context
}

func setupReviewControllerTest() *reviewTestDeps {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	svc := &fakeReviewService{}
	ctrl := review.NewController(svc)

	return &reviewTestDeps{
		svc:  svc,
		ctrl: ctrl,
		w:    w,
		ctx:  ctx,
	}
}

func (d *reviewTestDeps) performRequest(method, path string, body interface{}) {
	var jsonBody []byte
	if body != nil {
		jsonBody, _ = json.Marshal(body)
	}
	d.ctx.Request = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
	d.ctx.Request.Header.Set("Content-Type", "application/json")
}

// ==================== CREATE REVIEW ====================

func TestReviewController_Create(t *testing.T) {
	t.Run("positive - success create", func(t *testing.T) {
		d := setupReviewControllerTest()
		userID := uuid.New().String()
		slug := "iphone-15"
		req := review.CreateReviewRequest{Rating: 5, Comment: "Mantapsssssssssss"}

		d.ctx.Set("user_id", userID)
		d.ctx.Params = gin.Params{{Key: "slug", Value: slug}}
		d.performRequest(http.MethodPost, "/", req)

		d.svc.createFunc = func(ctx context.Context, uid, s string, r review.CreateReviewRequest) (review.ReviewResponse, error) {
			return review.ReviewResponse{Comment: r.Comment, Rating: r.Rating}, nil
		}

		d.ctrl.Create(d.ctx)

		assert.Equal(t, http.StatusCreated, d.w.Code)
		assert.Contains(t, d.w.Body.String(), "Mantapssssssss")
	})

	t.Run("negative - service returns unauthorized", func(t *testing.T) {
		d := setupReviewControllerTest()
		d.ctx.Params = gin.Params{{Key: "slug", Value: "iphone"}}
		d.performRequest(http.MethodPost, "/", review.CreateReviewRequest{Rating: 5})

		d.svc.createFunc = func(ctx context.Context, uid, s string, r review.CreateReviewRequest) (review.ReviewResponse, error) {
			return review.ReviewResponse{}, reviewerrors.ErrUnauthenticated // Assuming error exists in package
		}

		d.ctrl.Create(d.ctx)
		assert.Equal(t, http.StatusUnauthorized, d.w.Code)
	})
}

// ==================== GET BY PRODUCT SLUG ====================

func TestReviewController_GetReviewsByProductSlug(t *testing.T) {
	t.Run("positive - get reviews", func(t *testing.T) {
		d := setupReviewControllerTest()
		d.ctx.Params = gin.Params{{Key: "slug", Value: "iphone"}}
		d.performRequest(http.MethodGet, "/?page=1&limit=10", nil)

		d.svc.getByProductSlugFunc = func(ctx context.Context, slug string, page, limit int) (review.ReviewListResponse, error) {
			return review.ReviewListResponse{Total: 1, Page: 1}, nil
		}

		d.ctrl.GetReviewsByProductSlug(d.ctx)
		assert.Equal(t, http.StatusOK, d.w.Code)
	})
}

// ==================== GET BY USER ID ====================

func TestReviewController_GetReviewsByUserID(t *testing.T) {
	t.Run("positive - get user reviews", func(t *testing.T) {
		d := setupReviewControllerTest()
		authID := uuid.New().String()
		targetUserID := uuid.New().String()

		d.ctx.Set("user_id", authID)
		d.ctx.Params = gin.Params{{Key: "userId", Value: targetUserID}}
		d.performRequest(http.MethodGet, "/", nil)

		d.svc.getByUserIDFunc = func(ctx context.Context, uid string, page, limit int) (review.UserReviewListResponse, error) {
			return review.UserReviewListResponse{Total: 1}, nil
		}

		d.ctrl.GetReviewsByUserID(d.ctx)
		assert.Equal(t, http.StatusOK, d.w.Code)
	})

	t.Run("negative - forbidden via service", func(t *testing.T) {
		d := setupReviewControllerTest()
		d.ctx.Set("user_id", uuid.New().String())
		d.ctx.Params = gin.Params{{Key: "userId", Value: uuid.New().String()}}
		d.performRequest(http.MethodGet, "/", nil)

		d.svc.getByUserIDFunc = func(ctx context.Context, uid string, page, limit int) (review.UserReviewListResponse, error) {
			return review.UserReviewListResponse{}, reviewerrors.ErrForbidden // Assuming error exists
		}

		d.ctrl.GetReviewsByUserID(d.ctx)
		assert.Equal(t, http.StatusForbidden, d.w.Code)
	})
}

// ==================== ELIGIBILITY ====================

func TestReviewController_CheckReviewEligibility(t *testing.T) {
	t.Run("positive - eligible", func(t *testing.T) {
		d := setupReviewControllerTest()
		d.ctx.Set("user_id", uuid.New().String())
		d.ctx.Params = gin.Params{{Key: "slug", Value: "iphone"}}
		d.performRequest(http.MethodGet, "/", nil)

		d.svc.checkEligibilityFunc = func(ctx context.Context, uid, s string) (review.ReviewEligibilityResponse, error) {
			return review.ReviewEligibilityResponse{CanReview: true}, nil
		}

		d.ctrl.CheckReviewEligibility(d.ctx)
		assert.Equal(t, http.StatusOK, d.w.Code)
	})
}

// ==================== UPDATE REVIEW ====================

func TestReviewController_UpdateReview(t *testing.T) {
	t.Run("positive - success update", func(t *testing.T) {
		d := setupReviewControllerTest()
		userID := uuid.New().String()
		reviewID := uuid.New().String()
		req := review.UpdateReviewRequest{Rating: 4, Comment: "Updated Comment"}

		d.ctx.Set("user_id", userID)
		d.ctx.Params = gin.Params{{Key: "id", Value: reviewID}}
		d.performRequest(http.MethodPut, "/", req)

		d.svc.updateFunc = func(ctx context.Context, rid, uid string, r review.UpdateReviewRequest) (review.ReviewResponse, error) {
			return review.ReviewResponse{ID: rid, Comment: r.Comment}, nil
		}

		d.ctrl.UpdateReview(d.ctx)
		assert.Equal(t, http.StatusOK, d.w.Code)
	})
}

// ==================== DELETE REVIEW ====================

func TestReviewController_DeleteReview(t *testing.T) {
	t.Run("positive - success delete", func(t *testing.T) {
		d := setupReviewControllerTest()
		userID := uuid.New().String()
		reviewID := uuid.New().String()

		d.ctx.Set("user_id", userID)
		d.ctx.Params = gin.Params{{Key: "id", Value: reviewID}}
		d.performRequest(http.MethodDelete, "/", nil)

		d.svc.deleteFunc = func(ctx context.Context, rid, uid string) error {
			return nil
		}

		d.ctrl.DeleteReview(d.ctx)
		assert.Equal(t, http.StatusOK, d.w.Code)
	})
}
