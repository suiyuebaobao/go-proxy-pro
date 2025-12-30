/*
 * 文件作用：套餐管理处理器，处理套餐模板和用户套餐的CRUD操作
 * 负责功能：
 *   - 套餐模板管理（创建、更新、删除）
 *   - 用户套餐分配和管理
 *   - 用户可用套餐查询
 *   - 套餐状态管理（有效、过期）
 * 重要程度：⭐⭐⭐ 一般（套餐功能）
 * 依赖模块：repository, model
 */
package handler

import (
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/response"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PackageHandler struct {
	packageRepo     *repository.PackageRepository
	userPackageRepo *repository.UserPackageRepository
	apiKeyRepo      *repository.APIKeyRepository
}

func NewPackageHandler() *PackageHandler {
	return &PackageHandler{
		packageRepo:     repository.NewPackageRepository(),
		userPackageRepo: repository.NewUserPackageRepository(),
		apiKeyRepo:      repository.NewAPIKeyRepository(),
	}
}

// ========== 套餐模板管理 (管理员) ==========

// ListPackages 获取所有套餐
func (h *PackageHandler) ListPackages(c *gin.Context) {
	packages, err := h.packageRepo.GetAll()
	if err != nil {
		response.InternalError(c, "获取套餐列表失败")
		return
	}
	response.Success(c, packages)
}

// CreatePackage 创建套餐
func (h *PackageHandler) CreatePackage(c *gin.Context) {
	var req struct {
		Name          string  `json:"name" binding:"required"`
		Type          string  `json:"type" binding:"required,oneof=subscription quota"`
		Price         float64 `json:"price"`
		Duration      int     `json:"duration"`
		DailyQuota    float64 `json:"daily_quota"`    // 订阅类型：每日额度
		WeeklyQuota   float64 `json:"weekly_quota"`   // 订阅类型：每周额度
		MonthlyQuota  float64 `json:"monthly_quota"`  // 订阅类型：每月额度
		QuotaAmount   float64 `json:"quota_amount"`   // 额度类型：总额度
		AllowedModels string  `json:"allowed_models"` // 允许的模型
		Description   string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	pkg := &model.Package{
		Name:          req.Name,
		Type:          req.Type,
		Price:         req.Price,
		Duration:      req.Duration,
		DailyQuota:    req.DailyQuota,
		WeeklyQuota:   req.WeeklyQuota,
		MonthlyQuota:  req.MonthlyQuota,
		QuotaAmount:   req.QuotaAmount,
		AllowedModels: req.AllowedModels,
		Description:   req.Description,
		Status:        "active",
	}

	if err := h.packageRepo.Create(pkg); err != nil {
		response.InternalError(c, "创建套餐失败")
		return
	}

	response.Success(c, pkg)
}

// UpdatePackage 更新套餐
func (h *PackageHandler) UpdatePackage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	pkg, err := h.packageRepo.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "套餐不存在")
		return
	}

	var req struct {
		Name          string   `json:"name"`
		Price         *float64 `json:"price"`
		Duration      *int     `json:"duration"`
		DailyQuota    *float64 `json:"daily_quota"`
		WeeklyQuota   *float64 `json:"weekly_quota"`
		MonthlyQuota  *float64 `json:"monthly_quota"`
		QuotaAmount   *float64 `json:"quota_amount"`
		AllowedModels *string  `json:"allowed_models"`
		Description   string   `json:"description"`
		Status        string   `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.Name != "" {
		pkg.Name = req.Name
	}
	if req.Price != nil {
		pkg.Price = *req.Price
	}
	if req.Duration != nil {
		pkg.Duration = *req.Duration
	}
	if req.DailyQuota != nil {
		pkg.DailyQuota = *req.DailyQuota
	}
	if req.WeeklyQuota != nil {
		pkg.WeeklyQuota = *req.WeeklyQuota
	}
	if req.MonthlyQuota != nil {
		pkg.MonthlyQuota = *req.MonthlyQuota
	}
	if req.QuotaAmount != nil {
		pkg.QuotaAmount = *req.QuotaAmount
	}
	if req.AllowedModels != nil {
		pkg.AllowedModels = *req.AllowedModels
	}
	if req.Description != "" {
		pkg.Description = req.Description
	}
	if req.Status != "" {
		pkg.Status = req.Status
	}

	if err := h.packageRepo.Update(pkg); err != nil {
		response.InternalError(c, "更新套餐失败")
		return
	}

	response.Success(c, pkg)
}

// DeletePackage 删除套餐
func (h *PackageHandler) DeletePackage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.packageRepo.Delete(uint(id)); err != nil {
		response.InternalError(c, "删除套餐失败")
		return
	}

	response.Success(c, nil)
}

// ========== 用户套餐管理 (管理员) ==========

// ListUserPackages 获取用户的所有套餐
func (h *PackageHandler) ListUserPackages(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 64)

	packages, err := h.userPackageRepo.GetByUserID(uint(userID))
	if err != nil {
		response.InternalError(c, "获取用户套餐失败")
		return
	}

	response.Success(c, packages)
}

// AssignPackage 给用户分配套餐
func (h *PackageHandler) AssignPackage(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 64)

	var req struct {
		PackageID uint `json:"package_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 获取套餐模板
	pkg, err := h.packageRepo.GetByID(req.PackageID)
	if err != nil {
		response.NotFound(c, "套餐不存在")
		return
	}

	// 创建用户套餐
	now := time.Now()
	expire := now.AddDate(0, 0, pkg.Duration)

	up := &model.UserPackage{
		UserID:        uint(userID),
		PackageID:     req.PackageID,
		Name:          pkg.Name,
		Type:          pkg.Type,
		Status:        "active",
		StartTime:     &now,
		ExpireTime:    &expire,
		AllowedModels: pkg.AllowedModels,
	}

	if pkg.Type == "subscription" {
		// 订阅类型：复制周期额度限制
		up.DailyQuota = pkg.DailyQuota
		up.WeeklyQuota = pkg.WeeklyQuota
		up.MonthlyQuota = pkg.MonthlyQuota
	} else if pkg.Type == "quota" {
		// 额度类型：设置总额度
		up.QuotaTotal = pkg.QuotaAmount
		up.QuotaUsed = 0
	}

	if err := h.userPackageRepo.Create(up); err != nil {
		response.InternalError(c, "分配套餐失败")
		return
	}

	// 重新查询以获取关联的 Package 信息
	up, _ = h.userPackageRepo.GetByID(up.ID)

	response.Success(c, up)
}

// UpdateUserPackage 更新用户套餐
func (h *PackageHandler) UpdateUserPackage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	up, err := h.userPackageRepo.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "用户套餐不存在")
		return
	}

	var req struct {
		Status        string     `json:"status"`
		ExpireTime    *time.Time `json:"expire_time"`
		DailyQuota    *float64   `json:"daily_quota"`
		WeeklyQuota   *float64   `json:"weekly_quota"`
		MonthlyQuota  *float64   `json:"monthly_quota"`
		DailyUsed     *float64   `json:"daily_used"`
		WeeklyUsed    *float64   `json:"weekly_used"`
		MonthlyUsed   *float64   `json:"monthly_used"`
		QuotaTotal    *float64   `json:"quota_total"`
		QuotaUsed     *float64   `json:"quota_used"`
		AllowedModels *string    `json:"allowed_models"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.Status != "" {
		up.Status = req.Status
	}
	if req.ExpireTime != nil {
		up.ExpireTime = req.ExpireTime
	}
	if req.DailyQuota != nil {
		up.DailyQuota = *req.DailyQuota
	}
	if req.WeeklyQuota != nil {
		up.WeeklyQuota = *req.WeeklyQuota
	}
	if req.MonthlyQuota != nil {
		up.MonthlyQuota = *req.MonthlyQuota
	}
	if req.DailyUsed != nil {
		up.DailyUsed = *req.DailyUsed
	}
	if req.WeeklyUsed != nil {
		up.WeeklyUsed = *req.WeeklyUsed
	}
	if req.MonthlyUsed != nil {
		up.MonthlyUsed = *req.MonthlyUsed
	}
	if req.QuotaTotal != nil {
		up.QuotaTotal = *req.QuotaTotal
	}
	if req.QuotaUsed != nil {
		up.QuotaUsed = *req.QuotaUsed
	}
	if req.AllowedModels != nil {
		up.AllowedModels = *req.AllowedModels
	}

	if err := h.userPackageRepo.Update(up); err != nil {
		response.InternalError(c, "更新用户套餐失败")
		return
	}

	response.Success(c, up)
}

// DeleteUserPackage 删除用户套餐
func (h *PackageHandler) DeleteUserPackage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// 先解绑关联的 API Key，避免删除后 API Key 仍然指向已删除的套餐
	unbindCount, err := h.apiKeyRepo.UnbindPackage(uint(id))
	if err != nil {
		response.InternalError(c, "解绑API Key失败")
		return
	}

	if err := h.userPackageRepo.Delete(uint(id)); err != nil {
		response.InternalError(c, "删除用户套餐失败")
		return
	}

	response.Success(c, map[string]interface{}{
		"message":       "删除成功",
		"unbound_keys":  unbindCount,
	})
}

// ========== 用户自己的套餐 ==========

// GetMyPackages 获取当前用户的套餐
func (h *PackageHandler) GetMyPackages(c *gin.Context) {
	userID := c.GetUint("user_id")

	packages, err := h.userPackageRepo.GetByUserID(userID)
	if err != nil {
		response.InternalError(c, "获取套餐失败")
		return
	}

	response.Success(c, packages)
}

// GetMyActivePackages 获取当前用户的有效套餐
func (h *PackageHandler) GetMyActivePackages(c *gin.Context) {
	userID := c.GetUint("user_id")

	packages, err := h.userPackageRepo.GetActiveByUserID(userID)
	if err != nil {
		response.InternalError(c, "获取套餐失败")
		return
	}

	response.Success(c, packages)
}

// GetAvailablePackages 获取可购买的套餐列表
func (h *PackageHandler) GetAvailablePackages(c *gin.Context) {
	packages, err := h.packageRepo.GetActive()
	if err != nil {
		response.InternalError(c, "获取套餐列表失败")
		return
	}
	response.Success(c, packages)
}
