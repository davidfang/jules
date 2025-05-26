package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型定义了数据库中用户表的结构。
// 它包含了用户的基本信息，如ID、用户名、密码哈希、电子邮件以及时间戳。
type User struct {
	// ID 是用户的主键，类型为 uint，并且在数据库中自增。
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// Username 是用户的唯一标识名。
	// 设置了唯一索引 (uniqueIndex) 以确保用户名的唯一性。
	// 不能为空 (not null)。
	Username string `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`

	// PasswordHash 存储用户密码的哈希值。
	// 不能为空 (not null)。
	PasswordHash string `gorm:"type:varchar(255);not null" json:"-"` // JSON输出时忽略

	// Email 是用户的电子邮件地址。
	// 设置了唯一索引 (uniqueIndex) 以确保电子邮件的唯一性。
	// 允许为空 (default:null)，因为用户可能在注册初期不提供邮箱。
	Email *string `gorm:"type:varchar(100);uniqueIndex;default:null" json:"email,omitempty"` // omitempty表示如果为空则在JSON中忽略

	// CreatedAt 记录用户创建的时间。
	// GORM 会在创建记录时自动填充此字段。
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 记录用户最后更新的时间。
	// GORM 会在创建和更新记录时自动填充此字段。
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 用于 GORM 的软删除功能。
	// 如果设置了此字段，记录在调用 Delete 时不会真正从数据库中删除，而是将此字段设置为当前时间。
	// 查询时会自动过滤掉已软删除的记录。
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // JSON输出时忽略

	// Role 存储用户的角色，例如 "role_user", "role_admin"。
	// gorm:"type:varchar(50)" 指定字段类型。
	// gorm:"default:'role_user'" 指定数据库中此字段的默认值为 "role_user"。
	// gorm:"not null" 指定字段不能为空。
	// json:"role,omitempty" 指定 JSON 字段名，并在值为空时从 JSON 输出中省略此字段。
	Role string `gorm:"type:varchar(50);not null;default:'role_user'" json:"role,omitempty"`
}

// TableName 方法指定了 User 模型对应的数据库表名。
// GORM 默认会使用结构体名称的蛇形复数形式作为表名 (例如 "users")，
// 但显式定义此方法可以提供更好的控制和清晰度。
func (User) TableName() string {
	return "users" // 指定表名为 "users"
}

// BeforeCreate 是一个 GORM Hook，在创建记录之前自动调用。
// 可以在这里添加一些记录创建前的默认值设置或数据校验逻辑。
// func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
// 	if u.Username == "" {
// 		return errors.New("username can't be empty")
// 	}
// 	// 可以在这里进行密码加密等操作
// 	return
// }

// BeforeUpdate 是一个 GORM Hook，在更新记录之前自动调用。
// func (u *User) BeforeUpdate(tx *gorm.DB) (err error) {
// 	// 可以在这里添加更新前的逻辑，例如更新时间戳或进行数据验证
// 	return
// }
