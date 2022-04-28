FROM golang:1.18 as builder
WORKDIR /app
COPY . .
RUN go build -o /app/bot /app/cmd/sync

FROM gcr.io/distroless/base-debian11
WORKDIR /
COPY --from=builder /app/sync /sync
ENTRYPOINT ["/sync"]