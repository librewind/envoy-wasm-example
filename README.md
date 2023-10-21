# envoy-wasm-example

### Go useful commands:
```
go mod init main.go
go mod tidy
tinygo build -o sip_uri_plugin.wasm -scheduler=none -target=wasi ./main.go
```

### Test request:
```
curl --location 'http://localhost:8080/invite' \
--header 'X-Sip-Uri: sip:1@dev-sip.webinar.ru;kamailio=test' \
--header 'Content-Type: application/json' \
--header 'Cookie: sessionId=8f460feabd589bb867634b99582f7c50' \
--data-raw '{
    "callId": "1",
    "uri": "sip:1@dev-sip.local;kamailio=test"
}'
```
