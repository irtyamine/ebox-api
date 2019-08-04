package users

import (
	"ebox-api/internal/db"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"regexp"
)

const (
	minPasswordLength = 8
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidEmail = errors.New("invalid email address")
	ErrPasswordContainsInvalidChars = errors.New("password contains invalid characters")
	ErrPasswordTooShort = errors.New("password is too short")
	ErrWrongCredentials = errors.New("wrong credentials")

	emailRegExp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	// regex for valid password chars - we are using character classes https://github.com/google/re2/wiki/Syntax#ascii
	validPasswordCharsRegExp = regexp.MustCompile("^[[:print:]]+$")
)

type UsersRepository interface {
	CreateUser(reqData PostUserRequestData) (*User, error)
	GetUserById(userID int) (*User, error)
	ValidateUser(email string, password string) (int, error)
}

type usersRepository struct {
	db *db.DB
}

func NewUsersRepository(db *db.DB) UsersRepository {
	return &usersRepository{
		db: db,
	}
}

func (r *usersRepository) CreateUser(reqData PostUserRequestData) (*User, error) {
	password, err := hashPassword(reqData.Password)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO ebox.users (email, password, firstName, lastName, avatarUrl)
		VALUES  ($1, $2, $3, $4, $5)
		RETURNING id, email, firstName, lastName, avatarUrl
	`

	user := new(User)
	err = r.db.QueryRow(query, reqData.Email, password, reqData.FirstName, reqData.LastName, reqData.AvatarURL).
		Scan(&user.Id, &user.Email, &user.FirstName, &user.LastName, &user.AvatarURL)

	if err != nil {
		dbErr := db.GetError(err)

		if dbErr == db.ErrUniqueConstraintViolation {
			err = ErrUserAlreadyExists
		} else {
			err = dbErr
		}

		return nil, err
	}

	return user, nil
}

func (r *usersRepository) GetUserById(userID int) (*User, error) {
	query := `SELECT id, email, firstName, lastName, avatarUrl FROM ebox.users WHERE id = $1 LIMIT 1`
	user := new(User)
	err := r.db.QueryRow(query, userID).Scan(&user.Id, &user.Email, &user.FirstName, &user.LastName, &user.AvatarURL)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *usersRepository) ValidateUser(email string, password string) (int, error) {
	query := `SELECT id, password FROM ebox.users WHERE email = $1 LIMIT 1`
	var currentPassword string
	var userID int
	err := r.db.QueryRow(query, email).Scan(&userID, &currentPassword)
	if err != nil {
		return 0, ErrWrongCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(currentPassword), []byte(password))
	if err != nil {
		return 0, ErrWrongCredentials
	}

	return userID, nil
}

func validatePassword(password string) error {
	isValid := validPasswordCharsRegExp.MatchString(password)

	if !isValid {
		return ErrPasswordContainsInvalidChars
	}

	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}

	return nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}
