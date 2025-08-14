package order

import "strconv"

// IsValidOrderNumber checks the order number using the Moon algorithm
func IsValidOrderNumber(number string) bool {
	if len(number) == 0 {
		return false
	}

	for _, char := range number {
		if char < '0' || char > '9' {
			return false
		}
	}

	sum := 0
	alternate := false
	for i := len(number) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(number[i]))

		if alternate {
			digit *= 2
			if digit > 9 {
				digit = digit%10 + digit/10
			}
		}

		sum += digit
		alternate = !alternate
	}

	return sum%10 == 0
}
