# Stage 1: build binary (optional if Makefile already builds)
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN apk add --no-cache git build-base
RUN make build

# Stage 2: minimal runtime image
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /app/build/yourapp /app/yourapp
USER nonroot:nonroot
ENTRYPOINT ["/app/yourapp"]
