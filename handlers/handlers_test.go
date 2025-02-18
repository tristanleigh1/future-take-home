package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"future-take-home/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var h *Handler

func setupTestDB() (func(), error) {
	dbOptions := "host=db user=postgres password=postgres dbname=future_test port=5432 sslmode=disable TimeZone=UTC"
	var err error
	db, err = gorm.Open(postgres.Open(dbOptions), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Appointment{}); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	h = &Handler{DB: db}

	cleanup := func() {
		db.Exec("DELETE FROM appointments")
	}

	return cleanup, nil
}

func setupTestApp() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.Use(AuthMiddleware)
	router.GET("/appointments", h.GetAvailableAppointments)
	router.POST("/appointments", h.CreateAppointment)
	router.GET("/appointments/trainer/:trainer_id", h.GetScheduledAppointments)
	return router
}

func TestAuthMiddleware(t *testing.T) {
	cleanup, err := setupTestDB()
	if err != nil {
		t.Fatalf("failed to set up test DB: %v", err)
	}
	defer cleanup()

	router := setupTestApp()
	token := os.Getenv("SERVICE_TOKEN")

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{
			name:       "valid token",
			token:      token,
			wantStatus: 200,
		},
		{
			name:       "missing token",
			token:      "",
			wantStatus: 401,
		},
		{
			name:       "invalid token format",
			token:      "not-a-bearer-token",
			wantStatus: 401,
		},
		{
			name:       "wrong token",
			token:      "Bearer wrong-token",
			wantStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/appointments?trainer_id=1&starts_at=2019-01-24T16:00:00Z&ends_at=2019-01-24T17:00:00Z", nil)
			if tt.token != "" {
				if strings.HasPrefix(tt.token, "Bearer ") {
					req.Header.Set("Authorization", tt.token)
				} else {
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tt.token))
				}
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestGetAvailableAppointments(t *testing.T) {
	cleanup, err := setupTestDB()
	if err != nil {
		t.Fatalf("failed to set up test DB: %v", err)
	}
	defer cleanup()

	router := setupTestApp()
	token := os.Getenv("SERVICE_TOKEN")

	appointment := models.Appointment{
		TrainerID: 1,
		UserID:    100,
		StartsAt:  time.Date(2019, 1, 24, 9, 0, 0, 0, pacificTZ),
		EndsAt:    time.Date(2019, 1, 24, 9, 30, 0, 0, pacificTZ),
	}

	db.Create(&appointment)

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantSlots  int
	}{
		{
			name:       "PT timezone input",
			query:      "trainer_id=1&starts_at=2019-01-24T08:00:00-08:00&ends_at=2019-01-24T17:00:00-08:00",
			wantStatus: 200,
			wantSlots:  17, // 8am-5pm minus one booked slot
		},
		{
			name:       "UTC timezone input",
			query:      "trainer_id=1&starts_at=2019-01-24T16:00:00Z&ends_at=2019-01-25T01:00:00Z",
			wantStatus: 200,
			wantSlots:  17, // 8am-5pm minus one booked slot
		},
		{
			name:       "PT weekend day",
			query:      "trainer_id=1&starts_at=2019-01-20T08:00:00-08:00&ends_at=2019-01-20T17:00:00-08:00",
			wantStatus: 200,
			wantSlots:  0, // weekend, no slots available
		},
		{
			name:       "UTC weekend day",
			query:      "trainer_id=1&starts_at=2019-01-20T16:00:00Z&ends_at=2019-01-20T17:00:00Z",
			wantStatus: 200,
			wantSlots:  0, // weekend, no slots available
		},
		{
			name:       "PT outside business hours",
			query:      "trainer_id=1&starts_at=2019-01-24T07:00:00-08:00&ends_at=2019-01-24T08:00:00-08:00",
			wantStatus: 200,
			wantSlots:  0, // outside business hours, no slots available
		},
		{
			name:       "UTC outside business hours",
			query:      "trainer_id=1&starts_at=2019-01-24T11:00:00Z&ends_at=2019-01-24T12:00:00Z",
			wantStatus: 200,
			wantSlots:  0, // outside business hours, no slots available
		},
		{
			name:       "invalid date format",
			query:      "trainer_id=1&starts_at=invalid&ends_at=2019-01-24T17:00:00",
			wantStatus: 400,
			wantSlots:  0,
		},
		{
			name:       "missing trainer_id",
			query:      "starts_at=2019-01-24T16:00:00Z&ends_at=2019-01-24T17:00:00Z",
			wantStatus: 400,
			wantSlots:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/appointments?"+tt.query, nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == 200 {
				var slots []AvailableSlot
				err := json.NewDecoder(w.Body).Decode(&slots)
				assert.NoError(t, err)
				assert.Len(t, slots, tt.wantSlots)
			}
		})
	}
}

func TestCreateAppointment(t *testing.T) {
	cleanup, err := setupTestDB()
	if err != nil {
		t.Fatalf("failed to set up test DB: %v", err)
	}
	defer cleanup()

	router := setupTestApp()
	token := os.Getenv("SERVICE_TOKEN")

	validAppointment := map[string]interface{}{
		"trainer_id": 1,
		"user_id":    100,
		"starts_at":  "2019-01-24T09:00:00-08:00",
		"ends_at":    "2019-01-24T09:30:00-08:00",
	}

	jsonData, _ := json.Marshal(validAppointment)
	req := httptest.NewRequest(http.MethodPost, "/appointments", bytes.NewReader(jsonData))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Edge cases
	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "double booking same slot",
			body:       validAppointment,
			wantStatus: 409,
			wantError:  "Appointment slot is already booked",
		},
		{
			name: "non-business hours PT",
			body: map[string]interface{}{
				"trainer_id": 1,
				"user_id":    101,
				"starts_at":  "2019-01-24T06:00:00-08:00",
				"ends_at":    "2019-01-24T06:30:00-08:00",
			},
			wantStatus: 400,
			wantError:  "Appointments must be during business hours (M-F 8am-5pm PT)",
		},
		{
			name: "non-business hours UTC",
			body: map[string]interface{}{
				"trainer_id": 1,
				"user_id":    101,
				"starts_at":  "2019-01-24T11:00:00Z", // before 8am
				"ends_at":    "2019-01-24T11:30:00Z",
			},
			wantStatus: 400,
			wantError:  "Appointments must be during business hours (M-F 8am-5pm PT)",
		},
		{
			name: "invalid time slot PT",
			body: map[string]interface{}{
				"trainer_id": 1,
				"user_id":    102,
				"starts_at":  "2019-01-24T09:15:00-08:00", // not :00 or :30
				"ends_at":    "2019-01-24T09:45:00-08:00",
			},
			wantStatus: 400,
			wantError:  "Appointments must start at :00 or :30 minutes",
		},
		{
			name: "invalid time slot UTC",
			body: map[string]interface{}{
				"trainer_id": 1,
				"user_id":    102,
				"starts_at":  "2019-01-24T17:15:00Z", // not :00 or :30
				"ends_at":    "2019-01-24T17:45:00Z",
			},
			wantStatus: 400,
			wantError:  "Appointments must start at :00 or :30 minutes",
		},
		{
			name: "wrong duration PT",
			body: map[string]interface{}{
				"trainer_id": 1,
				"user_id":    103,
				"starts_at":  "2019-01-24T09:00:00-08:00",
				"ends_at":    "2019-01-24T09:45:00-08:00",
			},
			wantStatus: 400,
			wantError:  "Appointments must be exactly 30 minutes long",
		},
		{
			name: "wrong duration UTC",
			body: map[string]interface{}{
				"trainer_id": 1,
				"user_id":    103,
				"starts_at":  "2019-01-24T17:00:00Z",
				"ends_at":    "2019-01-24T18:00:00Z", // 1 hour
			},
			wantStatus: 400,
			wantError:  "Appointments must be exactly 30 minutes long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/appointments", bytes.NewReader(jsonData))
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantError != "" {
				assert.Contains(t, w.Body.String(), tt.wantError)
			}
		})
	}
}

func TestGetScheduledAppointments(t *testing.T) {
	cleanup, err := setupTestDB()
	if err != nil {
		t.Fatalf("failed to set up test DB: %v", err)
	}
	defer cleanup()

	router := setupTestApp()
	token := os.Getenv("SERVICE_TOKEN")

	appointment := models.Appointment{
		TrainerID: 1,
		UserID:    100,
		StartsAt:  time.Date(2019, 1, 24, 17, 0, 0, 0, time.UTC),
		EndsAt:    time.Date(2019, 1, 24, 17, 30, 0, 0, time.UTC),
	}

	db.Create(&appointment)

	tests := []struct {
		name       string
		trainerID  string
		wantStatus int
		wantCount  int
	}{
		{
			name:       "trainer with appointments",
			trainerID:  "1",
			wantStatus: 200,
			wantCount:  1, // just the one we created
		},
		{
			name:       "trainer without appointments",
			trainerID:  "999",
			wantStatus: 200,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/appointments/trainer/"+tt.trainerID, nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == 200 {
				var appointments []map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&appointments)
				assert.NoError(t, err)
				assert.Len(t, appointments, tt.wantCount)
			}
		})
	}
}
