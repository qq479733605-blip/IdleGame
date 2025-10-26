module idlemmoserver/gate

go 1.22

require (
    github.com/gin-gonic/gin v1.11.0
    github.com/gorilla/websocket v1.5.3
    github.com/nats-io/nats.go v1.31.0
    idlemmoserver/common v0.0.0
)

replace idlemmoserver/common => ../common
