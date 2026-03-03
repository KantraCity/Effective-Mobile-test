package repository_test

import (
	"Testwork/internal/repository"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rs/zerolog"
)

const validUUID = "60601fee-2bf1-4721-ae6f-7636e79a0cba"

func mustParse(s string) time.Time {
	t, err := time.Parse("01-2006", s)
	if err != nil {
		panic(err)
	}
	return t
}

func setup(t *testing.T) (*repository.SubscriptionRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := repository.NewSubscriptionRepository(db, zerolog.Nop())
	return repo, mock
}

func TestCreate_Success(t *testing.T) {
	repo, mock := setup(t)

	mock.ExpectQuery(`INSERT INTO subscriptions`).
		WithArgs("Netflix", 799, validUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	id, err := repo.Create(context.Background(), "Netflix", 799, validUUID, mustParse("01-2024"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 1 {
		t.Errorf("expected id=1, got %d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCreate_DBError(t *testing.T) {
	repo, mock := setup(t)

	mock.ExpectQuery(`INSERT INTO subscriptions`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("db error"))

	_, err := repo.Create(context.Background(), "Netflix", 799, validUUID, mustParse("01-2024"), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetByID_Success(t *testing.T) {
	repo, mock := setup(t)

	rows := sqlmock.NewRows([]string{"id", "service_name", "price", "user_id", "start_date", "end_date"}).
		AddRow(1, "Netflix", 799, validUUID, mustParse("01-2024"), nil)

	mock.ExpectQuery(`SELECT .+ FROM subscriptions WHERE id`).
		WithArgs("1").
		WillReturnRows(rows)

	sub, err := repo.GetByID(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.ID != 1 || sub.ServiceName != "Netflix" {
		t.Errorf("unexpected subscription: %+v", sub)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo, mock := setup(t)

	mock.ExpectQuery(`SELECT .+ FROM subscriptions WHERE id`).
		WithArgs("999").
		WillReturnRows(sqlmock.NewRows([]string{"id", "service_name", "price", "user_id", "start_date", "end_date"}))

	_, err := repo.GetByID(context.Background(), "999")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestList_All(t *testing.T) {
	repo, mock := setup(t)

	rows := sqlmock.NewRows([]string{"id", "service_name", "price", "user_id", "start_date", "end_date"}).
		AddRow(1, "Netflix", 799, validUUID, mustParse("01-2024"), nil).
		AddRow(2, "Spotify", 299, validUUID, mustParse("03-2024"), nil)

	mock.ExpectQuery(`SELECT .+ FROM subscriptions`).
		WillReturnRows(rows)

	subs, err := repo.List(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(subs))
	}
}

func TestList_FilterByUserID(t *testing.T) {
	repo, mock := setup(t)

	rows := sqlmock.NewRows([]string{"id", "service_name", "price", "user_id", "start_date", "end_date"}).
		AddRow(1, "Netflix", 799, validUUID, mustParse("01-2024"), nil)

	mock.ExpectQuery(`SELECT .+ FROM subscriptions WHERE user_id`).
		WithArgs(validUUID).
		WillReturnRows(rows)

	subs, err := repo.List(context.Background(), validUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("expected 1 subscription, got %d", len(subs))
	}
}

func TestUpdate_Success(t *testing.T) {
	repo, mock := setup(t)

	mock.ExpectExec(`UPDATE subscriptions`).
		WithArgs("Netflix", 999, sqlmock.AnyArg(), sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(context.Background(), "1", "Netflix", 999, mustParse("01-2024"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo, mock := setup(t)

	mock.ExpectExec(`UPDATE subscriptions`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "999").
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	err := repo.Update(context.Background(), "999", "Netflix", 999, mustParse("01-2024"), nil)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	repo, mock := setup(t)

	mock.ExpectExec(`DELETE FROM subscriptions`).
		WithArgs("1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Delete(context.Background(), "1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo, mock := setup(t)

	mock.ExpectExec(`DELETE FROM subscriptions`).
		WithArgs("999").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(context.Background(), "999")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetPeriodsByFilter_Success(t *testing.T) {
	repo, mock := setup(t)

	end := mustParse("12-2024")
	rows := sqlmock.NewRows([]string{"price", "start_date", "end_date"}).
		AddRow(799, mustParse("01-2024"), end)

	mock.ExpectQuery(`SELECT price, start_date, end_date FROM subscriptions`).
		WithArgs(validUUID, "Netflix").
		WillReturnRows(rows)

	periods, err := repo.GetPeriodsByFilter(context.Background(), validUUID, "Netflix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(periods) != 1 {
		t.Errorf("expected 1 period, got %d", len(periods))
	}
	if periods[0].Price != 799 {
		t.Errorf("expected price=799, got %d", periods[0].Price)
	}
}

func TestGetPeriodsByFilter_Empty(t *testing.T) {
	repo, mock := setup(t)

	mock.ExpectQuery(`SELECT price, start_date, end_date FROM subscriptions`).
		WithArgs(validUUID, "").
		WillReturnRows(sqlmock.NewRows([]string{"price", "start_date", "end_date"}))

	periods, err := repo.GetPeriodsByFilter(context.Background(), validUUID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(periods) != 0 {
		t.Errorf("expected 0 periods, got %d", len(periods))
	}
}
