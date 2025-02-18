package main

import (
	"github.com/gin-gonic/gin"

	"future-take-home/database"
	"future-take-home/handlers"
)

func main() {
	db := database.ConnectDb()
	handler := handlers.Handler{DB: db}

	router := gin.Default()
	router.Use(handlers.AuthMiddleware)
	router.GET("/appointments", handler.GetAvailableAppointments)
	router.POST("/appointments", handler.CreateAppointment)
	router.GET("/appointments/trainer/:trainer_id", handler.GetScheduledAppointments)
	router.Run(":3001")
}
