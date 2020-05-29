# =====================
FROM golang:1.14-alpine3.11 as go-builder

ENV CGO_ENABLED=0

COPY go.mod /go/src/github.com/seqsense/aws-iot-device-sdk-go/go.mod
COPY go.sum /go/src/github.com/seqsense/aws-iot-device-sdk-go/go.sum
RUN cd /go/src/github.com/seqsense/aws-iot-device-sdk-go/ && go mod download

COPY . /go/src/github.com/seqsense/aws-iot-device-sdk-go
WORKDIR /go/src/github.com/seqsense/aws-iot-device-sdk-go/tunnel/cmd/secure-tunnel-server
RUN go build -tags netgo -installsuffix netgo
RUN cp secure-tunnel-server /usr/local/bin/

# =====================
FROM alpine:3.12

COPY --from=go-builder /usr/local/bin/secure-tunnel-server /usr/local/bin/

CMD ["/usr/local/bin/secure-tunnel-server"]
