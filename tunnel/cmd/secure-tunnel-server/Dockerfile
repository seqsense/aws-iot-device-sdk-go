# =====================
FROM golang:1.18-alpine as go-builder

RUN apk add --no-cache git
ENV CGO_ENABLED=0

COPY go.mod go.sum /go/src/github.com/seqsense/aws-iot-device-sdk-go/
RUN cd /go/src/github.com/seqsense/aws-iot-device-sdk-go/ && go mod download

COPY . /go/src/github.com/seqsense/aws-iot-device-sdk-go
WORKDIR /go/src/github.com/seqsense/aws-iot-device-sdk-go/tunnel/cmd/secure-tunnel-server
RUN go build -tags netgo -installsuffix netgo -ldflags "-extldflags -static"
RUN cp secure-tunnel-server /usr/local/bin/

RUN apk add --no-cache ca-certificates

# =====================
FROM scratch

COPY --from=go-builder /usr/local/bin/secure-tunnel-server /usr/local/bin/
COPY --from=go-builder /etc/ssl /etc/ssl

CMD ["/usr/local/bin/secure-tunnel-server"]
