package service_test

import (
	"Testwork/internal/model"
	"Testwork/internal/repository"
	"Testwork/internal/service"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

type mockRepo struct {
	createFn           func(ctx context.Context, serviceName string, price int, userID string, start time.Time, end *time.Time) (int, error)
	getPeriodsByFilter func(ctx context.Context, userID string, serviceName string) ([]model.SubCalcData, error)
	listFn             func(ctx context.Context, userID string) ([]model.Subscription, error)
	getByIDFn          func(ctx context.Context, id string) (model.Subscription, error)
	updateFn           func(ctx context.Context, id string, serviceName string, price int, start time.Time, end *time.Time) error
	deleteFn           func(ctx context.Context, id string) error
}

func (m *mockRepo) Create(ctx context.Context, serviceName string, price int, userID string, start time.Time, end *time.Time) (int, error) {
	return m.createFn(ctx, serviceName, price, userID, start, end)
}
func (m *mockRepo) GetPeriodsByFilter(ctx context.Context, userID, serviceName string) ([]model.SubCalcData, error) {
	return m.getPeriodsByFilter(ctx, userID, serviceName)
}
func (m *mockRepo) List(ctx context.Context, userID string) ([]model.Subscription, error) {
	return m.listFn(ctx, userID)
}
func (m *mockRepo) GetByID(ctx context.Context, id string) (model.Subscription, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockRepo) Update(ctx context.Context, id, serviceName string, price int, start time.Time, end *time.Time) error {
	return m.updateFn(ctx, id, serviceName, price, start, end)
}
func (m *mockRepo) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

const validUUID = "60601fee-2bf1-4721-ae6f-7636e79a0cba"

func TestCreate_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ string, _ int, _ string, _ time.Time, _ *time.Time) (int, error) {
			return 1, nil
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	id, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       799,
		UserID:      validUUID,
		StartDate:   "01-2024",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 1 {
		t.Errorf("expected id=1, got %d", id)
	}
}

func TestCreate_InvalidStartDate(t *testing.T) {
	svc := service.NewSubscriptionServiceWithRepo(&mockRepo{}, zerolog.Nop())
	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       799,
		UserID:      validUUID,
		StartDate:   "2024-01",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreate_InvalidEndDate(t *testing.T) {
	svc := service.NewSubscriptionServiceWithRepo(&mockRepo{}, zerolog.Nop())
	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       799,
		UserID:      validUUID,
		StartDate:   "01-2024",
		EndDate:     "bad-date",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreate_EndDateBeforeStartDate(t *testing.T) {
	svc := service.NewSubscriptionServiceWithRepo(&mockRepo{}, zerolog.Nop())
	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       799,
		UserID:      validUUID,
		StartDate:   "06-2024",
		EndDate:     "01-2024",
	})
	if err == nil {
		t.Fatal("expected error for end_date before start_date")
	}
}

func TestUpdate_Success(t *testing.T) {
	repo := &mockRepo{
		updateFn: func(_ context.Context, _ string, _ string, _ int, _ time.Time, _ *time.Time) error {
			return nil
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	err := svc.Update(context.Background(), "1", model.UpdateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       999,
		StartDate:   "01-2024",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &mockRepo{
		updateFn: func(_ context.Context, _ string, _ string, _ int, _ time.Time, _ *time.Time) error {
			return repository.ErrNotFound
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	err := svc.Update(context.Background(), "999", model.UpdateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       999,
		StartDate:   "01-2024",
	})
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdate_InvalidStartDate(t *testing.T) {
	svc := service.NewSubscriptionServiceWithRepo(&mockRepo{}, zerolog.Nop())
	err := svc.Update(context.Background(), "1", model.UpdateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       999,
		StartDate:   "bad",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCalculateTotal_NoEndDate(t *testing.T) {
	repo := &mockRepo{
		getPeriodsByFilter: func(_ context.Context, _, _ string) ([]model.SubCalcData, error) {
			return []model.SubCalcData{
				{Price: 100, StartDate: mustParse("01-2024"), EndDate: nil},
			}, nil
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	from := mustParse("01-2024")
	to := mustParse("03-2024")
	total, err := svc.CalculateTotal(context.Background(), validUUID, "", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 300 {
		t.Errorf("expected 300, got %d", total)
	}
}

func TestCalculateTotal_WithEndDate(t *testing.T) {
	end := mustParse("02-2024")
	repo := &mockRepo{
		getPeriodsByFilter: func(_ context.Context, _, _ string) ([]model.SubCalcData, error) {
			return []model.SubCalcData{
				{Price: 100, StartDate: mustParse("01-2024"), EndDate: &end},
			}, nil
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	from := mustParse("01-2024")
	to := mustParse("06-2024")
	total, err := svc.CalculateTotal(context.Background(), validUUID, "", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 200 {
		t.Errorf("expected 200, got %d", total)
	}
}

func TestCalculateTotal_NoOverlap(t *testing.T) {
	end := mustParse("01-2023")
	repo := &mockRepo{
		getPeriodsByFilter: func(_ context.Context, _, _ string) ([]model.SubCalcData, error) {
			return []model.SubCalcData{
				{Price: 100, StartDate: mustParse("01-2022"), EndDate: &end},
			}, nil
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	total, err := svc.CalculateTotal(context.Background(), validUUID, "", mustParse("01-2024"), mustParse("12-2024"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0, got %d", total)
	}
}

func TestCalculateTotal_MultipleSubscriptions(t *testing.T) {
	repo := &mockRepo{
		getPeriodsByFilter: func(_ context.Context, _, _ string) ([]model.SubCalcData, error) {
			return []model.SubCalcData{
				{Price: 100, StartDate: mustParse("01-2024"), EndDate: nil},
				{Price: 200, StartDate: mustParse("01-2024"), EndDate: nil},
			}, nil
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	total, err := svc.CalculateTotal(context.Background(), validUUID, "", mustParse("01-2024"), mustParse("01-2024"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 300 {
		t.Errorf("expected 300, got %d", total)
	}
}

func TestCalculateTotal_RepoError(t *testing.T) {
	repo := &mockRepo{
		getPeriodsByFilter: func(_ context.Context, _, _ string) ([]model.SubCalcData, error) {
			return nil, errors.New("db error")
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	_, err := svc.CalculateTotal(context.Background(), validUUID, "", mustParse("01-2024"), mustParse("12-2024"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDelete_Success(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, _ string) error { return nil },
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	if err := svc.Delete(context.Background(), "1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, _ string) error { return repository.ErrNotFound },
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	err := svc.Delete(context.Background(), "999")
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	want := model.Subscription{ID: 1, ServiceName: "Netflix"}
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, _ string) (model.Subscription, error) { return want, nil },
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	got, err := svc.GetByID(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected ID=%d, got %d", want.ID, got.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, _ string) (model.Subscription, error) {
			return model.Subscription{}, repository.ErrNotFound
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	_, err := svc.GetByID(context.Background(), "999")
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func mustParse(s string) time.Time {
	t, err := time.Parse("01-2006", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestCreate_RepoError(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ string, _ int, _ string, _ time.Time, _ *time.Time) (int, error) {
			return 0, errors.New("db error")
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Netflix", Price: 799, UserID: validUUID, StartDate: "01-2024",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdate_InvalidEndDate(t *testing.T) {
	svc := service.NewSubscriptionServiceWithRepo(&mockRepo{}, zerolog.Nop())
	err := svc.Update(context.Background(), "1", model.UpdateSubscriptionRequest{
		ServiceName: "Netflix", Price: 999, StartDate: "01-2024", EndDate: "bad",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdate_EndDateBeforeStart(t *testing.T) {
	svc := service.NewSubscriptionServiceWithRepo(&mockRepo{}, zerolog.Nop())
	err := svc.Update(context.Background(), "1", model.UpdateSubscriptionRequest{
		ServiceName: "Netflix", Price: 999, StartDate: "06-2024", EndDate: "01-2024",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestList_RepoError(t *testing.T) {
	repo := &mockRepo{
		listFn: func(_ context.Context, _ string) ([]model.Subscription, error) {
			return nil, errors.New("db error")
		},
	}
	svc := service.NewSubscriptionServiceWithRepo(repo, zerolog.Nop())
	_, err := svc.List(context.Background(), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
