# Appointment Scheduling API

A RESTful API for managing trainer appointments. Built with Go, Fiber, GORM, and PostgreSQL. Uses Docker for containerization and Air for hot reloading during development.

## Features
- Schedule 30-minute appointments during business hours (M-F 8am-5pm PT)
- View available appointment slots for a given trainer and time range
- View scheduled appointments for a given trainer
- Automatic timezone handling (all times returned in Pacific Time, but can handle any timezone for input)
- Prevents double-booking

## Project Structure
```
.
├── cmd/
│ ├── main.go # Application entry point
├── database/
│ └── database.go # Database connection and utilities
│ └── seed.json # Seed data for testing
├── handlers/
│ └── handlers_test.go # Test for appointment handlers
│ └── handlers.go # Appointment request handlers
│ └── middleware.go # Middleware for authentication
├── models/
│ └── models.go # Data models
```

## Running the Application

### Prerequisites
- Docker
- Docker Compose

### Environment Setup
Create a `.env` file in the root directory:

```
SERVICE_TOKEN=your-secret-token-here  # Required for API authentication
```

### Take Down the Application

```bash
docker compose down
```

### Build and Run the Application

```bash
docker compose up -d --build
```

The API will be available at `http://localhost:3001`

## Authentication
All API endpoints require a service token passed in the Authorization header:
```
Authorization: Bearer SERVICE_TOKEN
```

## API Endpoints

### Get Available Appointments

```bash
curl -X GET 'http://localhost:3001/appointments?trainer_id=1&starts_at=2019-01-24T16:00:00Z&ends_at=2019-01-25T01:00:00Z' \
  -H 'Authorization: Bearer SERVICE_TOKEN'
```

Returns available 30-minute slots between the given dates for a trainer.

Response (200 OK):
```
[
  {
    "starts_at": "2019-01-24T16:00:00Z",
    "ends_at": "2019-01-24T16:30:00Z"
  },
  ...
  {
    "starts_at": "2019-01-25T00:30:00Z",
    "ends_at": "2019-01-25T01:00:00Z"
  }
]
```

### Create Appointment

```bash
curl -X POST 'http://localhost:3001/appointments' \
  -H 'Authorization: Bearer SERVICE_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "trainer_id": 1,
    "user_id": 2,
    "starts_at": "2019-01-24T17:00:00Z",
    "ends_at": "2019-01-24T17:30:00Z"
  }'
```

Creates a new appointment.

Response (201 Created):
```
{
  "id": 1,
  "trainer_id": 1,
  "user_id": 2,
  "starts_at": "2019-01-24T17:00:00Z",
  "ends_at": "2019-01-24T17:30:00Z"
}
```

### Get Scheduled Appointments

```bash
curl -X GET 'http://localhost:3001/appointments/trainer/1' \
  -H 'Authorization: Bearer SERVICE_TOKEN'
```

Returns all scheduled appointments for a given trainer.

Response (200 OK):
```
[
  {
    "id": 1,
    "trainerId": 1,
    "userId": 2,
    "startsAt": "2019-01-24T17:00:00Z",
    "endsAt": "2019-01-24T17:30:00Z"
  }
]
```

## Testing

Run the tests:
```bash
docker compose exec app go test ./handlers -v
```

The tests use a separate test database to verify:
- Business hour validation
- Appointment slot availability
- Double booking prevention
- Authentication
