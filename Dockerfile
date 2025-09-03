# Using Go with Alpine Linux
FROM golang:1.20-alpine 

# Set the working directory inside the container
WORKDIR /app

# Install dependencies for mattn/go-sqlite3
RUN apk add --no-cache gcc musl-dev sqlite-dev  

# Copy go mod and sum files
COPY go.mod go.sum ./   
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o forum main.go

# Expose the port the app runs on
EXPOSE 8080

# Command to run the executable
CMD ["./forum"]