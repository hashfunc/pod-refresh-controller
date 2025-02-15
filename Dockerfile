# Build Stage
FROM golang:1.23 AS builder

WORKDIR /opt/build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o controller cmd/controller/main.go

# Runtime
FROM gcr.io/distroless/static:nonroot

WORKDIR /opt/controller

COPY --from=builder /opt/build/controller .
USER 65532:65532

ENTRYPOINT ["/opt/controller/controller"] 
