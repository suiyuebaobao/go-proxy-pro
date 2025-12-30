package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Username  string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"size:100;not null" json:"-"`
	Email     string         `gorm:"size:100;uniqueIndex" json:"email,omitempty"`
	Role      string         `gorm:"size:20;default:user" json:"role"`     // admin, user
	Status    string         `gorm:"size:20;default:active" json:"status"` // active, disabled
	Balance        float64        `gorm:"type:decimal(10,4);default:0" json:"balance"`      // 余额（美元）
	PriceRate      float64        `gorm:"type:decimal(5,2);default:1.0" json:"price_rate"`  // 价格倍率，默认1.0（原价），0表示免费
	MaxConcurrency int            `gorm:"default:10" json:"max_concurrency"`               // 最大并发数
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

func (u *User) TableName() string {
	return "users"
}
