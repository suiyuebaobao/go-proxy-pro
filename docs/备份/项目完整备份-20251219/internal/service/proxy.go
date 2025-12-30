package service

import (
	"errors"
	"sync"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"

	"gorm.io/gorm"
)

var (
	proxyService     *ProxyService
	proxyServiceOnce sync.Once
)

type ProxyService struct {
	log *logger.Logger
}

func GetProxyService() *ProxyService {
	proxyServiceOnce.Do(func() {
		proxyService = &ProxyService{
			log: logger.GetLogger("proxy"),
		}
	})
	return proxyService
}

// List 获取代理列表
func (s *ProxyService) List(page, pageSize int, keyword string) ([]model.Proxy, int64, error) {
	var proxies []model.Proxy
	var total int64

	query := repository.DB.Model(&model.Proxy{})

	if keyword != "" {
		query = query.Where("name LIKE ? OR host LIKE ? OR remark LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	if err := query.Order("id DESC").Find(&proxies).Error; err != nil {
		return nil, 0, err
	}

	return proxies, total, nil
}

// GetByID 根据 ID 获取代理
func (s *ProxyService) GetByID(id uint) (*model.Proxy, error) {
	var proxy model.Proxy
	if err := repository.DB.First(&proxy, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &proxy, nil
}

// Create 创建代理
func (s *ProxyService) Create(proxy *model.Proxy) error {
	if err := repository.DB.Create(proxy).Error; err != nil {
		s.log.Error("创建代理失败: %v", err)
		return err
	}
	s.log.Info("创建代理成功: %s (%s:%d)", proxy.Name, proxy.Host, proxy.Port)
	return nil
}

// Update 更新代理
func (s *ProxyService) Update(proxy *model.Proxy) error {
	if err := repository.DB.Save(proxy).Error; err != nil {
		s.log.Error("更新代理失败: %v", err)
		return err
	}
	s.log.Info("更新代理成功: %s (%s:%d)", proxy.Name, proxy.Host, proxy.Port)
	return nil
}

// Delete 删除代理
func (s *ProxyService) Delete(id uint) error {
	// 先检查是否有账户在使用这个代理
	var count int64
	if err := repository.DB.Model(&model.Account{}).Where("proxy_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该代理正在被账户使用，无法删除")
	}

	if err := repository.DB.Delete(&model.Proxy{}, id).Error; err != nil {
		s.log.Error("删除代理失败: %v", err)
		return err
	}
	s.log.Info("删除代理成功: ID=%d", id)
	return nil
}

// GetEnabledProxies 获取所有启用的代理
func (s *ProxyService) GetEnabledProxies() ([]model.Proxy, error) {
	var proxies []model.Proxy
	if err := repository.DB.Where("enabled = ?", true).Find(&proxies).Error; err != nil {
		return nil, err
	}
	return proxies, nil
}

// ToggleEnabled 切换代理启用状态
func (s *ProxyService) ToggleEnabled(id uint) error {
	var proxy model.Proxy
	if err := repository.DB.First(&proxy, id).Error; err != nil {
		return err
	}
	proxy.Enabled = !proxy.Enabled
	return repository.DB.Save(&proxy).Error
}

// GetDefaultProxy 获取默认代理（用于 OAuth 认证）
func (s *ProxyService) GetDefaultProxy() (*model.Proxy, error) {
	var proxy model.Proxy
	if err := repository.DB.Where("is_default = ? AND enabled = ?", true, true).First(&proxy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &proxy, nil
}

// SetDefault 设置默认代理
func (s *ProxyService) SetDefault(id uint) error {
	// 先取消所有代理的默认状态
	if err := repository.DB.Model(&model.Proxy{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		return err
	}

	// 设置指定代理为默认
	if id > 0 {
		if err := repository.DB.Model(&model.Proxy{}).Where("id = ?", id).Update("is_default", true).Error; err != nil {
			return err
		}
		s.log.Info("设置默认代理: ID=%d", id)
	} else {
		s.log.Info("清除默认代理")
	}

	return nil
}

// ClearDefault 清除默认代理
func (s *ProxyService) ClearDefault() error {
	return s.SetDefault(0)
}

// UpdateTestStatus 更新代理测试状态
func (s *ProxyService) UpdateTestStatus(id uint, status string, latency int, errMsg string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"test_status":  status,
		"test_latency": latency,
		"test_error":   errMsg,
		"last_test_at": now,
	}

	if err := repository.DB.Model(&model.Proxy{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		s.log.Error("更新代理测试状态失败: %v", err)
		return err
	}

	s.log.Info("更新代理测试状态: ID=%d, status=%s, latency=%dms", id, status, latency)
	return nil
}
