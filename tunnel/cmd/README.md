# Secure Tunneling commands

## secure-tunnel-server

`secure-tunnel-server` provides similar functionality of AWS IoT Secure Tunneling service.
Note that it doesn't have access control of the API.
It should be used in the closed network.

### Demo

1. Build the image
    ```shell
    $ cd secure-tunnel-server
    $ docker-compose build
    ```
2. Uncomment `-generate-test-token` option in [secure-tunnel-server/docker-compose.yml](secure-tunnel-server/docker-compose.yml)
3. Run the container. Example token pair will be shown
    ```shell
    $ docker-compose up
    secure-tunnel-server_1  | 2000/00/00 00:00:00 {
      "destinationAccessToken": "${DESTINATION_ACCESS_TOKEN}",
      "sourceAccessToken": "${SOURCE_ACCESS_TOKEN}",
      "tunnelArn": "arn:clone:iotsecuretunneling:::00000000",
      "tunnelId": "00000000"
    }
    ```

## localproxy

`localproxy` is a Go implementation of [aws-samples/aws-iot-securetunneling-localproxy](https://github.com/aws-samples/aws-iot-securetunneling-localproxy).

### Demo

Following commands proxies connection to your local ssh server via `secure-tunnel-server`.

1. Build the localproxy
    ```shell
    cd localproxy
    ```
2. Run the destination proxy
    ```shell
    $ ./localproxy -access-token=${DESTINATION_ACCESS_TOKEN} \
        -destination-app=localhost:22 \
        -proxy-endpoint=localhost:443 \
        -no-ssl-host-verify=true
    ```
3. Run the sourceproxy
    ```shell
    $ ./localproxy -access-token=${SOURCE_ACCESS_TOKEN} \
        -source-listen-port=2222 \
        -proxy-endpoint=localhost:443 \
        -no-ssl-host-verify
    ```
4. Connect to the server via proxy
    ```shell
    $ ssh localhost -p 2222
    ```
