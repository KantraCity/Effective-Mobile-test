package handler

import (
	"Testwork/internal/model"
	"Testwork/internal/service"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type SubscriptionService interface {
	Create(ctx context.Context, req model.CreateSubscriptionRequest) (int, error)
	Update(ctx context.Context, id string, req model.UpdateSubscriptionRequest) error
	List(ctx context.Context, userID string) ([]model.Subscription, error)
	GetByID(ctx context.Context, id string) (model.Subscription, error)
	Delete(ctx context.Context, id string) error
	CalculateTotal(ctx context.Context, userID, serviceName string, from, to time.Time) (int, error)
}

type Handler struct {
	srv SubscriptionService
	log zerolog.Logger
}

func NewHandler(srv SubscriptionService, log zerolog.Logger) *Handler {
	return &Handler{
		srv: srv,
		log: log.With().Str("component", "handler").Logger(),
	}
}

// @Summary Создать подписку
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param input body model.CreateSubscriptionRequest true "Данные подписки"
// @Success 201 {object} map[string]int "ID созданной подписки"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions [post]
func (h *Handler) CreateSubscription(c *gin.Context) {
	var input model.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		h.log.Warn().Err(err).Msg("failed to bind JSON in CreateSubscription")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}

	if _, err := uuid.Parse(input.UserID); err != nil {
		h.log.Warn().Str("user_id", input.UserID).Msg("invalid user_id format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format, expected UUID"})
		return
	}

	id, err := h.srv.Create(c.Request.Context(), input)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to create subscription")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Info().Int("id", id).Msg("subscription created successfully")
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// @Summary Обновить подписку
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path int true "ID подписки"
// @Param input body model.UpdateSubscriptionRequest true "Новые данные подписки"
// @Success 200 {object} map[string]string "Успешно обновлено"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 404 {object} map[string]string "Подписка не найдена"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions/{id} [put]
func (h *Handler) UpdateSubscription(c *gin.Context) {
	id := c.Param("id")

	var input model.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		h.log.Warn().Err(err).Str("id", id).Msg("failed to bind JSON in UpdateSubscription")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}

	if err := h.srv.Update(c.Request.Context(), id, input); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		h.log.Error().Err(err).Str("id", id).Msg("failed to update subscription")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Info().Str("id", id).Msg("subscription updated successfully")
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// @Summary Подсчитать стоимость за период
// @Tags subscriptions
// @Produce json
// @Param user_id query string true "UUID пользователя"
// @Param service_name query string false "Название сервиса"
// @Param from query string true "Начало периода (MM-YYYY)"
// @Param to query string true "Конец периода (MM-YYYY)"
// @Success 200 {object} map[string]int "Общая стоимость"
// @Failure 400 {object} map[string]string "Неверные параметры"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions/total [get]
func (h *Handler) GetTotalCost(c *gin.Context) {
	userID := c.Query("user_id")
	fromStr := c.Query("from")
	toStr := c.Query("to")
	serviceName := c.Query("service_name")

	if userID == "" || fromStr == "" || toStr == "" {
		h.log.Warn().Msg("missing required query params in GetTotalCost")
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id, from and to are required"})
		return
	}

	if _, err := uuid.Parse(userID); err != nil {
		h.log.Warn().Str("user_id", userID).Msg("invalid user_id format in GetTotalCost")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format, expected UUID"})
		return
	}

	layout := "01-2006"
	from, errF := time.Parse(layout, fromStr)
	to, errT := time.Parse(layout, toStr)
	if errF != nil || errT != nil {
		h.log.Warn().Msg("invalid date format in GetTotalCost")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use MM-YYYY"})
		return
	}

	if from.After(to) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'from' must not be after 'to'"})
		return
	}

	total, err := h.srv.CalculateTotal(c.Request.Context(), userID, serviceName, from, to)
	if err != nil {
		h.log.Error().Err(err).Str("user_id", userID).Msg("failed to calculate total cost")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.log.Info().Int("total_cost", total).Str("user_id", userID).Msg("total cost calculated")
	c.JSON(http.StatusOK, gin.H{"total_cost": total})
}

// @Summary Список подписок
// @Tags subscriptions
// @Produce json
// @Param user_id query string false "UUID пользователя"
// @Success 200 {array} model.Subscription "Список подписок"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions [get]
func (h *Handler) ListSubscriptions(c *gin.Context) {
	userID := c.Query("user_id")

	if userID != "" {
		if _, err := uuid.Parse(userID); err != nil {
			h.log.Warn().Str("user_id", userID).Msg("invalid user_id format in ListSubscriptions")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id format, expected UUID"})
			return
		}
	}

	subs, err := h.srv.List(c.Request.Context(), userID)
	if err != nil {
		h.log.Error().Err(err).Msg("failed to list subscriptions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.log.Debug().Int("count", len(subs)).Msg("subscriptions listed")
	c.JSON(http.StatusOK, subs)
}

// @Summary Удалить подписку
// @Tags subscriptions
// @Param id path int true "ID подписки"
// @Success 204 "Подписка удалена"
// @Failure 404 {object} map[string]string "Подписка не найдена"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions/{id} [delete]
func (h *Handler) DeleteSubscription(c *gin.Context) {
	id := c.Param("id")
	h.log.Info().Str("id", id).Msg("delete subscription request received")

	if err := h.srv.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		h.log.Error().Err(err).Str("id", id).Msg("failed to delete subscription")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.log.Info().Str("id", id).Msg("subscription deleted")
	c.Status(http.StatusNoContent)
}

// @Summary Получить подписку по ID
// @Tags subscriptions
// @Produce json
// @Param id path int true "ID подписки"
// @Success 200 {object} model.Subscription "Данные подписки"
// @Failure 404 {object} map[string]string "Подписка не найдена"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions/{id} [get]
func (h *Handler) GetSubscription(c *gin.Context) {
	id := c.Param("id")
	h.log.Debug().Str("id", id).Msg("get subscription by id request received")

	sub, err := h.srv.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		h.log.Error().Err(err).Str("id", id).Msg("failed to get subscription")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.log.Debug().Str("id", id).Msg("subscription retrieved")
	c.JSON(http.StatusOK, sub)
}
