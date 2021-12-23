FROM alpine as lb-proxy
ADD ./hack/lb-entrypoint.sh /entrypoint.sh
RUN apk add iptables && chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]

FROM golang:1.17 AS builder
WORKDIR /app
COPY . .
RUN GOOS=linux ARCH=amd64 go build -o /bin/loadbalancer cmd/loadbalancer/main.go

FROM alpine AS lb-controller
COPY --from=builder /bin/loadbalancer  /bin/loadbalancer
COPY --from=builder /app/config/loadbalancer.yaml  /etc/loadbalancer.yaml
ENTRYPOINT ["/bin/loadbalancer"]
