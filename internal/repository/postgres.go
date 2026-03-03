package repository

import (
	"Testwork/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

var ErrNotFound = errors.New("subscription not found")

type SubscriptionRepository struct {
	db  *sql.DB
	log zerolog.Logger
}

func NewSubscriptionRepository(db *sql.DB, log zerolog.Logger) *SubscriptionRepository {
	return &SubscriptionRepository{
		db:  db,
		log: log.With().Str("component", "repository").Logger(),
	}
}

func (r *SubscriptionRepository) Create(ctx context.Context, serviceName string, price int, userID string, start time.Time, end *time.Time) (int, error) {
	var id int
	query := `
        INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id`

	r.log.Debug().Str("service", serviceName).Str("user", userID).Msg("executing insert query")
	err := r.db.QueryRowContext(ctx, query, serviceName, price, userID, start, end).Scan(&id)
	if err != nil {
		r.log.Error().Err(err).Str("user", userID).Msg("failed to insert subscription")
		return 0, fmt.Errorf("failed to insert subscription: %w", err)
	}
	return id, nil
}

func (r *SubscriptionRepository) GetPeriodsByFilter(ctx context.Context, userID string, serviceName string) ([]model.SubCalcData, error) {
	query := `
        SELECT price, start_date, end_date
        FROM subscriptions
        WHERE user_id = $1 AND ($2 = '' OR service_name = $2)`

	r.log.Debug().Str("user_id", userID).Str("service", serviceName).Msg("fetching periods for calculation")

	rows, err := r.db.QueryContext(ctx, query, userID, serviceName)
	if err != nil {
		r.log.Error().Err(err).Msg("failed to execute query in GetPeriodsByFilter")
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error().Err(err).Msg("failed to close rows")
		}
	}()

	var periods []model.SubCalcData
	for rows.Next() {
		var p model.SubCalcData
		if err := rows.Scan(&p.Price, &p.StartDate, &p.EndDate); err != nil {
			r.log.Error().Err(err).Msg("failed to scan row in GetPeriodsByFilter")
			return nil, err
		}
		periods = append(periods, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return periods, nil
}

func (r *SubscriptionRepository) List(ctx context.Context, userID string) ([]model.Subscription, error) {
	query := `SELECT id, service_name, price, user_id, start_date, end_date FROM subscriptions`
	var args []interface{}
	if userID != "" {
		query += ` WHERE user_id = $1`
		args = append(args, userID)
	}

	r.log.Debug().Str("user_id", userID).Msg("listing subscriptions")

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.log.Error().Err(err).Msg("failed to list subscriptions")
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error().Err(err).Msg("failed to close rows")
		}
	}()

	subs := make([]model.Subscription, 0)
	for rows.Next() {
		var s model.Subscription
		if err := rows.Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &s.StartDate, &s.EndDate); err != nil {
			r.log.Error().Err(err).Msg("failed to scan subscription row")
			return nil, err
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return subs, nil
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id string) (model.Subscription, error) {
	var s model.Subscription
	query := `SELECT id, service_name, price, user_id, start_date, end_date FROM subscriptions WHERE id = $1`

	r.log.Debug().Str("id", id).Msg("getting subscription by id")

	err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &s.StartDate, &s.EndDate)
	if errors.Is(err, sql.ErrNoRows) {
		r.log.Warn().Str("id", id).Msg("subscription not found")
		return s, ErrNotFound
	}
	if err != nil {
		r.log.Error().Err(err).Str("id", id).Msg("query failed in GetByID")
		return s, err
	}
	return s, nil
}

func (r *SubscriptionRepository) Update(ctx context.Context, id string, serviceName string, price int, start time.Time, end *time.Time) error {
	query := `
        UPDATE subscriptions
        SET service_name = $1, price = $2, start_date = $3, end_date = $4
        WHERE id = $5`

	r.log.Info().Str("id", id).Msg("updating subscription")

	res, err := r.db.ExecContext(ctx, query, serviceName, price, start, end, id)
	if err != nil {
		r.log.Error().Err(err).Str("id", id).Msg("failed to update subscription")
		return err
	}
	count, _ := res.RowsAffected()
	r.log.Debug().Int64("rows_affected", count).Str("id", id).Msg("update completed")
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SubscriptionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM subscriptions WHERE id = $1`
	r.log.Info().Str("id", id).Msg("deleting subscription")

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.log.Error().Err(err).Str("id", id).Msg("failed to delete subscription")
		return err
	}
	count, _ := res.RowsAffected()
	if count == 0 {
		r.log.Warn().Str("id", id).Msg("subscription not found on delete")
		return ErrNotFound
	}
	return nil
}
