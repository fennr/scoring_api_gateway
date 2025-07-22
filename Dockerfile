# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder
WORKDIR /app
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org
COPY . .
RUN go mod download
RUN go build -o scoring_api_gateway .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/scoring_api_gateway /app/scoring_api_gateway
COPY --from=builder /app/migrations ./migrations
EXPOSE 8080
CMD ["/app/scoring_api_gateway"] 