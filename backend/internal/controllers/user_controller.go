package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/middlewares"
	"backend/internal/models/domain"
	"backend/internal/models/web"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	Log          *logrus.Logger
	Service      *services.UserService
	SSOService   *services.SSOService
	LoginLimiter *middlewares.LoginLimiter
}

func NewUserController(service *services.UserService, ssoService *services.SSOService, log *logrus.Logger) *UserController {
	return &UserController{
		Log:          log,
		Service:      service,
		SSOService:   ssoService,
		LoginLimiter: middlewares.NewLoginLimiter(),
	}
}

func (c *UserController) Login(ctx *gin.Context) {
	// TODO: Validate the Request
	var request web.UserRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		ctx.Abort()
		return
	}

	clientIP := ctx.ClientIP()
	if wait := c.LoginLimiter.RetryAfter(clientIP, request.Username); wait > 0 {
		minutes := int(wait.Minutes()) + 1
		ctx.Header("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		ctx.JSON(http.StatusTooManyRequests, web.ErrorResponse{
			Code:   http.StatusTooManyRequests,
			Status: "Too Many Requests",
			Error:  fmt.Sprintf("Terlalu banyak percobaan login. Coba lagi dalam %d menit.", minutes),
		})
		ctx.Abort()
		return
	}

	// An unknown username and a wrong password get the same answer, so the login form
	// cannot be used to find out which usernames exist.
	rejectLogin := func() {
		c.LoginLimiter.Fail(clientIP, request.Username)
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Invalid username or password",
		})
		ctx.Abort()
	}

	// TODO: Find the user by username
	var user domain.User
	err := c.Service.UserReporsitory.FindByUsername(c.Service.DB, &user, request.Username)
	if err != nil {
		rejectLogin()
		return
	}

	// TODO: Login
	response, err := c.Service.Login(&user, request.Password)
	if err != nil {
		rejectLogin()
		return
	}
	c.LoginLimiter.Success(clientIP, request.Username)

	// TODO: Create Access Token
	tokenString, err := c.Service.CreateAccessToken(&user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  err.Error(),
		})
		ctx.Abort()
		return
	}

	// HTTP Only Cookie. A Secure cookie is dropped by the browser on plain HTTP, so it is
	// only marked Secure when this request arrived over HTTPS (directly or via a proxy);
	// tying it to ENV=production broke login on servers reached over http://.
	ctx.SetSameSite(http.SameSiteLaxMode) // Changed from None to Lax for same-site
	ctx.SetCookie(middlewares.TOKEN_COOKIE, tokenString, 3600*6, "/", "", requestIsHTTPS(ctx), true)

	// Send token in response body for frontend to store
	userData := response.Data.(web.UserData)
	loginResponse := web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data: web.UserLoginResponse{
			ID:       userData.ID,
			Username: userData.Username,
			Token:    tokenString,
		},
	}

	ctx.JSON(http.StatusOK, loginResponse)
}

func (c *UserController) Logout(ctx *gin.Context) {
	middlewares.DeleteToken(ctx)

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Status: "OK",
		Code:   http.StatusOK,
		Data:   "Logged out successfully",
	})
}

func (c *UserController) InitiateSSO(ctx *gin.Context) {
	if c.SSOService == nil || !c.SSOService.IsConfigured() {
		ctx.JSON(http.StatusServiceUnavailable, web.ErrorResponse{
			Code:   http.StatusServiceUnavailable,
			Status: "Service Unavailable",
			Error:  "SSO is not configured",
		})
		return
	}

	clientIDOverride := ctx.Query("client_id")
	redirectURL, err := c.SSOService.BuildInitiateRedirectURL(clientIDOverride)
	if err != nil {
		c.Log.Errorf("failed to build SSO redirect URL: %v", err)
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (c *UserController) SSOCallback(ctx *gin.Context) {
	if c.SSOService == nil || !c.SSOService.IsConfigured() {
		ctx.JSON(http.StatusServiceUnavailable, web.ErrorResponse{
			Code:   http.StatusServiceUnavailable,
			Status: "Service Unavailable",
			Error:  "SSO is not configured",
		})
		return
	}

	if ssoErr := ctx.Query("error"); ssoErr != "" {
		ctx.Redirect(http.StatusTemporaryRedirect, c.SSOService.BuildFrontendLoginErrorURL(ssoErr))
		return
	}

	code := ctx.Query("code")
	state := ctx.Query("state")

	localToken, err := c.SSOService.HandleCallback(code, state)
	if err != nil {
		c.Log.Errorf("SSO callback failed: %v", err)
		ctx.Redirect(http.StatusTemporaryRedirect, c.SSOService.BuildFrontendLoginErrorURL(err.Error()))
		return
	}

	ctx.Redirect(http.StatusTemporaryRedirect, c.SSOService.BuildFrontendCallbackURL(localToken))
}

// CreateUser membuat user baru (admin only)
func (c *UserController) CreateUser(ctx *gin.Context) {
	var request web.UserCreateRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	user, err := c.Service.CreateUser(&request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, web.SuccessResponse{
		Code:   http.StatusCreated,
		Status: "User created successfully",
		Data:   user,
	})
}

// GetAllUsers mengambil semua user (admin only)
func (c *UserController) GetAllUsers(ctx *gin.Context) {
	users, err := c.Service.GetAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   users,
	})
}

// GetUserByID mengambil user berdasarkan ID (admin only)
func (c *UserController) GetUserByID(ctx *gin.Context) {
	userID := ctx.Param("id")

	var id int
	if _, err := fmt.Sscanf(userID, "%d", &id); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Invalid user ID",
		})
		return
	}

	user, err := c.Service.GetUserByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, web.ErrorResponse{
			Code:   http.StatusNotFound,
			Status: "Not Found",
			Error:  "User not found",
		})
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "OK",
		Data:   user,
	})
}

// UpdateUser update user (admin only)
func (c *UserController) UpdateUser(ctx *gin.Context) {
	userID := ctx.Param("id")

	var id int
	if _, err := fmt.Sscanf(userID, "%d", &id); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Invalid user ID",
		})
		return
	}

	var request web.UserUpdateRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	user, err := c.Service.UpdateUser(id, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "User updated successfully",
		Data:   user,
	})
}

// DeleteUser hapus user (admin only)
func (c *UserController) DeleteUser(ctx *gin.Context) {
	userID := ctx.Param("id")

	var id int
	if _, err := fmt.Sscanf(userID, "%d", &id); err != nil {
		ctx.JSON(http.StatusBadRequest, web.ErrorResponse{
			Code:   http.StatusBadRequest,
			Status: "Bad Request",
			Error:  "Invalid user ID",
		})
		return
	}

	if err := c.Service.DeleteUser(id); err != nil {
		ctx.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Code:   http.StatusInternalServerError,
			Status: "Internal Server Error",
			Error:  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, web.SuccessResponse{
		Code:   http.StatusOK,
		Status: "User deleted successfully",
		Data:   nil,
	})
}

// requestIsHTTPS reports whether the client reached the API over HTTPS, either directly or
// through a reverse proxy that terminates TLS.
func requestIsHTTPS(ctx *gin.Context) bool {
	if ctx.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(ctx.GetHeader("X-Forwarded-Proto")), "https")
}
