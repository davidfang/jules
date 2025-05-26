// Package service_test 包含了对 service 层业务逻辑的单元测试。
package service_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time" // 用于JWT过期时间

	"go-base-system-v2/internal/dto"
	"go-base-system-v2/internal/model"
	"go-base-system-v2/internal/repository"
	mock_repository "go-base-system-v2/internal/repository/mock" // Mock Repository
	"go-base-system-v2/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/golang/mock/gomock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// newTestUserServiceWithMocks 创建一个包含 mock 依赖的 userServiceImpl 实例用于测试。
// 返回 userService, mockUserRepo, 一个基础的 Casbin Enforcer 实例, 和 gomock Controller。
func newTestUserServiceWithMocks(t *testing.T) (
	service.UserService,
	*mock_repository.MockUserRepository,
	*casbin.Enforcer,
	*gomock.Controller,
) {
	ctrl := gomock.NewController(t)
	mockUserRepo := mock_repository.NewMockUserRepository(ctrl)

	// 创建一个临时的 Zap logger 用于测试，避免实际日志输出干扰测试结果。
	// 可以使用 zap.NewNop().Sugar() 来完全禁止日志输出。
	// tempLogger, _ := zap.NewDevelopment() // 或者使用 zap.NewExample()
	tempLogger := zap.NewNop() // 在测试中通常不需要看到日志输出
	sugaredLogger := tempLogger.Sugar()

	// 创建一个临时的 Viper 实例并设置必要的 JWT 配置。
	tempViper := viper.New()
	tempViper.Set("jwt.secret_key", "test_very_secret_key_for_unit_test_longer_than_32_bytes")
	tempViper.Set("jwt.issuer", "test_issuer_for_unit_test")
	tempViper.Set("jwt.access_token_expire_duration", "1m") // 短过期时间便于测试

	// 为 Casbin Enforcer 创建一个内存适配器和模型。
	// 注意模型文件的路径。测试通常在包目录内执行。
	// 假设项目根目录是 GOPATH/src/go-base-system-v2 或类似的结构。
	// 或者，我们可以动态地确定项目根路径。
	// 简便起见，我们假设测试执行时能够找到相对路径的配置文件。
	// 找到项目根目录
	// _, b, _, _ := runtime.Caller(0)
	// projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(b))) // 获取项目根目录，可能需要调整层数
	// modelPath := filepath.Join(projectRoot, "config", "casbin_model.conf")

	// 更健壮的方式是确保测试文件知道配置文件的相对位置。
	// 如果测试在 internal/service 包下执行，那么 config 目录的相对路径是 ../../config
	modelPath := "../../config/casbin_model.conf"
	// 检查文件是否存在，如果不存在则尝试其他可能的相对路径或跳过Enforcer的初始化（如果测试不依赖它）
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		// 尝试另一种常见的项目结构下的路径，例如从项目根目录执行测试
		altModelPath := filepath.Join("..", "..", "config", "casbin_model.conf") // 退两级到项目根，再进入config
		if _, errAlt := os.Stat(altModelPath); errAlt == nil {
			modelPath = altModelPath
		} else {
			// 如果都找不到，可以尝试更灵活的方式或直接失败
			t.Logf("Casbin model file not found at primary path: %s or alt path: %s. Attempting default.", modelPath, altModelPath)
			// 尝试从一个更通用的位置（相对于可能的执行路径）
			modelPath = "config/casbin_model.conf" // 如果测试是从项目根目录执行
			if _, errFinal := os.Stat(modelPath); os.IsNotExist(errFinal) {
				t.Fatalf("无法找到 Casbin 模型文件进行测试。尝试路径: %s, %s, %s. Error: %v",
					"../../config/casbin_model.conf", altModelPath, "config/casbin_model.conf", errFinal)
			}
		}
	}


	enforcer, err := casbin.NewEnforcer(modelPath)
	if err != nil {
		t.Fatalf("无法创建测试用的 Casbin Enforcer: %v (model path: %s)", err, modelPath)
	}
	// （可选）如果测试需要特定的策略或角色，可以在这里添加到 enforcer 实例
	// enforcer.AddPolicy("test_role", "/test_resource", "read")
	// enforcer.AddGroupingPolicy("test_user", "test_role")

	// NewUserService 依赖 userRepo, enforcer, logger, viper
	userService := service.NewUserService(mockUserRepo, enforcer, sugaredLogger, tempViper)

	return userService, mockUserRepo, enforcer, ctrl
}

// TestUserService_RegisterUser_Success 测试用户成功注册的场景。
func TestUserService_RegisterUser_Success(t *testing.T) {
	userService, mockUserRepo, enforcer, ctrl := newTestUserServiceWithMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	registerReq := &dto.UserRegisterReq{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	// 预期行为：
	// 1. GetByUsername 检查用户名不存在
	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), registerReq.Username).Return(nil, repository.ErrNotFound)
	// 2. GetByEmail 检查邮箱不存在
	mockUserRepo.EXPECT().GetByEmail(gomock.Any(), registerReq.Email).Return(nil, repository.ErrNotFound)
	// 3. Create 创建用户
	mockUserRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, user *model.User) error {
			// 模拟数据库操作，例如赋予ID和默认角色
			assert.Equal(t, registerReq.Username, user.Username)
			assert.Equal(t, registerReq.Email, *user.Email)
			// 校验密码是否已哈希 (bcrypt.CompareHashAndPassword 会因盐值不同而不直接相等)
			// 我们可以简单地检查 PasswordHash 是否非空且与原始密码不同
			assert.NotEmpty(t, user.PasswordHash)
			assert.NotEqual(t, registerReq.Password, user.PasswordHash)
			// 模拟GORM填充默认值
			if user.Role == "" { // 确保Create之前没有设置，由数据库或GORM默认值填充
				user.Role = "role_user" // 模拟GORM填充默认值
			}
			user.ID = 1 // 模拟数据库赋予ID
			return nil
		},
	)

	// Casbin Enforcer 的 AddGroupingPolicy 预期会被调用
	// 由于我们使用的是真实的内存 Enforcer，这个调用会实际执行。
	// 如果要 mock Enforcer，需要定义 Enforcer 接口并生成 mock。
	// 当前测试中，我们依赖其正常工作，或者可以后续断言 Enforcer 的状态。
	// enforcer.EXPECT().AddGroupingPolicy(registerReq.Username, "role_user").Return(true, nil) // 如果 mock Enforcer

	// 执行注册
	createdUser, err := userService.RegisterUser(ctx, registerReq)

	// 断言结果
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.Equal(t, registerReq.Username, createdUser.Username)
	assert.Equal(t, registerReq.Email, *createdUser.Email)
	assert.Equal(t, uint(1), createdUser.ID) // 确认ID被赋予
	assert.Empty(t, createdUser.PasswordHash) // 确认返回的密码哈希为空
	assert.Equal(t, "role_user", createdUser.Role) // 确认角色被设置

	// （可选）验证 Casbin Enforcer 中是否添加了分组策略
	hasPolicy, err := enforcer.HasGroupingPolicy(registerReq.Username, "role_user")
	assert.NoError(t, err, "检查 Casbin 分组策略时出错")
	assert.True(t, hasPolicy, "Casbin 分组策略未按预期添加")
}

// TestUserService_RegisterUser_UsernameExists 测试用户名已存在时的注册失败场景。
func TestUserService_RegisterUser_UsernameExists(t *testing.T) {
	userService, mockUserRepo, _, ctrl := newTestUserServiceWithMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	registerReq := &dto.UserRegisterReq{
		Username: "existinguser",
		Email:    "test@example.com",
		Password: "password123",
	}

	// 预期行为：GetByUsername 返回已存在的用户
	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), registerReq.Username).Return(&model.User{ID: 1, Username: registerReq.Username}, nil)
	// GetByEmail 和 Create 不应被调用

	createdUser, err := userService.RegisterUser(ctx, registerReq)

	assert.Error(t, err) // 预期发生错误
	assert.Nil(t, createdUser)
	assert.Contains(t, err.Error(), fmt.Sprintf("用户名 '%s' 已被注册", registerReq.Username))
}

// TestUserService_RegisterUser_EmailExists 测试邮箱已存在时的注册失败场景。
func TestUserService_RegisterUser_EmailExists(t *testing.T) {
	userService, mockUserRepo, _, ctrl := newTestUserServiceWithMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	registerReq := &dto.UserRegisterReq{
		Username: "newuser",
		Email:    "existing@example.com",
		Password: "password123",
	}
	existingEmail := "existing@example.com"

	// 预期行为：
	// 1. GetByUsername 检查用户名不存在
	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), registerReq.Username).Return(nil, repository.ErrNotFound)
	// 2. GetByEmail 检查邮箱已存在
	mockUserRepo.EXPECT().GetByEmail(gomock.Any(), registerReq.Email).Return(&model.User{ID: 1, Email: &existingEmail}, nil)
	// Create 不应被调用

	createdUser, err := userService.RegisterUser(ctx, registerReq)

	assert.Error(t, err) // 预期发生错误
	assert.Nil(t, createdUser)
	assert.Contains(t, err.Error(), fmt.Sprintf("邮箱 '%s' 已被注册", registerReq.Email))
}


// TestUserService_LoginUser_Success 测试用户成功登录的场景。
func TestUserService_LoginUser_Success(t *testing.T) {
	userService, mockUserRepo, _, ctrl := newTestUserServiceWithMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	loginReq := &dto.UserLoginReq{
		UsernameOrEmail: "testuser",
		Password:        "password123",
	}
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(loginReq.Password), bcrypt.DefaultCost)
	expectedUser := &model.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
		Email:        nil, // 或者提供一个邮箱
		Role:         "role_user",
	}

	// 预期行为：
	// 1. GetByUsername (或 GetByEmail) 返回用户
	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), loginReq.UsernameOrEmail).Return(expectedUser, nil)

	loginRes, err := userService.LoginUser(ctx, loginReq)

	assert.NoError(t, err)
	assert.NotNil(t, loginRes)
	assert.NotEmpty(t, loginRes.AccessToken)
	assert.Equal(t, "Bearer", loginRes.TokenType)
	assert.True(t, loginRes.ExpiresIn > time.Now().Unix()) // 检查过期时间戳是否在未来
	assert.Equal(t, expectedUser.ID, loginRes.UserID)
	assert.Equal(t, expectedUser.Username, loginRes.Username)
}

// TestUserService_LoginUser_UserNotFound 测试登录时用户未找到的场景。
func TestUserService_LoginUser_UserNotFound(t *testing.T) {
	userService, mockUserRepo, _, ctrl := newTestUserServiceWithMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	loginReq := &dto.UserLoginReq{
		UsernameOrEmail: "nonexistentuser",
		Password:        "password123",
	}

	// 预期行为：GetByUsername 返回 ErrNotFound
	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), loginReq.UsernameOrEmail).Return(nil, repository.ErrNotFound)

	loginRes, err := userService.LoginUser(ctx, loginReq)

	assert.Error(t, err)
	assert.Nil(t, loginRes)
	assert.Contains(t, err.Error(), "不存在或密码错误")
}

// TestUserService_LoginUser_IncorrectPassword 测试登录时密码不正确的场景。
func TestUserService_LoginUser_IncorrectPassword(t *testing.T) {
	userService, mockUserRepo, _, ctrl := newTestUserServiceWithMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	loginReq := &dto.UserLoginReq{
		UsernameOrEmail: "testuser",
		Password:        "wrongpassword",
	}
	// 正确密码是 "password123"
	correctHashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	userWithCorrectPassword := &model.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: string(correctHashedPassword),
	}

	// 预期行为：GetByUsername 返回用户，但密码比较失败
	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), loginReq.UsernameOrEmail).Return(userWithCorrectPassword, nil)

	loginRes, err := userService.LoginUser(ctx, loginReq)

	assert.Error(t, err)
	assert.Nil(t, loginRes)
	assert.Contains(t, err.Error(), "不存在或密码错误")
}

// TestUserService_GetUserByUsername_Success 测试通过用户名成功获取用户的场景。
func TestUserService_GetUserByUsername_Success(t *testing.T) {
	userService, mockUserRepo, _, ctrl := newTestUserServiceWithMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	username := "testuser"
	expectedUser := &model.User{ID: 1, Username: username, Email: new(string), Role: "role_user"}
	*expectedUser.Email = "test@example.com"


	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), username).Return(expectedUser, nil)

	user, err := userService.GetUserByUsername(ctx, username)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.Username, user.Username)
	assert.Equal(t, *expectedUser.Email, *user.Email)
	assert.Empty(t, user.PasswordHash) // 密码哈希应为空
}

// TestUserService_GetUserByUsername_NotFound 测试通过用户名获取用户但用户未找到的场景。
func TestUserService_GetUserByUsername_NotFound(t *testing.T) {
	userService, mockUserRepo, _, ctrl := newTestUserServiceWithMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	username := "nonexistentuser"

	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), username).Return(nil, repository.ErrNotFound)

	user, err := userService.GetUserByUsername(ctx, username)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "未找到")
}

// TestUserService_GetUserByID_Success 测试通过ID成功获取用户的场景。
func TestUserService_GetUserByID_Success(t *testing.T) {
	userService, mockUserRepo, _, ctrl := newTestUserServiceWithMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := uint(1)
	expectedUser := &model.User{ID: userID, Username: "testuser", Email: new(string), Role: "role_user"}
	*expectedUser.Email = "test@example.com"


	mockUserRepo.EXPECT().GetByID(gomock.Any(), userID).Return(expectedUser, nil)

	user, err := userService.GetUserByID(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Username, user.Username)
	assert.Empty(t, user.PasswordHash) // 密码哈希应为空
}

// TestUserService_GetUserByID_NotFound 测试通过ID获取用户但用户未找到的场景。
func TestUserService_GetUserByID_NotFound(t *testing.T) {
	userService, mockUserRepo, _, ctrl := newTestUserServiceWithMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := uint(999)

	mockUserRepo.EXPECT().GetByID(gomock.Any(), userID).Return(nil, repository.ErrNotFound)

	user, err := userService.GetUserByID(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "未找到")
}
