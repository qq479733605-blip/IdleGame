module idlemmoserver/game

go 1.22

require (
    github.com/asynkron/protoactor-go/actor v1.1.0
    github.com/nats-io/nats.go v1.31.0
    idlemmoserver/common v0.0.0
)

replace idlemmoserver/common => ../common
