package router

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/radiophysiker/d56/internal/infrastructure/http/handler"
	authmiddleware "github.com/radiophysiker/d56/internal/infrastructure/http/middleware"
)

// Router is the HTTP router for the application
type Router struct {
	userHandler    *handler.UserHandler
	authMiddleware *authmiddleware.AuthMiddleware
	logger         *zap.Logger
}

// NewRouter creates a new Router instance
func NewRouter(
	userHandler *handler.UserHandler,
	authMiddleware *authmiddleware.AuthMiddleware,
	logger *zap.Logger,
) *Router {
	return &Router{
		userHandler:    userHandler,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// Setup sets up the HTTP routes and middleware
func (rt *Router) Setup() *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	rt.setupGlobalMiddleware(r)

	// Routes
	rt.setupPublicRoutes(r)
	rt.setupProtectedRoutes(r)

	return r
}

func (rt *Router) setupGlobalMiddleware(r *chi.Mux) {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.Compress(5))
}

func (rt *Router) setupPublicRoutes(r *chi.Mux) {
	r.Post("/api/user/register", rt.userHandler.Register)
	r.Post("/api/user/login", rt.userHandler.Login)
}

func (rt *Router) setupProtectedRoutes(r *chi.Mux) {
	r.Route("/api/user", func(r chi.Router) {
		r.Use(rt.authMiddleware.RequireAuth)
		r.Post("/orders", rt.userHandler.UploadOrder)
		r.Get("/orders", rt.userHandler.GetOrders)
		r.Get("/balance", rt.userHandler.GetBalance)
		r.Post("/balance/withdraw", rt.userHandler.WithdrawFunds)
		r.Get("/withdrawals", rt.userHandler.GetWithdrawals)
	})
}
