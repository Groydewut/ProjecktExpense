FROM golang:1.26.2 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# ИСПРАВЛЕНО: Добавлены CGO_ENABLED=0 и GOOS=linux для статической сборки
RUN CGO_ENABLED=0 GOOS=linux go build -o expense .

FROM alpine:latest
WORKDIR /
COPY --from=builder /app/expense .
EXPOSE 8080
CMD [ "./expense"]