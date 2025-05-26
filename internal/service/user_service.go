// Package service 包含了应用的业务逻辑层。
// Service 层负责处理具体的业务规则，协调 Repository 层进行数据操作，
// 并为 Handler 层（例如 API 端点处理函数）提供服务。
package service

import (
	"context" // 用于在请求处理和依赖调用中传递截止日期、取消信号等。
	"fmt"     // 用于格式化错误信息。
	"strings" // 导入 strings 包

	"go-base-system-v2/internal/dto"       // 数据传输对象，用于服务层方法的参数和返回值。
	"go-base-system-v2/internal/model"     // 数据模型，代表数据库中的实体。
	"go-base-system-v2/internal/repository" // Repository 层接口，用于数据持久化操作。
	"go-base-system-v2/pkg/jwtutil"        // 导入 JWT 工具包

	"github.com/casbin/casbin/v2" // 导入 Casbin 包
	"github.com/google/wire"     // Wire 包，用于依赖注入。
	"github.com/spf13/viper"     // 导入 Viper，用于 JWT 配置
	"go.uber.org/zap"            // Zap 日志库，用于结构化日志记录。
	"golang.org/x/crypto/bcrypt" // 用于密码哈希处理。
)

// UserService 定义了用户相关业务逻辑的服务接口。
// 它抽象了用户注册、信息查询等核心业务功能。
type UserService interface {
	// RegisterUser 处理用户注册的业务逻辑。
	// ctx: 请求上下文。
	// req: 指向 dto.UserRegisterReq 的指针，包含用户注册所需的信息 (用户名, 邮箱, 密码)。
	// 返回:
	//   - *model.User: 成功注册后的用户信息 (密码字段应为空)。
	//   - error: 如果注册过程中发生错误 (例如用户名已存在、邮箱已存在、密码处理失败、数据库操作失败等)，则返回错误。
	RegisterUser(ctx context.Context, req *dto.UserRegisterReq) (*model.User, error)

	// GetUserByUsername 根据用户名查询用户。
	// ctx: 请求上下文。
	// username: 要查询的用户名。
	// 返回:
	//   - *model.User: 找到的用户信息 (密码字段应为空)。
	//   - error: 如果用户未找到或查询过程中发生其他错误。
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)

	// GetUserByID 根据用户ID查询用户。
	// ctx: 请求上下文。
	// id: 要查询的用户ID。
	// 返回:
	//   - *model.User: 找到的用户信息 (密码字段应为空)。
	//   - error: 如果用户未找到或查询过程中发生其他错误。
	GetUserByID(ctx context.Context, id uint) (*model.User, error)

	// LoginUser 处理用户登录的业务逻辑。
	// ctx: 请求上下文。
	// req: 指向 dto.UserLoginReq 的指针，包含用户的登录凭据 (用户名/邮箱, 密码)。
	// 返回:
	//   - *dto.UserLoginRes: 成功登录后包含 JWT 信息的响应对象。
	//   - error: 如果登录失败 (例如，用户不存在、密码错误、Token 生成失败等)，则返回错误。
	LoginUser(ctx context.Context, req *dto.UserLoginReq) (*dto.UserLoginRes, error)
}

// userServiceImpl 是 UserService 接口的具体实现。
// 它依赖 UserRepository, Casbin Enforcer, Viper 配置和 Zap SugaredLogger。
type userServiceImpl struct {
	userRepo repository.UserRepository // 用户数据仓库依赖
	enforcer *casbin.Enforcer          // Casbin Enforcer 实例，用于权限管理
	logger   *zap.SugaredLogger        // 日志记录器依赖
	cfg      *viper.Viper              // Viper 配置实例，用于 JWT 等
}

// NewUserService 是 userServiceImpl 的构造函数 Provider。
// 此函数用于 Wire 进行依赖注入，创建一个新的 UserService 实例。
// 参数:
//   userRepo: 实现了 repository.UserRepository 接口的实例。
//   enforcer: Casbin Enforcer 实例。
//   logger: Zap SugaredLogger 实例。
//   cfg: Viper 配置实例。
// 返回:
//   UserService 接口的实例。
func NewUserService(
	userRepo repository.UserRepository,
	enforcer *casbin.Enforcer, // 注入 Casbin Enforcer
	logger *zap.SugaredLogger,
	cfg *viper.Viper,
) UserService {
	if userRepo == nil {
		panic("NewUserService: UserRepository is nil")
	}
	if enforcer == nil {
		panic("NewUserService: Casbin Enforcer is nil")
	}
	if logger == nil {
		panic("NewUserService: Zap SugaredLogger is nil")
	}
	if cfg == nil {
		panic("NewUserService: Viper config instance is nil")
	}
	return &userServiceImpl{
		userRepo: userRepo,
		enforcer: enforcer, // 存储 Enforcer
		logger:   logger,
		cfg:      cfg,
	}
}

// RegisterUser 实现用户注册的业务逻辑。
func (s *userServiceImpl) RegisterUser(ctx context.Context, req *dto.UserRegisterReq) (*model.User, error) {
	s.logger.Infow("Attempting to register new user", "username", req.Username, "email", req.Email)

	// 1. 检查用户名是否已存在
	s.logger.Debugw("Checking if username exists", "username", req.Username)
	existingUser, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil && err != repository.ErrNotFound { // 如果是 ErrNotFound 以外的错误
		s.logger.Errorw("Error checking username existence", "username", req.Username, "error", err)
		return nil, fmt.Errorf("检查用户名时发生错误: %w", err)
	}
	if existingUser != nil {
		s.logger.Warnw("Username already registered", "username", req.Username)
		return nil, fmt.Errorf("用户名 '%s' 已被注册", req.Username)
	}

	// 2. 检查邮箱是否已存在
	s.logger.Debugw("Checking if email exists", "email", req.Email)
	existingUserByEmail, err := s.userRepo.GetByEmail(ctx, req.Email) // 使用新变量名避免覆盖
	if err != nil && err != repository.ErrNotFound { // 如果是 ErrNotFound 以外的错误
		s.logger.Errorw("Error checking email existence", "email", req.Email, "error", err)
		return nil, fmt.Errorf("检查邮箱时发生错误: %w", err)
	}
	if existingUserByEmail != nil {
		s.logger.Warnw("Email already registered", "email", req.Email)
		return nil, fmt.Errorf("邮箱 '%s' 已被注册", req.Email)
	}

	// 3. 对密码进行哈希处理
	s.logger.Debug("Hashing password")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Errorw("Failed to hash password", "error", err)
		return nil, fmt.Errorf("密码处理失败: %w", err)
	}

	// 4. 创建 model.User 实例
	// Role 字段将使用数据库定义的默认值 "role_user" (在 model.User 中设置)
	user := &model.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Email:        &req.Email,
		// Role: "role_user", // GORM 将使用字段标签中的 default 值，此处无需显式设置
	}

	// 5. 调用 Repository 创建用户
	s.logger.Debugw("Creating user in repository", "username", user.Username)
	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Errorw("Failed to create user in repository", "username", user.Username, "error", err)
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}
	s.logger.Infow("User record created successfully in DB", "userID", user.ID, "username", user.Username, "defaultRole", user.Role)

	// 6. 为新用户添加 Casbin 分组策略 (用户-角色映射)
	// user.Role 此时应已被 GORM 用默认值 "role_user" 填充 (如果数据库支持并正确配置了默认值返回)
	// 或者，如果数据库不返回默认值，我们需要在创建后重新查询用户以获取 Role，或在此处硬编码默认角色。
	// 为了简单起见，我们假设 Role 字段在 Create 后被 GORM 正确填充或我们直接使用默认值。
	// 如果 user.Role 为空，则手动设置为 "role_user"。
	if user.Role == "" {
		user.Role = "role_user" // 确保角色不为空
		s.logger.Debugw("User role was empty after DB create, setting to default", "userID", user.ID, "defaultRole", user.Role)
		// (可选) 如果需要，可以在此处更新数据库中的用户角色，但这通常由 GORM 的 default 标签处理。
	}

	// AddGroupingPolicy(subject, role)
	// subject 通常是用户名或用户唯一标识符
	// role 是角色名
	added, err := s.enforcer.AddGroupingPolicy(user.Username, user.Role)
	if err != nil {
		// 如果添加分组策略失败，这可能是一个严重问题，需要记录并可能回滚用户创建。
		// 为了简化，这里只记录错误。在生产环境中，应考虑事务和补偿措施。
		s.logger.Errorw("Failed to add Casbin grouping policy for new user",
			"userID", user.ID, "username", user.Username, "role", user.Role, "error", err)
		// 注意：即使这里失败，用户记录已创建。根据业务需求决定是否需要回滚。
		// return nil, fmt.Errorf("为新用户设置权限失败: %w", err) // 可选：将此视为注册失败
	}
	if added {
		s.logger.Infow("Casbin grouping policy added for new user", "username", user.Username, "role", user.Role)
	} else {
		s.logger.Infow("Casbin grouping policy already exists for user or no change needed", "username", user.Username, "role", user.Role)
	}
	// (可选) 主动保存 Casbin 策略到持久化存储 (如果适配器没有自动保存)
	// if err := s.enforcer.SavePolicy(); err != nil {
	// 	s.logger.Errorw("Failed to save Casbin policy after adding grouping policy", "error", err)
	// }


	s.logger.Infow("User registered and initial role assigned successfully", "userID", user.ID, "username", user.Username, "role", user.Role)

	// 7. 返回用户信息（密码字段置空）
	user.PasswordHash = "" // 不返回密码哈希
	return user, nil
}

// GetUserByUsername 实现根据用户名查询用户的业务逻辑。
func (s *userServiceImpl) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	s.logger.Infow("Attempting to get user by username from service", "username", username)

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if err == repository.ErrNotFound { // 注意：这里比较的是 repository.ErrNotFound
			s.logger.Infow("User not found by username in service", "username", username)
			return nil, fmt.Errorf("用户 '%s' 未找到", username) // 返回业务层面的错误
		}
		s.logger.Errorw("Error getting user by username from repository in service", "username", username, "error", err)
		return nil, fmt.Errorf("查询用户 '%s' 失败: %w", username, err)
	}

	s.logger.Infow("User found by username in service", "userID", user.ID, "username", user.Username)
	user.PasswordHash = "" // 不返回密码哈希
	return user, nil
}

// GetUserByID 实现根据用户ID查询用户的业务逻辑。
func (s *userServiceImpl) GetUserByID(ctx context.Context, id uint) (*model.User, error) {
	s.logger.Infow("Attempting to get user by ID from service", "userID", id)

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound { // 注意：这里比较的是 repository.ErrNotFound
			s.logger.Infow("User not found by ID in service", "userID", id)
			return nil, fmt.Errorf("用户ID '%d' 未找到", id) // 返回业务层面的错误
		}
		s.logger.Errorw("Error getting user by ID from repository in service", "userID", id, "error", err)
		return nil, fmt.Errorf("查询用户ID '%d' 失败: %w", id, err)
	}

	s.logger.Infow("User found by ID in service", "userID", user.ID, "username", user.Username)
	user.PasswordHash = "" // 不返回密码哈希
	return user, nil
}

// ProviderSet 是 service 包的 Wire Provider Set。
// 由于 NewUserService 返回 UserService 接口类型，我们直接将其添加到 ProviderSet 中。
var ProviderSet = wire.NewSet(NewUserService)

// LoginUser 实现用户登录的业务逻辑。
func (s *userServiceImpl) LoginUser(ctx context.Context, req *dto.UserLoginReq) (*dto.UserLoginRes, error) {
	s.logger.Infow("Attempting to login user", "usernameOrEmail", req.UsernameOrEmail)

	var user *model.User
	var err error

	// 1. 根据 UsernameOrEmail 查询用户
	// 简单的判断是否为邮箱格式
	if strings.Contains(req.UsernameOrEmail, "@") {
		s.logger.Debugw("Attempting to find user by email", "email", req.UsernameOrEmail)
		user, err = s.userRepo.GetByEmail(ctx, req.UsernameOrEmail)
	} else {
		s.logger.Debugw("Attempting to find user by username", "username", req.UsernameOrEmail)
		user, err = s.userRepo.GetByUsername(ctx, req.UsernameOrEmail)
	}

	if err != nil {
		if err == repository.ErrNotFound {
			s.logger.Warnw("User not found during login", "usernameOrEmail", req.UsernameOrEmail)
			return nil, fmt.Errorf("用户 '%s' 不存在或密码错误", req.UsernameOrEmail) // 统一错误信息，避免泄露用户是否存在
		}
		s.logger.Errorw("Error finding user during login", "usernameOrEmail", req.UsernameOrEmail, "error", err)
		return nil, fmt.Errorf("登录时查询用户失败: %w", err)
	}

	// 2. 验证密码
	s.logger.Debugw("Verifying password for user", "userID", user.ID, "username", user.Username)
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		// 包括 bcrypt.ErrMismatchedHashAndPassword 和其他潜在错误
		s.logger.Warnw("Password verification failed for user", "userID", user.ID, "username", user.Username, "error", err)
		return nil, fmt.Errorf("用户 '%s' 不存在或密码错误", req.UsernameOrEmail) // 统一错误信息
	}

	// 3. 生成 JWT
	// 使用注入的 s.cfg (Viper 实例) 来获取 JWT 配置
	tokenString, expiresAt, err := jwtutil.GenerateToken(user.ID, user.Username, s.cfg)
	if err != nil {
		s.logger.Errorw("Failed to generate JWT for user", "userID", user.ID, "username", user.Username, "error", err)
		return nil, fmt.Errorf("生成认证令牌失败: %w", err)
	}

	s.logger.Infow("User logged in successfully and JWT generated", "userID", user.ID, "username", user.Username)

	// 4. 构建并返回 UserLoginRes
	return &dto.UserLoginRes{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   expiresAt,
		UserID:      user.ID,
		Username:    user.Username,
	}, nil
}

// 确保 userServiceImpl 实现了 UserService 接口 (编译时检查)。
var _ UserService = (*userServiceImpl)(nil)
