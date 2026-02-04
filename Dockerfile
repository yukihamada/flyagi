# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.24-alpine AS backend-builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 3: Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY --from=backend-builder /server /app/server
COPY --from=frontend-builder /app/web/dist /app/web/dist

EXPOSE 8080

CMD ["/app/server"]
