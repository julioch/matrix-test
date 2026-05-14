FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api-go .

FROM alpine:3.20

WORKDIR /app
COPY --from=build /out/api-go /app/api-go

ENV PORT=8080
EXPOSE 8080

CMD ["/app/api-go"]
