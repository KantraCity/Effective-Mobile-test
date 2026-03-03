package handler_test

import (
	"Testwork/internal/handler"
	"Testwork/internal/model"
	"Testwork/internal/service"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ── мок сервиса ──────────────────────────────────────────────────────────────

type mockService struct {
	createFn         func(ctx context.Context, req model.CreateSubscriptionRequest) (int, error)
	updateFn         func(ctx context.Context, id string, req model.UpdateSubscriptionRequest) error
	listFn           func(ctx context.Context, userID string) ([]model.Subscription, error)
	getByIDFn        func(ctx context.Context, id string) (model.Subscription, error)
	deleteFn         func(ctx context.Context, id string) error
	calculateTotalFn func(ctx context.Context, userID, serviceName string, from, to time.Time) (int, error)
}

func (m *mockService) Create(ctx context.Context, req model.CreateSubscriptionRequest) (int, error) {
	return m.createFn(ctx, req)
}
func (m *mockService) Update(ctx context.Context, id string, req model.UpdateSubscriptionRequest) error {
	return m.updateFn(ctx, id, req)
}
func (m *mockService) List(ctx context.Context, userID string) ([]model.Subscription, error) {
	return m.listFn(ctx, userID)
}
func (m *mockService) GetByID(ctx context.Context, id string) (model.Subscription, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockService) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}
func (m *mockService) CalculateTotal(ctx context.Context, userID, serviceName string, from, to time.Time) (int, error) {
	return m.calculateTotalFn(ctx, userID, serviceName, from, to)
}

const validUUID = "60601fee-2bf1-4721-ae6f-7636e79a0cba"

func setupRouter(svc handler.SubscriptionService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := handler.NewHandler(svc, zerolog.Nop())
	r := gin.New()
	subs := r.Group("/api/v1/subscriptions")
	subs.POST("/", h.CreateSubscription)
	subs.GET("/", h.ListSubscriptions)
	subs.GET("/total", h.GetTotalCost)
	subs.GET("/:id", h.GetSubscription)
	subs.PUT("/:id", h.UpdateSubscription)
	subs.DELETE("/:id", h.DeleteSubscription)
	return r
}

func toJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	return bytes.NewBuffer(b)
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Errorf("expected status %d, got %d; body: %s", want, w.Code, w.Body.String())
	}
}

func TestCreateSubscription_Success(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ model.CreateSubscriptionRequest) (int, error) {
			return 42, nil
		},
	}
	r := setupRouter(svc)

	body := toJSON(t, map[string]any{
		"service_name": "Netflix",
		"price":        799,
		"user_id":      validUUID,
		"start_date":   "01-2024",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)

	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["id"] != 42 {
		t.Errorf("expected id=42, got %d", resp["id"])
	}
}

func TestCreateSubscription_InvalidJSON(t *testing.T) {
	r := setupRouter(&mockService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestCreateSubscription_InvalidUUID(t *testing.T) {
	r := setupRouter(&mockService{})
	body := toJSON(t, map[string]any{
		"service_name": "Netflix",
		"price":        799,
		"user_id":      "not-a-uuid",
		"start_date":   "01-2024",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestCreateSubscription_ServiceError(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ model.CreateSubscriptionRequest) (int, error) {
			return 0, errors.New("invalid start_date format, use MM-YYYY")
		},
	}
	r := setupRouter(svc)
	body := toJSON(t, map[string]any{
		"service_name": "Netflix",
		"price":        799,
		"user_id":      validUUID,
		"start_date":   "2024-01",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestUpdateSubscription_Success(t *testing.T) {
	svc := &mockService{
		updateFn: func(_ context.Context, _ string, _ model.UpdateSubscriptionRequest) error {
			return nil
		},
	}
	r := setupRouter(svc)
	body := toJSON(t, map[string]any{
		"service_name": "Netflix",
		"price":        999,
		"start_date":   "01-2024",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/subscriptions/1", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusOK)
}

func TestUpdateSubscription_NotFound(t *testing.T) {
	svc := &mockService{
		updateFn: func(_ context.Context, _ string, _ model.UpdateSubscriptionRequest) error {
			return service.ErrNotFound
		},
	}
	r := setupRouter(svc)
	body := toJSON(t, map[string]any{
		"service_name": "Netflix",
		"price":        999,
		"start_date":   "01-2024",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/subscriptions/999", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusNotFound)
}

func TestUpdateSubscription_InvalidJSON(t *testing.T) {
	r := setupRouter(&mockService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/subscriptions/1", bytes.NewBufferString("{bad}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetSubscription_Success(t *testing.T) {
	sub := model.Subscription{ID: 1, ServiceName: "Netflix", Price: 799, UserID: validUUID}
	svc := &mockService{
		getByIDFn: func(_ context.Context, _ string) (model.Subscription, error) {
			return sub, nil
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/1", nil))
	assertStatus(t, w, http.StatusOK)

	var got model.Subscription
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.ID != 1 || got.ServiceName != "Netflix" {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestGetSubscription_NotFound(t *testing.T) {
	svc := &mockService{
		getByIDFn: func(_ context.Context, _ string) (model.Subscription, error) {
			return model.Subscription{}, service.ErrNotFound
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/999", nil))
	assertStatus(t, w, http.StatusNotFound)
}

func TestGetSubscription_InternalError(t *testing.T) {
	svc := &mockService{
		getByIDFn: func(_ context.Context, _ string) (model.Subscription, error) {
			return model.Subscription{}, errors.New("db error")
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/1", nil))
	assertStatus(t, w, http.StatusInternalServerError)
}

func TestDeleteSubscription_Success(t *testing.T) {
	svc := &mockService{
		deleteFn: func(_ context.Context, _ string) error { return nil },
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/subscriptions/1", nil))
	assertStatus(t, w, http.StatusNoContent)
}

func TestDeleteSubscription_NotFound(t *testing.T) {
	svc := &mockService{
		deleteFn: func(_ context.Context, _ string) error { return service.ErrNotFound },
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/subscriptions/999", nil))
	assertStatus(t, w, http.StatusNotFound)
}

func TestListSubscriptions_Success(t *testing.T) {
	svc := &mockService{
		listFn: func(_ context.Context, _ string) ([]model.Subscription, error) {
			return []model.Subscription{{ID: 1, ServiceName: "Netflix"}}, nil
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/", nil))
	assertStatus(t, w, http.StatusOK)
}

func TestListSubscriptions_InvalidUUID(t *testing.T) {
	r := setupRouter(&mockService{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/?user_id=bad-uuid", nil))
	assertStatus(t, w, http.StatusBadRequest)
}

func TestListSubscriptions_FilterByUserID(t *testing.T) {
	var capturedUserID string
	svc := &mockService{
		listFn: func(_ context.Context, userID string) ([]model.Subscription, error) {
			capturedUserID = userID
			return nil, nil
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/?user_id="+validUUID, nil))
	assertStatus(t, w, http.StatusOK)
	if capturedUserID != validUUID {
		t.Errorf("expected userID=%s, got %s", validUUID, capturedUserID)
	}
}

func TestGetTotalCost_Success(t *testing.T) {
	svc := &mockService{
		calculateTotalFn: func(_ context.Context, _, _ string, _, _ time.Time) (int, error) {
			return 9588, nil
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	url := "/api/v1/subscriptions/total?user_id=" + validUUID + "&from=01-2024&to=12-2024"
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	assertStatus(t, w, http.StatusOK)

	var resp map[string]int
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["total_cost"] != 9588 {
		t.Errorf("expected total_cost=9588, got %d", resp["total_cost"])
	}
}

func TestGetTotalCost_MissingParams(t *testing.T) {
	r := setupRouter(&mockService{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/total?user_id="+validUUID, nil))
	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetTotalCost_InvalidUUID(t *testing.T) {
	r := setupRouter(&mockService{})
	w := httptest.NewRecorder()
	url := "/api/v1/subscriptions/total?user_id=bad&from=01-2024&to=12-2024"
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetTotalCost_InvalidDateFormat(t *testing.T) {
	r := setupRouter(&mockService{})
	w := httptest.NewRecorder()
	url := "/api/v1/subscriptions/total?user_id=" + validUUID + "&from=2024-01&to=2024-12"
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	assertStatus(t, w, http.StatusBadRequest)
}

func TestGetTotalCost_FromAfterTo(t *testing.T) {
	r := setupRouter(&mockService{})
	w := httptest.NewRecorder()
	url := "/api/v1/subscriptions/total?user_id=" + validUUID + "&from=12-2024&to=01-2024"
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	assertStatus(t, w, http.StatusBadRequest)
}

func TestUpdateSubscription_ServiceError(t *testing.T) {
	svc := &mockService{
		updateFn: func(_ context.Context, _ string, _ model.UpdateSubscriptionRequest) error {
			return errors.New("invalid end_date format")
		},
	}
	r := setupRouter(svc)
	body := toJSON(t, map[string]any{"service_name": "Netflix", "price": 999, "start_date": "01-2024"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/subscriptions/1", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusBadRequest)
}

func TestListSubscriptions_ServiceError(t *testing.T) {
	svc := &mockService{
		listFn: func(_ context.Context, _ string) ([]model.Subscription, error) {
			return nil, errors.New("db error")
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/", nil))
	assertStatus(t, w, http.StatusInternalServerError)
}

func TestDeleteSubscription_InternalError(t *testing.T) {
	svc := &mockService{
		deleteFn: func(_ context.Context, _ string) error {
			return errors.New("db error")
		},
	}
	r := setupRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/subscriptions/1", nil))
	assertStatus(t, w, http.StatusInternalServerError)
}
