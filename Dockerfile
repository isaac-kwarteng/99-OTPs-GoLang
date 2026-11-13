# Start from golang base image
FROM golang:alpine as builder

# Add Maintainer info.
LABEL maintainer="Gilbert Amo <gilberyt.amo@deftpal.com>"

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git && apk add --no-cache bash && apk add build-base

# Set the current working directory inside the container.
WORKDIR /app

# Copy go mod and sum files.
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and the go.sum files are not changed.
RUN go mod download && go mod verify

# Copy the source from the current directory to the working directory inside the container.
COPY . .

# Build the Go api
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go

# Start a new stage from scratch
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage. Observe we also copied the .env file
COPY --from=builder /app/main .

# Expose port to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
