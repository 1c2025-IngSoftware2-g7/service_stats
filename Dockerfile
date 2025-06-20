FROM golang:1.24-alpine

WORKDIR /app_service_stats
COPY . .
RUN go mod download
RUN go build -o service_stats .

EXPOSE 8080
CMD ["./service_stats"]