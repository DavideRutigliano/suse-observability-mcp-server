# --- Build Stage ---
# Use a specific Go version from your go.mod file for reproducible builds.
FROM registry.suse.com/bci/golang:1.24 AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies.
# This is done as a separate step to leverage Docker's layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application's source code.
COPY . .

# Build the application.
RUN CGO_ENABLED=0 go build cmd/server/main.go -o /app/mcp

# --- Final Stage ---
FROM registry.suse.com/bci/bci-micro:15.7

# Copy the static binary from the build stage.
COPY --from=builder /app/mcp /suse-observability-mcp-server

# Expose the port that the HTTP server will listen on, as mentioned in the README.
EXPOSE 8080

# Set the entrypoint for the container.
# This allows you to pass command-line arguments when running the container.
# Example: docker run <image> -http :8080 -url "..." -token "..."
ENTRYPOINT ["/mcp"]
