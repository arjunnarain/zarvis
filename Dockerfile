# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: Build backend
FROM golang:1.25-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -o /zarvis .

# Stage 3: Production image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates poppler-utils
WORKDIR /app

COPY --from=backend /zarvis /app/zarvis
COPY --from=frontend /app/frontend/dist /app/static
COPY docs/ /app/docs/

ENV ZARVIS_DB=/app/data/zarvis.db
ENV ZARVIS_STATIC_DIR=/app/static
ENV ZARVIS_DOCS_DIR=/app/docs

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["/app/zarvis"]
