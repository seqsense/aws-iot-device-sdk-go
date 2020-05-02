# =====================
FROM golang:1.14-alpine3.11 as go-builder

ENV CGO_ENABLED=0

COPY . /go/src/github.com/seqsense/aws-iot-device-sdk-go
WORKDIR /go/src/github.com/seqsense/aws-iot-device-sdk-go/tunnel/server/cmd/secure-tunnel-server
RUN go build -tags netgo -installsuffix netgo
RUN cp secure-tunnel-server /usr/local/bin/

# =====================
FROM alpine:3.11

COPY --from=go-builder /usr/local/bin/secure-tunnel-server /usr/local/bin/

CMD ["/usr/local/bin/secure-tunnel-server"]
