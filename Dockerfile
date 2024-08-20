# Stage 1: Build the frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY wishpage-app/ .
RUN npm install
RUN npm run build

# Stage 2: Build the backend
FROM golang:1.23 AS backend-builder
WORKDIR /app/backend
COPY wishpage-server/ .
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Stage 3: Final stage
FROM gcr.io/distroless/base-nossl-debian12
WORKDIR /app
COPY --from=frontend-builder /app/frontend/dist ./frontend
COPY --from=backend-builder /app/backend/main ./main

EXPOSE 8080
CMD ["./main"]