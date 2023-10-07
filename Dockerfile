FROM golang:1.21 AS builder
WORKDIR /frp-service-provider
COPY . .
RUN GOOS=linux ARCH=amd64 go build -o /bin/controller cmd/controller/main.go

FROM ubuntu:22.04
COPY --from=builder /bin/controller-manager  /bin/controller-manager
COPY --from=builder /frp-service-provider/config/config.tpl.yaml  /etc/controller-manager.yaml
ENTRYPOINT ["/bin/controller-manager"]
