package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/domain/user"
	"github.com/radiophysiker/d56/internal/infrastructure/http/middleware"
	"github.com/radiophysiker/d56/internal/infrastructure/jwt"
	"github.com/radiophysiker/d56/internal/service"
)

type UserHandler struct {
	userService       *service.UserService
	orderService      *service.OrderService
	withdrawalService *service.WithdrawalService
	jwtService        *jwt.Service
}

func NewUserHandler(
	userService *service.UserService,
	orderService *service.OrderService,
	withdrawalService *service.WithdrawalService,
	jwtService *jwt.Service,
) *UserHandler {
	return &UserHandler{
		userService:       userService,
		orderService:      orderService,
		withdrawalService: withdrawalService,
		jwtService:        jwtService,
	}
}

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type OrderResponse struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    *float64  `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type WithdrawalRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type WithdrawalResponse struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

/*
Register handles user registration
*/
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	newUser, err := h.userService.Register(r.Context(), req.Login, req.Password)

	if err != nil {

		if validationErr, ok := err.(user.ValidationErrors); ok {
			http.Error(w, validationErr.Error(), http.StatusBadRequest)
			return
		}

		if errors.Is(err, user.ErrLoginAlreadyExists) {
			http.Error(w, "Login already taken", http.StatusConflict)
			return
		}
		zap.L().Error("Failed to register user", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	token, err := h.jwtService.GenerateToken(newUser.ID())
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

/*
Login handles user login
This function verifies the user's credentials and returns a JWT token if successful.
*/
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Login == "" || req.Password == "" {
		http.Error(w, "Login and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.userService.Authenticate(r.Context(), req.Login, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.jwtService.GenerateToken(user.ID())
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

// UploadOrder handles the upload of a new order
func (h *UserHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	UserID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" {
		http.Error(w, "Order number is required", http.StatusBadRequest)
		return
	}

	isNew, err := h.orderService.CreateOrder(r.Context(), UserID, orderNumber)
	if err != nil {
		if errors.Is(err, order.ErrOrderNumberEmpty) {
			http.Error(w, "Order number cannot be empty", http.StatusBadRequest)
			return
		}
		if errors.Is(err, order.ErrInvalidFormat) {
			http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, order.ErrOrderAlreadyTaken) {
			http.Error(w, "Order already uploaded by another user", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if isNew {
		w.WriteHeader(http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (h *UserHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	UserID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.orderService.GetUserOrders(r.Context(), UserID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var response []OrderResponse
	for _, order := range orders {
		response = append(response, OrderResponse{
			Number:     order.Number(),
			Status:     string(order.Status()),
			Accrual:    order.Accrual(),
			UploadedAt: order.UploadedAt(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *UserHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	UserID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	balance, err := h.userService.GetBalance(r.Context(), UserID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(BalanceResponse{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	})
}

func (h *UserHandler) WithdrawFunds(w http.ResponseWriter, r *http.Request) {
	UserID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req WithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Order == "" || req.Sum <= 0 {
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	_, err := h.withdrawalService.WithdrawFunds(r.Context(), UserID, req.Order, req.Sum)
	if err != nil {
		if errors.Is(err, order.ErrInvalidFormat) {
			http.Error(w, "Invalid order number format", http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, order.ErrInsufficientFunds) {
			http.Error(w, "Insufficient funds", http.StatusPaymentRequired)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	UserID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.withdrawalService.GetUserWithdrawals(r.Context(), UserID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var response []WithdrawalResponse
	for _, withdrawal := range withdrawals {
		response = append(response, WithdrawalResponse{
			Order:       withdrawal.OrderNumber(),
			Sum:         withdrawal.Amount(),
			ProcessedAt: withdrawal.ProcessedAt(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
