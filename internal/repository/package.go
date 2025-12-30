/*
 * 文件作用：套餐数据仓库，提供套餐和用户套餐的数据库操作
 * 负责功能：
 *   - 套餐模板CRUD操作
 *   - 用户套餐分配和管理
 *   - 额度使用量更新（原子操作）
 *   - 套餐状态自动更新（过期/耗尽）
 * 重要程度：⭐⭐⭐ 一般（套餐管理仓库）
 * 依赖模块：model, gorm
 */
package repository

import (
	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

type PackageRepository struct {
	db *gorm.DB
}

func NewPackageRepository() *PackageRepository {
	return &PackageRepository{db: DB}
}

// ========== Package (套餐模板) ==========

// Create 创建套餐
func (r *PackageRepository) Create(pkg *model.Package) error {
	return r.db.Create(pkg).Error
}

// GetByID 根据 ID 获取套餐
func (r *PackageRepository) GetByID(id uint) (*model.Package, error) {
	var pkg model.Package
	err := r.db.First(&pkg, id).Error
	if err != nil {
		return nil, err
	}
	return &pkg, nil
}

// GetAll 获取所有套餐
func (r *PackageRepository) GetAll() ([]model.Package, error) {
	var packages []model.Package
	err := r.db.Order("type, price").Find(&packages).Error
	return packages, err
}

// GetActive 获取所有启用的套餐
func (r *PackageRepository) GetActive() ([]model.Package, error) {
	var packages []model.Package
	err := r.db.Where("status = ?", "active").Order("type, price").Find(&packages).Error
	return packages, err
}

// Update 更新套餐
func (r *PackageRepository) Update(pkg *model.Package) error {
	return r.db.Save(pkg).Error
}

// Delete 删除套餐
func (r *PackageRepository) Delete(id uint) error {
	return r.db.Delete(&model.Package{}, id).Error
}

// ========== UserPackage (用户套餐) ==========

type UserPackageRepository struct {
	db *gorm.DB
}

func NewUserPackageRepository() *UserPackageRepository {
	return &UserPackageRepository{db: DB}
}

// Create 创建用户套餐
func (r *UserPackageRepository) Create(up *model.UserPackage) error {
	return r.db.Create(up).Error
}

// GetByID 根据 ID 获取用户套餐
func (r *UserPackageRepository) GetByID(id uint) (*model.UserPackage, error) {
	var up model.UserPackage
	err := r.db.Preload("Package").First(&up, id).Error
	if err != nil {
		return nil, err
	}
	return &up, nil
}

// GetByUserID 获取用户的所有套餐
func (r *UserPackageRepository) GetByUserID(userID uint) ([]model.UserPackage, error) {
	var packages []model.UserPackage
	err := r.db.Preload("Package").Where("user_id = ?", userID).Order("created_at DESC").Find(&packages).Error
	return packages, err
}

// GetActiveByUserID 获取用户的有效套餐
func (r *UserPackageRepository) GetActiveByUserID(userID uint) ([]model.UserPackage, error) {
	var packages []model.UserPackage
	err := r.db.Preload("Package").Where("user_id = ? AND status = ?", userID, "active").Order("created_at DESC").Find(&packages).Error
	return packages, err
}

// Update 更新用户套餐
func (r *UserPackageRepository) Update(up *model.UserPackage) error {
	return r.db.Save(up).Error
}

// UpdateQuotaUsed 更新已用额度（原子操作）
func (r *UserPackageRepository) UpdateQuotaUsed(id uint, amount float64) error {
	return r.db.Model(&model.UserPackage{}).Where("id = ?", id).
		Update("quota_used", gorm.Expr("quota_used + ?", amount)).Error
}

// Delete 删除用户套餐
func (r *UserPackageRepository) Delete(id uint) error {
	return r.db.Delete(&model.UserPackage{}, id).Error
}

// ExpireSubscriptions 将过期的订阅标记为已过期
func (r *UserPackageRepository) ExpireSubscriptions() (int64, error) {
	result := r.db.Model(&model.UserPackage{}).
		Where("type = ? AND status = ? AND expire_time < NOW()", "subscription", "active").
		Update("status", "expired")
	return result.RowsAffected, result.Error
}

// ExhaustQuotas 将额度耗尽的套餐标记为已耗尽
func (r *UserPackageRepository) ExhaustQuotas() (int64, error) {
	result := r.db.Model(&model.UserPackage{}).
		Where("type = ? AND status = ? AND quota_used >= quota_total", "quota", "active").
		Update("status", "exhausted")
	return result.RowsAffected, result.Error
}

// IncrementUsage 增加套餐使用量（原子操作）
// 对于订阅类型：增加 daily_used, weekly_used, monthly_used
// 对于额度类型：增加 quota_used
func (r *UserPackageRepository) IncrementUsage(id uint, pkgType string, amount float64) error {
	if pkgType == "subscription" {
		return r.db.Model(&model.UserPackage{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"daily_used":   gorm.Expr("daily_used + ?", amount),
				"weekly_used":  gorm.Expr("weekly_used + ?", amount),
				"monthly_used": gorm.Expr("monthly_used + ?", amount),
			}).Error
	} else if pkgType == "quota" {
		return r.db.Model(&model.UserPackage{}).Where("id = ?", id).
			Update("quota_used", gorm.Expr("quota_used + ?", amount)).Error
	}
	return nil
}
