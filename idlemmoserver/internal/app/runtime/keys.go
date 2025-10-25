package runtime

const (
	ServiceActorSystem    ServiceKey = "actor:system"
	ServiceActorRoot      ServiceKey = "actor:root"
	ServiceGatewayPID     ServiceKey = "actor:gateway_pid"
	ServicePersistPID     ServiceKey = "actor:persist_pid"
	ServiceSchedulerPID   ServiceKey = "actor:scheduler_pid"
	ServiceGameRepository ServiceKey = "persist:game_repo"

	ServiceGinEngine   ServiceKey = "http:gin_engine"
	ServiceHTTPServer  ServiceKey = "http:server"
	ServiceUserHandler ServiceKey = "http:user_handler"
	ServiceUserRepo    ServiceKey = "user:repo"
	ServiceUserService ServiceKey = "user:registration_service"
)
