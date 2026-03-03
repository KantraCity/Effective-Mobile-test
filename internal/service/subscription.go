package service

import (
	"Testwork/internal/model"
	"Testwork/internal/repository"
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
)

var ErrNotFound = repository.ErrNotFound

type SubscriptionRepo interface {
	Create(ctx context.Context, serviceName string, price int, userID string, start time.Time, end *time.Time) (int, error)
	GetPeriodsByFilter(ctx context.Context, userID string, serviceName string) ([]model.SubCalcData, error)
	List(ctx context.Context, userID string) ([]model.Subscription, error)
	GetByID(ctx context.Context, id string) (model.Subscription, error)
	Update(ctx context.Context, id, serviceName string, price int, start time.Time, end *time.Time) error
	Delete(ctx context.Context, id string) error
}

type SubscriptionService struct {
	repo SubscriptionRepo
	log  zerolog.Logger
}

func NewSubscriptionService(repo *repository.SubscriptionRepository, log zerolog.Logger) *SubscriptionService {
	return &SubscriptionService{repo: repo, log: log.With().Str("component", "service").Logger()}
}

func NewSubscriptionServiceWithRepo(repo SubscriptionRepo, log zerolog.Logger) *SubscriptionService {
	return &SubscriptionService{repo: repo, log: log.With().Str("component", "service").Logger()}
}

func (s *SubscriptionService) Create(ctx context.Context, req model.CreateSubscriptionRequest) (int, error) {
	s.log.Debug().Str("user_id", req.UserID).Str("service", req.ServiceName).Msg("Create subscription request received")

	startDate, err := time.Parse("01-2006", req.StartDate)
	if err != nil {
		s.log.Warn().Err(err).Str("start_date", req.StartDate).Msg("invalid start date format")
		return 0, errors.New("invalid start_date format, use MM-YYYY")
	}

	var endDate *time.Time
	if req.EndDate != "" {
		parsedEnd, err := time.Parse("01-2006", req.EndDate)
		if err != nil {
			s.log.Warn().Err(err).Str("end_date", req.EndDate).Msg("invalid end date format")
			return 0, errors.New("invalid end_date format, use MM-YYYY")
		}
		if !parsedEnd.After(startDate) {
			return 0, errors.New("end_date must be after start_date")
		}
		endDate = &parsedEnd
	}

	id, err := s.repo.Create(ctx, req.ServiceName, req.Price, req.UserID, startDate, endDate)
	if err != nil {
		s.log.Error().Err(err).Str("user_id", req.UserID).Msg("failed to create subscription")
		return 0, err
	}
	s.log.Info().Int("id", id).Str("user_id", req.UserID).Msg("subscription created successfully")
	return id, nil
}

func (s *SubscriptionService) Update(ctx context.Context, id string, req model.UpdateSubscriptionRequest) error {
	s.log.Debug().Str("id", id).Str("service", req.ServiceName).Msg("Update subscription request received")

	startDate, err := time.Parse("01-2006", req.StartDate)
	if err != nil {
		s.log.Warn().Err(err).Msg("invalid start date format in update")
		return errors.New("invalid start_date format, use MM-YYYY")
	}

	var endDate *time.Time
	if req.EndDate != "" {
		parsedEnd, err := time.Parse("01-2006", req.EndDate)
		if err != nil {
			s.log.Warn().Err(err).Msg("invalid end date format in update")
			return errors.New("invalid end_date format, use MM-YYYY")
		}
		if !parsedEnd.After(startDate) {
			return errors.New("end_date must be after start_date")
		}
		endDate = &parsedEnd
	}

	if err := s.repo.Update(ctx, id, req.ServiceName, req.Price, startDate, endDate); err != nil {
		s.log.Error().Err(err).Str("id", id).Msg("failed to update subscription")
		return err
	}
	s.log.Info().Str("id", id).Msg("subscription updated successfully")
	return nil
}

func (s *SubscriptionService) CalculateTotal(ctx context.Context, userID, serviceName string, from, to time.Time) (int, error) {
	s.log.Debug().Str("user_id", userID).Time("from", from).Time("to", to).Msg("CalculateTotal request received")

	periods, err := s.repo.GetPeriodsByFilter(ctx, userID, serviceName)
	if err != nil {
		s.log.Error().Err(err).Str("user_id", userID).Msg("failed to fetch periods")
		return 0, err
	}

	total := 0
	for _, p := range periods {
		actualStart := p.StartDate
		if from.After(actualStart) {
			actualStart = from
		}
		actualEnd := to
		if p.EndDate != nil && p.EndDate.Before(to) {
			actualEnd = *p.EndDate
		}
		if actualStart.After(actualEnd) {
			continue
		}
		months := (actualEnd.Year()-actualStart.Year())*12 + int(actualEnd.Month()-actualStart.Month()) + 1
		total += months * p.Price
	}

	s.log.Info().Int("total_cost", total).Str("user_id", userID).Msg("total cost calculated")
	return total, nil
}

func (s *SubscriptionService) List(ctx context.Context, userID string) ([]model.Subscription, error) {
	s.log.Debug().Str("user_id", userID).Msg("List subscriptions request received")
	subs, err := s.repo.List(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list subscriptions")
		return nil, err
	}
	return subs, nil
}

func (s *SubscriptionService) GetByID(ctx context.Context, id string) (model.Subscription, error) {
	s.log.Debug().Str("id", id).Msg("GetByID request received")
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.log.Warn().Err(err).Str("id", id).Msg("failed to get subscription")
		return model.Subscription{}, err
	}
	return sub, nil
}

func (s *SubscriptionService) Delete(ctx context.Context, id string) error {
	s.log.Info().Str("id", id).Msg("Delete subscription request received")
	if err := s.repo.Delete(ctx, id); err != nil {
		s.log.Error().Err(err).Str("id", id).Msg("failed to delete subscription")
		return err
	}
	s.log.Info().Str("id", id).Msg("subscription deleted successfully")
	return nil
}
