FROM nginx:1.23-alpine

RUN apk add --no-cache openssl

RUN openssl req -batch -new -x509 -newkey rsa:2048 -nodes -sha256 \
  -subj /CN=example.com/O=selfsigned -days 3650 \
  -keyout /etc/nginx/server.key -out /etc/nginx/server.crt

COPY etc /etc

COPY nginx_entrypoint.sh /nginx_entrypoint.sh
ENTRYPOINT ["/nginx_entrypoint.sh"]
CMD []
