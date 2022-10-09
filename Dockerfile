# Build stage
FROM golang:1.19 AS builder
WORKDIR /work
COPY . .
RUN go get -d -v .
RUN go build -o /dist/dapr-sqlite-statestore -v .

# Final stage
FROM gcr.io/distroless/base-debian11
COPY --from=builder /dist/dapr-sqlite-statestore /
CMD ["/dapr-sqlite-statestore"]
