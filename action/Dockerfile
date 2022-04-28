FROM golang:1.18 as builder
WORKDIR /app
COPY . .
RUN go build -o /sync /app/sync

FROM gcr.io/distroless/base-debian11
WORKDIR /
COPY --from=builder /sync /sync
ENTRYPOINT ["/sync"]