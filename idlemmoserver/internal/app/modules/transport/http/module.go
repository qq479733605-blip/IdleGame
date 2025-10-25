package httptransport

import (
	"context"
	"errors"
	"net/http"
	"time"

	"idlemmoserver/internal/app/runtime"
	"idlemmoserver/internal/gateway"
	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Module wires the Gin HTTP server and exposes public APIs.
type Module struct {
	addr   string
	server *http.Server
}

// New creates an HTTP transport module bound to addr.
func New(addr string) *Module {
	return &Module{addr: addr}
}

func (m *Module) Name() string { return "transport.http" }

func (m *Module) Configure(ctx context.Context, c *runtime.Container) error {
	engine := gin.Default()
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	c.MustProvide(runtime.ServiceGinEngine, engine)
	return nil
}

func (m *Module) Start(ctx context.Context, c *runtime.Container) error {
	engine, err := runtime.Resolve[*gin.Engine](c, runtime.ServiceGinEngine)
	if err != nil {
		return err
	}
	root, err := runtime.Resolve[*actor.RootContext](c, runtime.ServiceActorRoot)
	if err != nil {
		return err
	}
	gatewayPID, err := runtime.Resolve[*actor.PID](c, runtime.ServiceGatewayPID)
	if err != nil {
		return err
	}
	userHandler, err := runtime.Resolve[*gateway.UserHandler](c, runtime.ServiceUserHandler)
	if err != nil {
		return err
	}

	gateway.InitRoutes(engine, root, gatewayPID, userHandler)

	srv := &http.Server{Addr: m.addr, Handler: engine}
	c.MustProvide(runtime.ServiceHTTPServer, srv)
	m.server = srv

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logx.Error("http server failure", "err", err)
		}
	}()

	logx.Info("http server started", "addr", m.addr)
	return nil
}

func (m *Module) Stop(ctx context.Context, c *runtime.Container) error {
	if m.server == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := m.server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		return err
	}

	logx.Info("http server stopped", "addr", m.addr)
	return nil
}
