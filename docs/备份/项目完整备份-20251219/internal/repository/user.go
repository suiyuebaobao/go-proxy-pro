package repository

import (
	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{db: DB}
}

func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) GetByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) Delete(id uint) error {
	return r.db.Delete(&model.User{}, id).Error
}

func (r *UserRepository) List(page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	r.db.Model(&model.User{}).Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.Offset(offset).Limit(pageSize).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepository) ExistsByUsername(username string) bool {
	var count int64
	r.db.Model(&model.User{}).Where("username = ?", username).Count(&count)
	return count > 0
}

func (r *UserRepository) ExistsByEmail(email string) bool {
	var count int64
	r.db.Model(&model.User{}).Where("email = ?", email).Count(&count)
	return count > 0
}

// BatchUpdatePriceRate 批量更新用户费率倍率
func (r *UserRepository) BatchUpdatePriceRate(userIDs []uint, priceRate float64) error {
	return r.db.Model(&model.User{}).Where("id IN ?", userIDs).Update("price_rate", priceRate).Error
}

// UpdateAllPriceRate 更新所有用户费率倍率
func (r *UserRepository) UpdateAllPriceRate(priceRate float64) error {
	return r.db.Model(&model.User{}).Where("1 = 1").Update("price_rate", priceRate).Error
}

// GetByIDs 批量获取用户
func (r *UserRepository) GetByIDs(ids []uint) ([]model.User, error) {
	var users []model.User
	err := r.db.Where("id IN ?", ids).Find(&users).Error
	return users, err
}

// ListAll 获取所有用户（不分页）
func (r *UserRepository) ListAll() ([]model.User, error) {
	var users []model.User
	err := r.db.Find(&users).Error
	return users, err
}

// UserKeyBalance 用户 API Key 余额信息
type UserKeyBalance struct {
	QuotaKeyBalance         float64 // 额度 Key 的余额
	SubscriptionDailyRemain float64 // 订阅 Key 的当日剩余额度
}

// GetUserKeyBalances 批量获取用户的 API Key 余额信息
func (r *UserRepository) GetUserKeyBalances(userIDs []uint) (map[uint]UserKeyBalance, error) {
	result := make(map[uint]UserKeyBalance)
	if len(userIDs) == 0 {
		return result, nil
	}

	// 初始化所有用户的余额为 0
	for _, uid := range userIDs {
		result[uid] = UserKeyBalance{}
	}

	// 获取额度类型的用户套餐（quota）的余额：quota_total - quota_used
	// 直接从 user_packages 获取，不依赖 api_keys 关联
	type QuotaResult struct {
		UserID       uint
		QuotaBalance float64
	}
	var quotaResults []QuotaResult
	r.db.Raw(`
		SELECT user_id, COALESCE(SUM(quota_total - quota_used), 0) as quota_balance
		FROM user_packages
		WHERE user_id IN ? AND type = 'quota' AND status = 'active'
		GROUP BY user_id
	`, userIDs).Scan(&quotaResults)

	for _, qr := range quotaResults {
		if bal, ok := result[qr.UserID]; ok {
			bal.QuotaKeyBalance = qr.QuotaBalance
			result[qr.UserID] = bal
		}
	}

	// 获取订阅类型的用户套餐（subscription）的当日剩余额度：daily_quota - daily_used
	// 直接从 user_packages 获取，不依赖 api_keys 关联
	type SubscriptionResult struct {
		UserID      uint
		DailyRemain float64
	}
	var subResults []SubscriptionResult
	r.db.Raw(`
		SELECT user_id, COALESCE(SUM(CASE WHEN daily_quota > 0 THEN daily_quota - daily_used ELSE 999999 END), 0) as daily_remain
		FROM user_packages
		WHERE user_id IN ? AND type = 'subscription' AND status = 'active'
		GROUP BY user_id
	`, userIDs).Scan(&subResults)

	for _, sr := range subResults {
		if bal, ok := result[sr.UserID]; ok {
			// 如果是 999999 表示无限额度，显示为 -1
			if sr.DailyRemain >= 999999 {
				bal.SubscriptionDailyRemain = -1
			} else {
				bal.SubscriptionDailyRemain = sr.DailyRemain
			}
			result[sr.UserID] = bal
		}
	}

	return result, nil
}
