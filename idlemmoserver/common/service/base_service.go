package service

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Service 服务接口 - 所有服务都应该实现这个接口
type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetName() string
}

// BaseServiceImpl 服务基类实现 - 提供通用的服务启动/关闭逻辑
type BaseServiceImpl struct {
	name string
}

// NewBaseService 创建基础服务实例
func NewBaseService(name string) *BaseServiceImpl {
	return &BaseServiceImpl{
		name: name,
	}
}

// GetName 获取服务名称
func (s *BaseServiceImpl) GetName() string {
	return s.name
}

// Start 启动服务 - 基类实现，子类可以重写
func (s *BaseServiceImpl) Start(ctx context.Context) error {
	log.Printf("Starting %s service...", s.name)
	return nil
}

// Stop 停止服务 - 基类实现，子类可以重写
func (s *BaseServiceImpl) Stop(ctx context.Context) error {
	log.Printf("Stopping %s service...", s.name)
	return nil
}

// RunService 运行服务的通用模式 - 消除所有 main.go 中的重复代码
func RunService(service Service) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动服务
	log.Printf("Starting %s service...", service.GetName())
	if err := service.Start(ctx); err != nil {
		log.Fatalf("Failed to start %s service: %v", service.GetName(), err)
	}
	log.Printf("%s service started successfully", service.GetName())

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down %s service...", sig, service.GetName())

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := service.Stop(shutdownCtx); err != nil {
		log.Printf("Error during %s service shutdown: %v", service.GetName(), err)
	} else {
		log.Printf("%s service stopped successfully", service.GetName())
	}
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name            string
	NATSURL         string
	ShutdownTimeout time.Duration
}

// DefaultServiceConfig 默认服务配置
func DefaultServiceConfig(name string) *ServiceConfig {
	return &ServiceConfig{
		Name:            name,
		NATSURL:         "nats://localhost:4222",
		ShutdownTimeout: 10 * time.Second,
	}
}
