package password

import "golang.org/x/crypto/bcrypt"

// BcryptService implements the PasswordService interface using bcrypt for hashing
type BcryptService struct{}

// NewBcryptService creates a new instance of BcryptService
func NewBcryptService() *BcryptService {
	return &BcryptService{}
}

// HashPassword hashes the given password using bcrypt
func (s *BcryptService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword checks if the provided password matches the hashed password
func (s *BcryptService) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
