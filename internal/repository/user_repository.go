// Package repository 提供了数据持久化层的抽象和实现。
// 它定义了与数据存储（如数据库）交互的接口，以及这些接口的具体实现。
package repository

import (
	"context" // 导入 context 包，用于在请求处理和数据库操作中传递截止日期、取消信号等。
	"errors"  // 导入 errors 包，用于创建自定义错误类型。

	"go-base-system-v2/internal/model" // 导入应用的数据模型包。

	"github.com/google/wire" // 导入 Wire 包，用于依赖注入。
	"go.uber.org/zap"        // 导入 Zap 日志库。
	"gorm.io/gorm"           // 导入 GORM 数据库操作库。
)

// ErrNotFound 是一个自定义错误类型，用于表示当数据库查询未找到记录时的情况。
// 这使得上层代码可以区分“未找到”和“其他数据库错误”。
var ErrNotFound = errors.New("record not found in repository")

// UserRepository 定义了用户数据仓库的接口。
// 它抽象了用户数据的持久化操作，使得业务逻辑层可以独立于具体的数据库实现。
// 所有方法都接收一个 context.Context 参数，以支持超时和取消等操作。
type UserRepository interface {
	// Create 方法用于在数据库中创建一个新的用户记录。
	// ctx: 请求上下文，用于控制数据库操作的超时或取消。
	// user: 指向 model.User 结构体的指针，包含了要创建的用户信息。
	// 如果创建成功，返回 nil；否则返回一个错误。
	Create(ctx context.Context, user *model.User) error

	// GetByUsername 方法用于根据用户名从数据库中检索一个用户记录。
	// ctx: 请求上下文。
	// username: 要查找的用户名。
	// 如果找到用户，返回一个指向 model.User 结构体的指针和 nil 错误。
	// 如果未找到用户，返回 nil 和 repository.ErrNotFound。
	// 如果发生其他数据库错误，返回 nil 和相应的错误信息。
	GetByUsername(ctx context.Context, username string) (*model.User, error)

	// GetByEmail 方法用于根据电子邮件地址从数据库中检索一个用户记录。
	// ctx: 请求上下文。
	// email: 要查找的电子邮件地址。
	// 如果找到用户，返回一个指向 model.User 结构体的指针和 nil 错误。
	// 如果未找到用户，返回 nil 和 repository.ErrNotFound。
	// 如果发生其他数据库错误，返回 nil 和相应的错误信息。
	GetByEmail(ctx context.Context, email string) (*model.User, error)

	// GetByID 方法用于根据用户 ID 从数据库中检索一个用户记录。
	// ctx: 请求上下文。
	// id: 要查找的用户 ID (uint 类型)。
	// 如果找到用户，返回一个指向 model.User 结构体的指针和 nil 错误。
	// 如果未找到用户，返回 nil 和 repository.ErrNotFound。
	// 如果发生其他数据库错误，返回 nil 和相应的错误信息。
	GetByID(ctx context.Context, id uint) (*model.User, error)
}

// userRepositoryImpl 是 UserRepository 接口的 GORM 实现。
// 它嵌入了一个 *gorm.DB 实例（用于数据库操作）和一个 *zap.SugaredLogger 实例（用于日志记录）。
type userRepositoryImpl struct {
	db     *gorm.DB           // GORM 数据库连接实例
	logger *zap.SugaredLogger // Zap SugaredLogger 实例，用于结构化日志记录
}

// NewUserRepository 是 userRepositoryImpl 的构造函数 Provider。
// 它接收一个 GORM DB 连接实例和一个 Zap SugaredLogger 实例作为依赖，
// 并返回一个实现了 UserRepository 接口的实例。
// 此函数用于 Wire 进行依赖注入。
// 参数:
//   db: GORM 数据库连接实例。如果为 nil，函数将 panic。
//   logger: Zap SugaredLogger 实例。如果为 nil，函数将 panic。
// 返回:
//   UserRepository 接口的实例。
func NewUserRepository(db *gorm.DB, logger *zap.SugaredLogger) UserRepository { // 返回接口类型
	if db == nil {
		panic("NewUserRepository: GORM DB instance is nil")
	}
	if logger == nil {
		panic("NewUserRepository: Zap SugaredLogger instance is nil")
	}
	return &userRepositoryImpl{db: db, logger: logger} // 返回具体类型，Wire 会通过 Bind 转换
}

// Create 在数据库中创建一个新的用户记录。
// 它使用传入的 context 进行数据库操作，并记录操作的开始和结果。
func (r *userRepositoryImpl) Create(ctx context.Context, user *model.User) error {
	r.logger.Debugw("Attempting to create user in repository", "username", user.Username, "email", user.Email)

	// 使用 db.WithContext(ctx) 确保数据库操作可以响应上下文的取消或超时。
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		r.logger.Errorw("Failed to create user in repository", "username", user.Username, "error", err)
		return err // 直接返回 GORM 错误
	}

	r.logger.Infow("User created successfully in repository", "userID", user.ID, "username", user.Username)
	return nil
}

// GetByUsername 根据用户名检索用户。
// 如果未找到用户，返回 (nil, repository.ErrNotFound)。
func (r *userRepositoryImpl) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	r.logger.Debugw("Attempting to get user by username from repository", "username", username)
	var user model.User

	// 使用 First 方法，如果记录未找到，它会返回 gorm.ErrRecordNotFound。
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Infow("User not found by username in repository", "username", username)
			return nil, ErrNotFound // 返回自定义的 ErrNotFound
		}
		r.logger.Errorw("Failed to get user by username from repository", "username", username, "error", err)
		return nil, err // 其他数据库错误
	}

	r.logger.Debugw("User found by username in repository", "userID", user.ID, "username", user.Username)
	return &user, nil
}

// GetByEmail 根据电子邮件地址检索用户。
// 如果未找到用户，返回 (nil, repository.ErrNotFound)。
func (r *userRepositoryImpl) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	r.logger.Debugw("Attempting to get user by email from repository", "email", email)
	var user model.User

	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Infow("User not found by email in repository", "email", email)
			return nil, ErrNotFound // 返回自定义的 ErrNotFound
		}
		r.logger.Errorw("Failed to get user by email from repository", "email", email, "error", err)
		return nil, err // 其他数据库错误
	}

	r.logger.Debugw("User found by email in repository", "userID", user.ID, "email", email)
	return &user, nil
}

// GetByID 根据用户 ID 检索用户。
// 如果未找到用户，返回 (nil, repository.ErrNotFound)。
func (r *userRepositoryImpl) GetByID(ctx context.Context, id uint) (*model.User, error) {
	r.logger.Debugw("Attempting to get user by ID from repository", "userID", id)
	var user model.User

	err := r.db.WithContext(ctx).First(&user, id).Error // GORM 可以通过主键直接查询
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Infow("User not found by ID in repository", "userID", id)
			return nil, ErrNotFound // 返回自定义的 ErrNotFound
		}
		r.logger.Errorw("Failed to get user by ID from repository", "userID", id, "error", err)
		return nil, err // 其他数据库错误
	}

	r.logger.Debugw("User found by ID in repository", "userID", user.ID)
	return &user, nil
}

// ProviderSet 是 repository 包的 Wire Provider Set。
// 由于 NewUserRepository 返回 UserRepository 接口类型，
// 我们直接将其添加到 ProviderSet 中。
var ProviderSet = wire.NewSet(NewUserRepository)

// 确保 userRepositoryImpl 实现了 UserRepository 接口 (编译时检查)。
var _ UserRepository = (*userRepositoryImpl)(nil)
