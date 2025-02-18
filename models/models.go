package models

import (
	"time"

	"gorm.io/gorm"
)

type Appointment struct {
	gorm.Model
	TrainerID int64     `json:"trainerId" gorm:"column:trainer_id;not null;index:idx_trainer_start,unique:trainer_start"`
	UserID    int64     `json:"userId" gorm:"column:user_id;not null"`
	StartsAt  time.Time `json:"startsAt" gorm:"column:starts_at;type:timestamp with time zone;not null;index:idx_trainer_start,unique:trainer_start"`
	EndsAt    time.Time `json:"endsAt" gorm:"column:ends_at;type:timestamp with time zone;not null"`
}
