# deployments/docker/prober.Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
# COPY go.sum ./ (Uncomment once you have dependencies)
RUN go mod download

COPY . .
# Statically compile the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o prober ./cmd/prober

FROM alpine:latest
# Required for making HTTPS requests
RUN apk --no-cache add ca-certificates 

WORKDIR /root/
COPY --from=builder /app/prober .

CMD ["./prober"]
