FROM golang:1.24.0

WORKDIR /usr/src/app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Install Air for hot reloading
RUN go install github.com/air-verse/air@latest

# Install test dependencies
RUN go get github.com/stretchr/testify/assert

# Copy the rest of the code
COPY . .

EXPOSE 3001

CMD ["air", "./cmd/main.go", "-b", "0.0.0.0"]
