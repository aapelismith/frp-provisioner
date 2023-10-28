FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o /bin/controller-manager cmd/controller-manager/main.go

FROM ubuntu:22.04
COPY --from=builder /bin/controller-manager  /bin/controller-manager
COPY --from=builder /app/config/config.toml  /etc/frp-provisioner/config.toml
ENTRYPOINT ["/bin/controller-manager"]
