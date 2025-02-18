package handlers

import (
	"net/http"
	"strings"
	"time"

	"future-take-home/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	duration  = 30 * time.Minute
	startHour = 8
	endHour   = 17
	timezone  = "America/Los_Angeles"
)

var pacificTZ, _ = time.LoadLocation(timezone)

type Handler struct {
	DB *gorm.DB
}

// Checks whether given time falls within M-F 8am-5pm PT
func isBusinessHours(t time.Time) bool {
	ptTime := t.In(pacificTZ)
	return ptTime.Weekday() != time.Saturday &&
		ptTime.Weekday() != time.Sunday &&
		ptTime.Hour() >= startHour &&
		ptTime.Hour() < endHour
}

type AvailableSlot struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

func (h *Handler) GetAvailableAppointments(c *gin.Context) {
	trainerID := c.Query("trainer_id")
	if trainerID == "" || c.Query("starts_at") == "" || c.Query("ends_at") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required query parameters: trainer_id, starts_at, ends_at"})
		return
	}

	startsAt, startsAtErr := time.Parse(time.RFC3339, c.Query("starts_at"))
	endsAt, endsAtErr := time.Parse(time.RFC3339, c.Query("ends_at"))

	if startsAtErr != nil || endsAtErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format for starts_at or ends_at"})
		return
	}

	booked := []models.Appointment{}
	if err := h.DB.Where("trainer_id = ? AND starts_at >= ? AND starts_at < ?",
		trainerID, startsAt.UTC(), endsAt.UTC()).Find(&booked).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch appointments"})
		return
	}

	bookedTimes := make(map[string]bool)
	for _, apt := range booked {
		bookedTimes[apt.StartsAt.UTC().Format(time.RFC3339)] = true
	}

	available := []AvailableSlot{}
	for current := startsAt; current.Before(endsAt); current = current.Add(duration) {
		if isBusinessHours(current) && !bookedTimes[current.UTC().Format(time.RFC3339)] {
			slot := AvailableSlot{
				StartsAt: current,
				EndsAt:   current.Add(duration),
			}
			available = append(available, slot)
		}
	}

	response := make([]AvailableSlot, len(available))
	for i, slot := range available {
		response[i] = AvailableSlot{
			StartsAt: slot.StartsAt.In(pacificTZ),
			EndsAt:   slot.EndsAt.In(pacificTZ),
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreateAppointment(c *gin.Context) {
	var req struct {
		TrainerID int64  `json:"trainer_id"`
		UserID    int64  `json:"user_id"`
		StartsAt  string `json:"starts_at"`
		EndsAt    string `json:"ends_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startsAt, startsAtErr := time.Parse(time.RFC3339, req.StartsAt)
	endsAt, endsAtErr := time.Parse(time.RFC3339, req.EndsAt)

	if startsAtErr != nil || endsAtErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format for starts_at or ends_at"})
		return
	}

	if !isBusinessHours(startsAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Appointments must be during business hours (M-F 8am-5pm PT)"})
		return
	}

	if (startsAt.Minute() % 30) != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Appointments must start at :00 or :30 minutes"})
		return
	}

	if endsAt.Sub(startsAt) != duration {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Appointments must be exactly 30 minutes long"})
		return
	}

	appointment := models.Appointment{
		TrainerID: req.TrainerID,
		UserID:    req.UserID,
		StartsAt:  startsAt.UTC(),
		EndsAt:    endsAt.UTC(),
	}

	if err := h.DB.Create(&appointment).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Appointment slot is already booked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create appointment"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         appointment.ID,
		"trainer_id": appointment.TrainerID,
		"user_id":    appointment.UserID,
		"starts_at":  appointment.StartsAt.In(pacificTZ),
		"ends_at":    appointment.EndsAt.In(pacificTZ),
	})
}

func (h *Handler) GetScheduledAppointments(c *gin.Context) {
	var appointments []models.Appointment
	trainerID := c.Param("trainer_id")
	if trainerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required query parameters: trainer_id"})
		return
	}

	if err := h.DB.Where("trainer_id = ?", trainerID).Find(&appointments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch appointments"})
		return
	}

	response := make([]gin.H, len(appointments))
	for i, apt := range appointments {
		response[i] = gin.H{
			"id":         apt.ID,
			"starts_at":  apt.StartsAt.In(pacificTZ),
			"ends_at":    apt.EndsAt.In(pacificTZ),
			"trainer_id": apt.TrainerID,
			"user_id":    apt.UserID,
		}
	}

	c.JSON(http.StatusOK, response)
}
