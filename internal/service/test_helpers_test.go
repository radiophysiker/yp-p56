package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/radiophysiker/d56/internal/domain/order"
	user "github.com/radiophysiker/d56/internal/domain/user"
)

func randomUserID() user.UserID { return uuid.New() }

const testOrderNumber = "79927398713"

func matchContext(expected context.Context) func(context.Context) bool {
	return func(c context.Context) bool { return c == expected }
}

func matchOrder(expectedUserID user.UserID, number string, status order.Status) func(*order.Order) bool {
	return func(o *order.Order) bool {
		return o.UserID() == expectedUserID && o.Number() == number && o.Status() == status
	}
}
