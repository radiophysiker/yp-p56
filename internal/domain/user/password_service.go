package user

type PasswordService interface {
	HashPassword(password string) (string, error)
	CheckPassword(hashedPassword, password string) bool
}
