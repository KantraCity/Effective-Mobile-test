package model

import "time"

type Subscription struct {
	ID          int        `json:"id"`
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	UserID      string     `json:"user_id"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

type CreateSubscriptionRequest struct {
	ServiceName string `json:"service_name" binding:"required,max=255"`
	Price       int    `json:"price"        binding:"required,gt=0"`
	UserID      string `json:"user_id"      binding:"required"`
	StartDate   string `json:"start_date"   binding:"required"`
	EndDate     string `json:"end_date"`
}

type UpdateSubscriptionRequest struct {
	ServiceName string `json:"service_name" binding:"required"`
	Price       int    `json:"price"        binding:"required,gt=0"`
	StartDate   string `json:"start_date"   binding:"required"`
	EndDate     string `json:"end_date"`
}

type SubCalcData struct {
	Price     int
	StartDate time.Time
	EndDate   *time.Time
}
