/*
 * 文件作用：用户管理服务，处理用户认证和用户管理的业务逻辑
 * 负责功能：
 *   - 用户登录/注册
 *   - 用户CRUD操作
 *   - 密码修改
 *   - JWT Token生成
 *   - 费率倍率批量更新
 * 重要程度：⭐⭐⭐⭐ 重要（用户管理核心）
 * 依赖模块：repository, model, utils
 */
package service

import (
	"errors"
	"sync"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
	"go-aiproxy/pkg/utils"

	"gorm.io/gorm"
)

var (
	userLog     *logger.Logger
	userLogOnce sync.Once
)

func getUserLog() *logger.Logger {
	userLogOnce.Do(func() {
		userLog = logger.GetLogger("main")
	})
	return userLog
}

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService() *UserService {
	return &UserService{
		repo: repository.NewUserRepository(),
	}
}

type LoginRequest struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	CaptchaID string `json:"captcha_id"`
	Captcha   string `json:"captcha"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

type RegisterRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Password  string `json:"password" binding:"required,min=6"`
	Email     string `json:"email" binding:"omitempty,email"`
	CaptchaID string `json:"captcha_id"`
	Captcha   string `json:"captcha"`
}

// AdminCreateUserRequest 管理员创建用户请求
type AdminCreateUserRequest struct {
	Username       string  `json:"username" binding:"required,min=3,max=50"`
	Password       string  `json:"password" binding:"required,min=6"`
	Email          string  `json:"email" binding:"omitempty,email"`
	Role           string  `json:"role" binding:"omitempty,oneof=admin user"`
	Status         string  `json:"status" binding:"omitempty,oneof=active disabled"`
	PriceRate      float64 `json:"price_rate"`
	MaxConcurrency int     `json:"max_concurrency"`
}

type UpdateUserRequest struct {
	Email          string   `json:"email" binding:"omitempty,email"`
	Status         string   `json:"status" binding:"omitempty,oneof=active disabled"`
	Role           string   `json:"role" binding:"omitempty,oneof=admin user"`
	PriceRate      *float64 `json:"price_rate"`      // 使用指针以区分是否传入
	MaxConcurrency *int     `json:"max_concurrency"` // 使用指针以区分是否传入
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (s *UserService) Login(req *LoginRequest) (*LoginResponse, error) {
	getUserLog().Info("[user] 用户登录请求 | Username: %s", req.Username)

	user, err := s.repo.GetByUsername(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			getUserLog().Info("[user] 登录失败 | Username: %s | 原因: 用户不存在", req.Username)
			return nil, errors.New("invalid username or password")
		}
		getUserLog().Error("[user] 登录失败 | Username: %s | 原因: 数据库错误: %v", req.Username, err)
		return nil, err
	}

	if !user.CheckPassword(req.Password) {
		getUserLog().Info("[user] 登录失败 | Username: %s | 原因: 密码错误", req.Username)
		return nil, errors.New("invalid username or password")
	}

	if user.Status != "active" {
		getUserLog().Info("[user] 登录失败 | Username: %s | 原因: 用户已禁用", req.Username)
		return nil, errors.New("user is disabled")
	}

	token, err := utils.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		getUserLog().Error("[user] 登录失败 | Username: %s | 原因: Token 生成失败: %v", req.Username, err)
		return nil, err
	}

	getUserLog().Info("[user] 登录成功 | UserID: %d | Username: %s | Role: %s", user.ID, user.Username, user.Role)

	return &LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *UserService) Register(req *RegisterRequest) (*model.User, error) {
	getUserLog().Info("[user] 用户注册请求 | Username: %s | Email: %s", req.Username, req.Email)

	if s.repo.ExistsByUsername(req.Username) {
		getUserLog().Info("[user] 注册失败 | Username: %s | 原因: 用户名已存在", req.Username)
		return nil, errors.New("username already exists")
	}

	if req.Email != "" && s.repo.ExistsByEmail(req.Email) {
		getUserLog().Info("[user] 注册失败 | Username: %s | 原因: 邮箱已存在", req.Username)
		return nil, errors.New("email already exists")
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     "user",
		Status:   "active",
	}

	if err := user.SetPassword(req.Password); err != nil {
		getUserLog().Error("[user] 注册失败 | Username: %s | 原因: 设置密码失败: %v", req.Username, err)
		return nil, err
	}

	if err := s.repo.Create(user); err != nil {
		getUserLog().Error("[user] 注册失败 | Username: %s | 原因: 创建用户失败: %v", req.Username, err)
		return nil, err
	}

	getUserLog().Info("[user] 注册成功 | UserID: %d | Username: %s", user.ID, user.Username)
	return user, nil
}

// AdminCreateUser 管理员创建用户
func (s *UserService) AdminCreateUser(req *AdminCreateUserRequest) (*model.User, error) {
	getUserLog().Info("[user] 管理员创建用户请求 | Username: %s | Email: %s | Role: %s", req.Username, req.Email, req.Role)

	if s.repo.ExistsByUsername(req.Username) {
		getUserLog().Info("[user] 创建用户失败 | Username: %s | 原因: 用户名已存在", req.Username)
		return nil, errors.New("用户名已存在")
	}

	if req.Email != "" && s.repo.ExistsByEmail(req.Email) {
		getUserLog().Info("[user] 创建用户失败 | Username: %s | 原因: 邮箱已被使用", req.Username)
		return nil, errors.New("邮箱已被使用")
	}

	// 设置默认值
	role := req.Role
	if role == "" {
		role = "user"
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	priceRate := req.PriceRate
	if priceRate == 0 {
		priceRate = 1.0
	}
	maxConcurrency := req.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 10 // 默认并发限制
	}

	user := &model.User{
		Username:       req.Username,
		Email:          req.Email,
		Role:           role,
		Status:         status,
		PriceRate:      priceRate,
		MaxConcurrency: maxConcurrency,
	}

	if err := user.SetPassword(req.Password); err != nil {
		getUserLog().Error("[user] 创建用户失败 | Username: %s | 原因: 设置密码失败: %v", req.Username, err)
		return nil, err
	}

	if err := s.repo.Create(user); err != nil {
		getUserLog().Error("[user] 创建用户失败 | Username: %s | 原因: 数据库错误: %v", req.Username, err)
		return nil, err
	}

	getUserLog().Info("[user] 管理员创建用户成功 | UserID: %d | Username: %s | Role: %s", user.ID, user.Username, user.Role)
	return user, nil
}

func (s *UserService) GetByID(id uint) (*model.User, error) {
	return s.repo.GetByID(id)
}

func (s *UserService) Update(id uint, req *UpdateUserRequest) (*model.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Status != "" {
		user.Status = req.Status
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.PriceRate != nil {
		user.PriceRate = *req.PriceRate
	}
	if req.MaxConcurrency != nil {
		user.MaxConcurrency = *req.MaxConcurrency
	}

	if err := s.repo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *UserService) List(page, pageSize int) ([]model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize)
}

func (s *UserService) ChangePassword(userID uint, req *ChangePasswordRequest) error {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return err
	}

	if !user.CheckPassword(req.OldPassword) {
		return errors.New("incorrect old password")
	}

	if err := user.SetPassword(req.NewPassword); err != nil {
		return err
	}

	return s.repo.Update(user)
}

// BatchUpdatePriceRateRequest 批量更新用户费率倍率请求
type BatchUpdatePriceRateRequest struct {
	UserIDs   []uint  `json:"user_ids" binding:"required"`   // 用户ID列表
	PriceRate float64 `json:"price_rate" binding:"required"` // 费率倍率
}

// UpdateAllPriceRateRequest 更新所有用户费率倍率请求
type UpdateAllPriceRateRequest struct {
	PriceRate float64 `json:"price_rate" binding:"required"` // 费率倍率
}

// BatchUpdatePriceRate 批量更新用户费率倍率
func (s *UserService) BatchUpdatePriceRate(req *BatchUpdatePriceRateRequest) error {
	if len(req.UserIDs) == 0 {
		return errors.New("user_ids cannot be empty")
	}
	if req.PriceRate < 0 {
		return errors.New("price_rate cannot be negative")
	}
	return s.repo.BatchUpdatePriceRate(req.UserIDs, req.PriceRate)
}

// UpdateAllPriceRate 更新所有用户费率倍率
func (s *UserService) UpdateAllPriceRate(priceRate float64) error {
	if priceRate < 0 {
		return errors.New("price_rate cannot be negative")
	}
	return s.repo.UpdateAllPriceRate(priceRate)
}

// GetUsersByIDs 批量获取用户信息
func (s *UserService) GetUsersByIDs(ids []uint) ([]model.User, error) {
	return s.repo.GetByIDs(ids)
}

// ListAll 获取所有用户（不分页，用于管理员批量操作）
func (s *UserService) ListAll() ([]model.User, error) {
	return s.repo.ListAll()
}

// UserWithKeyBalances 用户信息带 API Key 余额
type UserWithKeyBalances struct {
	model.User
	QuotaKeyBalance         float64 `json:"quota_key_balance"`          // 额度 Key 的余额（套餐总额度 - 已用）
	SubscriptionDailyRemain float64 `json:"subscription_daily_remain"`  // 订阅 Key 的当日剩余额度
}

// ListWithKeyBalances 获取用户列表并带 API Key 余额信息
func (s *UserService) ListWithKeyBalances(page, pageSize int) ([]UserWithKeyBalances, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := s.repo.List(page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 获取所有用户的 ID
	userIDs := make([]uint, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	// 批量获取用户的 API Key 余额信息
	balances, err := s.repo.GetUserKeyBalances(userIDs)
	if err != nil {
		// 如果获取失败，返回原始用户数据（余额为 0）
		result := make([]UserWithKeyBalances, len(users))
		for i, u := range users {
			result[i] = UserWithKeyBalances{User: u}
		}
		return result, total, nil
	}

	// 组装结果
	result := make([]UserWithKeyBalances, len(users))
	for i, u := range users {
		result[i] = UserWithKeyBalances{
			User:                    u,
			QuotaKeyBalance:         balances[u.ID].QuotaKeyBalance,
			SubscriptionDailyRemain: balances[u.ID].SubscriptionDailyRemain,
		}
	}

	return result, total, nil
}
