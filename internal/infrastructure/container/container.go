package container

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/radiophysiker/d56/internal/config"
	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/domain/user"
	"github.com/radiophysiker/d56/internal/domain/withdrawal"
	"github.com/radiophysiker/d56/internal/infrastructure/accrual"
	"github.com/radiophysiker/d56/internal/infrastructure/database"
	"github.com/radiophysiker/d56/internal/infrastructure/http/handler"
	"github.com/radiophysiker/d56/internal/infrastructure/http/middleware"
	"github.com/radiophysiker/d56/internal/infrastructure/http/router"
	"github.com/radiophysiker/d56/internal/infrastructure/jwt"
	"github.com/radiophysiker/d56/internal/infrastructure/password"
	"github.com/radiophysiker/d56/internal/infrastructure/repository/postgres"
	"github.com/radiophysiker/d56/internal/service"
)

// Container contains all the dependencies of the application
type Container struct {
	// Infrastructure components
	dbConnection    *database.DatabaseConnection
	passwordService user.PasswordService
	jwtService      *jwt.Service
	accrualClient   *accrual.Client
	authMiddleware  *middleware.AuthMiddleware
	httpRouter      *router.Router

	// Repositories (Infrastructure Layer)
	userRepo       user.Repository
	orderRepo      order.Repository
	withdrawalRepo withdrawal.Repository

	// Services (Application Layer)
	userService             *service.UserService
	orderService            *service.OrderService
	withdrawalService       *service.WithdrawalService
	accrualProcessorService *service.AccrualProcessorService

	// Handlers (Presentation Layer)
	userHandler *handler.UserHandler
}

// New creates a new dependency injection container
func New(cfg *config.Config, logger *zap.Logger) (*Container, error) {
	container := &Container{}

	if err := container.initInfrastructure(cfg, logger); err != nil {
		return nil, err
	}

	// Initialize Repositories (Infrastructure Layer)
	container.initRepositories()

	// Initialize Services (Application Layer)
	container.initServices(logger)

	// Initialize Handlers (Presentation Layer)
	container.initHandlers()

	// Initialize Http-router
	container.initRouter(logger)

	return container, nil
}

func (c *Container) GetRouter() *router.Router {
	return c.httpRouter
}

// Close closes all container resources
func (c *Container) Close() error {
	if c.dbConnection != nil {
		return c.dbConnection.Close()
	}
	return nil
}

// initInfrastructure initializes the infrastructure layer
func (c *Container) initInfrastructure(cfg *config.Config, logger *zap.Logger) error {
	dbConn, err := database.NewPostgreSQLConnection(cfg.DatabaseURI)
	if err != nil {
		return err
	}
	c.dbConnection = dbConn

	c.passwordService = password.NewBcryptService()

	jwtService, err := jwt.NewService(cfg.JWTSecretKey)
	if err != nil {
		return fmt.Errorf("failed to initialize JWT service: %w", err)
	}
	c.jwtService = jwtService

	if cfg.AccrualSystemAddress != "" {
		c.accrualClient = accrual.NewClient(cfg.AccrualSystemAddress)
	}

	c.authMiddleware = middleware.NewAuthMiddleware(c.jwtService)

	return nil
}

// initRepositories initializes the repositories
func (c *Container) initRepositories() {
	db := c.dbConnection.GetDB()

	c.userRepo = postgres.NewUserRepository(db)
	c.orderRepo = postgres.NewOrderRepository(db)
	c.withdrawalRepo = postgres.NewWithdrawalRepository(db)
}

// initServices initializes the services
func (c *Container) initServices(logger *zap.Logger) {
	c.userService = service.NewUserService(c.userRepo, c.passwordService)
	c.orderService = service.NewOrderService(c.orderRepo, c.userRepo)
	c.withdrawalService = service.NewWithdrawalService(c.withdrawalRepo, c.userRepo)

	// Initialize AccrualProcessorService if accrual client is available
	if c.accrualClient != nil {
		c.accrualProcessorService = service.NewAccrualProcessorService(c.accrualClient, c.orderService, logger)
	}
}

// initHandlers инициализирует хендлеры
func (c *Container) initHandlers() {
	c.userHandler = handler.NewUserHandler(
		c.userService,
		c.orderService,
		c.withdrawalService,
		c.jwtService,
	)
}

// initRouter инициализирует роутер
func (c *Container) initRouter(logger *zap.Logger) {
	c.httpRouter = router.NewRouter(c.userHandler, c.authMiddleware, logger)
}

func (c *Container) GetAccrualProcessorService() *service.AccrualProcessorService {
	return c.accrualProcessorService
}
