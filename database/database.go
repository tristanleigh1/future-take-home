package database

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"future-take-home/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type seedAppointment struct {
	ID        uint      `json:"id"`
	TrainerID int64     `json:"trainer_id"`
	UserID    int64     `json:"user_id"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

func seedDatabase(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Appointment{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Println("Database already contains data, skipping seed")
		return nil
	}

	data, err := os.ReadFile("database/seed.json")
	if err != nil {
		return err
	}

	var seedData []seedAppointment
	if err := json.Unmarshal(data, &seedData); err != nil {
		return err
	}

	appointments := make([]models.Appointment, len(seedData))
	for i, seed := range seedData {
		appointments[i] = models.Appointment{
			TrainerID: seed.TrainerID,
			UserID:    seed.UserID,
			StartsAt:  seed.StartedAt,
			EndsAt:    seed.EndedAt,
		}
	}

	return db.Create(&appointments).Error
}

func ConnectDb() *gorm.DB {
	dbOptions := "host=db user=postgres password=postgres dbname=future port=5432 sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dbOptions), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database. \n", err)
		os.Exit(2)
	}

	log.Println("Connected to database! Running migrations...")
	db.AutoMigrate(&models.Appointment{})

	if err := seedDatabase(db); err != nil {
		log.Fatal("Failed to seed database:", err)
	}

	return db
}
