package services

import (
	"fmt"
	"net/http"
	"time"

	"backend/internal/models/converter"
	"backend/internal/models/domain"
	"backend/internal/models/web"
	"backend/internal/repositories"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	DB              *gorm.DB
	Log             *logrus.Logger
	Validate        *validator.Validate
	Config          *viper.Viper
	UserReporsitory *repositories.UserReporsitory
}

func NewUserService(db *gorm.DB, log *logrus.Logger, validate *validator.Validate, config *viper.Viper, userRepository *repositories.UserReporsitory) *UserService {
	return &UserService{
		DB:              db,
		Log:             log,
		Validate:        validate,
		Config:          config,
		UserReporsitory: userRepository,
	}
}

func (service *UserService) CreateAccessToken(user *domain.User) (string, error) {
	// TODO: Generate JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.ID,
		"username": user.Username,
		"expired":  time.Now().Add(time.Hour * 6).Unix(), // 6 Hours
	})

	// TODO: Sign and Get the complete encoded token as a string using the secret key
	secretKey := service.Config.GetString("JWT_SECRET_KEY")

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (service *UserService) Login(user *domain.User, password string) (*web.SuccessResponse, error) {
	// TODO: Compare the password
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	// TODO: Convert domain into data
	userData := converter.ToUserData(user)
	response := &web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   userData,
	}

	return response, nil
}

func (service *UserService) Logout(ctx *gin.Context, token string) {
}

// CreateUser membuat user baru dengan hashing password
func (service *UserService) CreateUser(req *web.UserCreateRequest) (*web.UserListResponse, error) {
	// Validasi request
	if err := service.Validate.Struct(req); err != nil {
		return nil, err
	}

	// Check apakah username sudah ada
	var existingUser domain.User
	err := service.UserReporsitory.FindByUsername(service.DB, &existingUser, req.Username)
	if err == nil {
		// User sudah ada
		return nil, fmt.Errorf("username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Buat user baru
	user := &domain.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Role:     req.Role,
	}

	// Save ke database
	if err := service.UserReporsitory.Create(service.DB, user); err != nil {
		return nil, err
	}

	// Convert dan return response
	response := converter.ToUserListResponse(user)
	return &response, nil
}

// GetAllUsers mengambil semua user
func (service *UserService) GetAllUsers() ([]web.UserListResponse, error) {
	users, err := service.UserReporsitory.FindAll(service.DB)
	if err != nil {
		return nil, err
	}

	var responses []web.UserListResponse
	for _, user := range users {
		responses = append(responses, converter.ToUserListResponse(&user))
	}

	return responses, nil
}

// GetUserByID mengambil user berdasarkan ID
func (service *UserService) GetUserByID(id int) (*web.UserListResponse, error) {
	user, err := service.UserReporsitory.FindByID(service.DB, id)
	if err != nil {
		return nil, err
	}

	response := converter.ToUserListResponse(user)
	return &response, nil
}

// UpdateUser update user
func (service *UserService) UpdateUser(id int, req *web.UserUpdateRequest) (*web.UserListResponse, error) {
	// Ambil user dari database
	user, err := service.UserReporsitory.FindByID(service.DB, id)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Update username jika diberikan
	if req.Username != "" {
		// Check apakah username baru sudah ada (dan bukan milik user yang sama)
		var existingUser domain.User
		err := service.UserReporsitory.FindByUsername(service.DB, &existingUser, req.Username)
		if err == nil && existingUser.ID != user.ID {
			// Username sudah digunakan user lain
			return nil, fmt.Errorf("username already in use")
		}
		user.Username = req.Username
	}

	// Update password jika diberikan
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hashedPassword)
	}

	// Update role jika diberikan
	if req.Role != "" {
		user.Role = req.Role
	}

	// Save perubahan
	if err := service.UserReporsitory.Update(service.DB, user); err != nil {
		return nil, err
	}

	// Re-fetch user to get updated data with timestamps
	updatedUser, err := service.UserReporsitory.FindByID(service.DB, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated user")
	}

	response := converter.ToUserListResponse(updatedUser)
	return &response, nil
}

// DeleteUser hapus user
func (service *UserService) DeleteUser(id int) error {
	return service.UserReporsitory.Delete(service.DB, id)
}
