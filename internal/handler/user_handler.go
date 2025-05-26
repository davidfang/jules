// Package handler 包含了处理 HTTP 请求的 Handler (控制器)。
package handler

import (
	"errors"        // 用于检查特定错误类型，例如 repository.ErrNotFound
	"net/http"      // HTTP 状态码常量
	"strconv"       // 用于将字符串类型的 ID 转换为 uint
	"strings"       // 导入 strings 包，用于错误消息检查
	"unicode/utf8" // 用于验证路径参数的 UTF-8 编码

	"go-base-system-v2/internal/dto"       // 数据传输对象
	"go-base-system-v2/internal/repository" // Repository 层错误 (ErrNotFound)
	"go-base-system-v2/internal/service"    // Service 层接口
	"go-base-system-v2/pkg/response"       // 统一 API 响应包

	"github.com/gin-gonic/gin" // Gin 框架
	"go.uber.org/zap"          // Zap 日志库
)

// UserHandler 结构体封装了用户相关的业务逻辑服务和日志记录器。
// 它依赖于 UserService 来处理具体的用户操作。
type UserHandler struct {
	userSvc service.UserService // 用户服务接口实例
	logger  *zap.SugaredLogger  // Zap SugaredLogger 实例，用于日志记录
}

// NewUserHandler 是 UserHandler 的构造函数 Provider。
// 此函数用于 Wire 进行依赖注入，创建一个新的 UserHandler 实例。
// 参数:
//   userSvc: 实现了 service.UserService 接口的实例。如果为 nil，函数将 panic。
//   logger: Zap SugaredLogger 实例。如果为 nil，函数将 panic。
// 返回:
//   *UserHandler: 指向 UserHandler 实例的指针。
func NewUserHandler(userSvc service.UserService, logger *zap.SugaredLogger) *UserHandler {
	if userSvc == nil {
		panic("NewUserHandler: UserService instance is nil")
	}
	if logger == nil {
		panic("NewUserHandler: Zap SugaredLogger instance is nil")
	}
	return &UserHandler{userSvc: userSvc, logger: logger}
}

// RegisterUser godoc
// @Summary      用户注册 (Register User)
// @Description  根据提供的用户名、邮箱和密码创建一个新用户。
// @Tags         用户认证 (User Authentication) // 将注册和登录归为同一 Tag
// @Accept       json
// @Produce      json
// @Param        user_register_req  body      dto.UserRegisterReq  true  "用户注册请求体"
// @Success      201  {object}  response.ResponseData{data=model.User}  "用户创建成功"
// @Failure      400  {object}  response.ResponseData "请求参数错误"
// @Failure      500  {object}  response.ResponseData "服务器内部错误"
// @Router       /users/register [post]
func (h *UserHandler) RegisterUser(c *gin.Context) {
	var req dto.UserRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("用户注册请求参数绑定或校验失败", "error", err)
		response.FailBadRequest(c, fmt.Sprintf("请求参数错误: %v", err))
		return
	}

	h.logger.Infow("尝试注册新用户", "username", req.Username, "email", req.Email)
	user, err := h.userSvc.RegisterUser(c.Request.Context(), &req)
	if err != nil {
		h.logger.Errorw("用户注册服务层处理失败", "username", req.Username, "error", err)
		// 根据 Service 层返回的错误类型或内容判断具体错误
		if strings.Contains(err.Error(), "已被注册") { // 假设 Service 层返回此类错误信息
			response.FailBadRequest(c, err.Error())
		} else {
			response.FailInternalError(c, "用户注册失败，请稍后重试")
		}
		return
	}
	response.Success(c, http.StatusCreated, user)
}

// LoginUser godoc
// @Summary      用户登录 (User Login)
// @Description  使用用户名/邮箱和密码进行登录，成功后返回 JWT。
// @Tags         用户认证 (User Authentication)
// @Accept       json
// @Produce      json
// @Param        user_login_req  body      dto.UserLoginReq  true  "用户登录请求体"
// @Success      200  {object}  response.ResponseData{data=dto.UserLoginRes}  "登录成功，返回 JWT"
// @Failure      400  {object}  response.ResponseData "请求参数错误"
// @Failure      401  {object}  response.ResponseData "认证失败 (用户名或密码错误)"
// @Failure      500  {object}  response.ResponseData "服务器内部错误 (例如，Token 生成失败)"
// @Router       /users/login [post]
func (h *UserHandler) LoginUser(c *gin.Context) {
	var req dto.UserLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("用户登录请求参数绑定或校验失败", "error", err)
		response.FailBadRequest(c, fmt.Sprintf("请求参数错误: %v", err))
		return
	}

	h.logger.Infow("尝试用户登录", "usernameOrEmail", req.UsernameOrEmail)
	loginRes, err := h.userSvc.LoginUser(c.Request.Context(), &req)
	if err != nil {
		h.logger.Errorw("用户登录服务层处理失败", "usernameOrEmail", req.UsernameOrEmail, "error", err)
		if strings.Contains(err.Error(), "不存在或密码错误") {
			response.FailUnauthorized(c, "用户名或密码错误")
		} else if strings.Contains(err.Error(), "生成认证令牌失败") {
			response.FailInternalError(c, "登录失败，请稍后重试")
		} else {
			response.FailBadRequest(c, err.Error()) // 其他业务错误
		}
		return
	}
	response.SuccessOK(c, loginRes)
}

// GetUserByUsername godoc
// @Summary      根据用户名获取用户信息 (Get User By Username)
// @Description  通过指定的用户名检索用户的详细信息。
// @Tags         用户 (User)
// @Accept       json
// @Produce      json
// @Param        username  path      string  true  "用户名 (Username)" minlength(3) maxlength(50)
// @Success      200  {object}  response.ResponseData{data=model.User}  "成功获取用户信息"
// @Failure      400  {object}  response.ResponseData "请求参数错误"
// @Failure      404  {object}  response.ResponseData "用户未找到"
// @Failure      500  {object}  response.ResponseData "服务器内部错误"
// @Router       /users/username/{username} [get]
func (h *UserHandler) GetUserByUsername(c *gin.Context) {
	username := c.Param("username")
	if !utf8.ValidString(username) {
		h.logger.Warnw("无效的用户名参数 (非 UTF-8 编码)", "username_param", username)
		response.FailBadRequest(c, "用户名参数包含无效字符")
		return
	}
	h.logger.Infow("尝试通过用户名获取用户信息", "username", username)
	user, err := h.userSvc.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || strings.Contains(strings.ToLower(err.Error()), "未找到") {
			h.logger.Infow("通过用户名查询用户：未找到", "username", username)
			response.FailNotFound(c, fmt.Sprintf("用户 '%s' 未找到", username))
		} else {
			h.logger.Errorw("通过用户名查询用户失败", "username", username, "error", err)
			response.FailInternalError(c, "查询用户信息时发生内部错误")
		}
		return
	}
	response.SuccessOK(c, user)
}

// GetUserByID godoc
// @Summary      根据用户ID获取用户信息 (Get User By ID)
// @Description  通过指定的用户ID检索用户的详细信息。
// @Tags         用户 (User)
// @Accept       json
// @Produce      json
// @Param        id    path      int  true  "用户ID (User ID)" Format(uint)
// @Success      200  {object}  response.ResponseData{data=model.User}  "成功获取用户信息"
// @Failure      400  {object}  response.ResponseData "无效的用户ID"
// @Failure      404  {object}  response.ResponseData "用户未找到"
// @Failure      500  {object}  response.ResponseData "服务器内部错误"
// @Router       /users/id/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	idUint64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.Warnw("无效的用户ID参数 (无法解析为uint)", "id_param", idStr, "error", err)
		response.FailBadRequest(c, "用户ID参数格式无效，必须为正整数")
		return
	}
	id := uint(idUint64)
	h.logger.Infow("尝试通过ID获取用户信息", "userID", id)
	user, err := h.userSvc.GetUserByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || strings.Contains(strings.ToLower(err.Error()), "未找到") {
			h.logger.Infow("通过ID查询用户：未找到", "userID", id)
			response.FailNotFound(c, fmt.Sprintf("用户ID '%d' 未找到", id))
		} else {
			h.logger.Errorw("通过ID查询用户失败", "userID", id, "error", err)
			response.FailInternalError(c, "查询用户信息时发生内部错误")
		}
		return
	}
	response.SuccessOK(c, user)
}

// 导入 fmt (如果之前未导入)
import "fmt"

// GetCurrentUserProfile godoc
// @Summary      获取当前登录用户信息 (Get Current User Profile)
// @Description  获取当前通过 JWT 认证的用户的详细信息。
// @Tags         用户 (User)
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.ResponseData{data=model.User}  "成功获取当前用户信息"
// @Failure      401  {object}  response.ResponseData "未授权或 Token 无效"
// @Failure      404  {object}  response.ResponseData "用户未找到 (例如，Token 中的用户 ID 在数据库中不存在)"
// @Failure      500  {object}  response.ResponseData "服务器内部错误"
// @Router       /me/profile [get] // 路由路径，相对于 API Group 的基础路径 (例如 /api/v1/me/profile)
// GetCurrentUserProfile 处理获取当前认证用户信息的 HTTP GET 请求。
// 它依赖 JWTAuthMiddleware 将 userID 设置到 Gin Context 中。
func (h *UserHandler) GetCurrentUserProfile(c *gin.Context) {
	// 从 Gin Context 中获取由 JWTAuthMiddleware 设置的 userID。
	userIDAny, exists := c.Get(middleware.UserIDKey) // 使用 middleware 包中定义的常量
	if !exists {
		h.logger.Error("GetCurrentUserProfile: userID not found in context (middleware not run or failed?)")
		response.FailUnauthorized(c, "用户未认证 (无法获取用户ID)")
		return
	}

	// 将 userIDAny 断言为 uint 类型。
	userID, ok := userIDAny.(uint)
	if !ok || userID == 0 { // UserID 为 0 通常是无效的
		h.logger.Errorw("GetCurrentUserProfile: userID in context is not a valid uint or is zero", "userID_from_context", userIDAny)
		response.FailUnauthorized(c, "用户认证信息无效 (用户ID格式错误)")
		return
	}

	h.logger.Infow("Attempting to get current user profile", "userID", userID)

	// 调用 Service 层获取用户信息。
	user, err := h.userSvc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		// 检查是否是 Service 层返回的 "用户未找到" 错误。
		if strings.Contains(err.Error(), "未找到") { // 假设 Service 层返回的错误消息包含 "未找到"
			h.logger.Infow("Current user profile not found by ID from token", "userID", userID, "error", err)
			response.FailNotFound(c, fmt.Sprintf("无法找到ID为 '%d' 的用户 (可能账户已被删除)", userID))
		} else {
			h.logger.Errorw("Error getting current user profile from service", "userID", userID, "error", err)
			response.FailInternalError(c, "获取用户信息时发生内部错误")
		}
		return
	}

	// 用户信息获取成功。
	response.SuccessOK(c, user)
}

// 确保导入了 middleware 包
// import "go-base-system-v2/internal/middleware" // 已在顶部导入
