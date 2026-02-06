package users

import (
	"context"
	"strings"
	"time"

	"github.com/baobei23/goapp/internal/pkg/logger"
	"github.com/naughtygopher/errors"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserEmailNotFound      = errors.New("user with the email not found")
	ErrUserEmailAlreadyExists = errors.New("user with the email already exists")
	QueryTimeoutDuration      = 5 * time.Second
)

type User struct {
	ID             string `json:"id"`
	FullName       string `json:"fullName"`
	Email          string `json:"email"`
	Password       []byte `json:"-"`
	Phone          string `json:"phone"`
	ContactAddress string `json:"contactAddress"`
}

// ValidateForCreate runs the validation required for when a user is being created. i.e. ID is not available
func (us *User) ValidateForCreate() error {
	if us.FullName == "" {
		return errors.Validation("full name cannot be empty")
	}

	if us.Email == "" {
		return errors.Validation("email cannot be empty")
	}

	if len(us.Password) == 0 {
		return errors.Validation("password cannot be empty")
	}

	return nil
}

func (us *User) Sanitize() {
	us.ID = strings.TrimSpace(us.ID)
	us.FullName = strings.TrimSpace(us.FullName)
	us.Email = strings.TrimSpace(us.Email)
	us.Phone = strings.TrimSpace(us.Phone)
	us.ContactAddress = strings.TrimSpace(us.ContactAddress)
}

func (us *User) HashPassword() error {
	hashed, err := bcrypt.GenerateFromPassword(us.Password, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	us.Password = hashed
	return nil
}

func (us *User) CheckPassword(plain string) bool {
	err := bcrypt.CompareHashAndPassword(us.Password, []byte(plain))
	return err == nil
}

type store interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	SaveUser(ctx context.Context, user *User) (string, error)
	BulkSaveUser(ctx context.Context, users []User) error
}
type Users struct {
	store store
}

func (us *Users) Register(ctx context.Context, user *User) (*User, error) {
	user.Sanitize()
	err := user.ValidateForCreate()
	if err != nil {
		return nil, err
	}

	if err := user.HashPassword(); err != nil {
		return nil, errors.Wrap(err, "failed to hash password")
	}

	newID, err := us.store.SaveUser(ctx, user)
	if err != nil {
		return nil, err
	}
	user.ID = newID

	return user, nil
}

func (us *Users) ReadByEmail(ctx context.Context, email string) (*User, error) {
	if email == "" {
		return nil, errors.Validation("no email provided")
	}

	return us.store.GetUserByEmail(ctx, email)
}

func (us *Users) AsyncRegisters(ctx context.Context, users []User) error {
	errList := make([]error, 0, len(users))
	for i := range users {
		err := users[i].ValidateForCreate()
		if err != nil {
			errList = append(errList, err)
			continue
		}

		if err := users[i].HashPassword(); err != nil {
			errList = append(errList, err)
		}
	}

	if len(errList) != 0 {
		return errors.Join(errList...)
	}

	go func() {
		ctx := context.TODO()
		err := us.store.BulkSaveUser(context.TODO(), users)
		if err != nil {
			logger.Error(ctx, err, users)
		}
	}()

	return nil
}

func (us *Users) Login(ctx context.Context, email, password string) (*User, error) {
	user, err := us.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if !user.CheckPassword(password) {
		return nil, errors.New("invalid credentials")
	}
	return user, nil
}

func NewService(store store) *Users {
	return &Users{
		store: store,
	}
}
