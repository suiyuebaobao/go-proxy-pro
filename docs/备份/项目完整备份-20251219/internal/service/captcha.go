package service

import (
	"sync"
	"time"

	"github.com/mojocn/base64Captcha"
)

// CaptchaService 验证码服务
type CaptchaService struct {
	store  base64Captcha.Store
	driver *base64Captcha.DriverString
}

var (
	captchaService *CaptchaService
	captchaOnce    sync.Once
)

// customStore 自定义存储，支持过期时间
type customStore struct {
	data sync.Map
}

type storeItem struct {
	value   string
	expires time.Time
}

func (s *customStore) Set(id string, value string) error {
	s.data.Store(id, storeItem{
		value:   value,
		expires: time.Now().Add(5 * time.Minute), // 5分钟过期
	})
	return nil
}

func (s *customStore) Get(id string, clear bool) string {
	v, ok := s.data.Load(id)
	if !ok {
		return ""
	}
	item := v.(storeItem)
	if time.Now().After(item.expires) {
		s.data.Delete(id)
		return ""
	}
	if clear {
		s.data.Delete(id)
	}
	return item.value
}

func (s *customStore) Verify(id, answer string, clear bool) bool {
	v := s.Get(id, clear)
	return v != "" && v == answer
}

// GetCaptchaService 获取验证码服务单例
func GetCaptchaService() *CaptchaService {
	captchaOnce.Do(func() {
		// 配置验证码驱动
		driver := &base64Captcha.DriverString{
			Height:          60,
			Width:           200,
			NoiseCount:      50,
			ShowLineOptions: base64Captcha.OptionShowSlimeLine | base64Captcha.OptionShowHollowLine,
			Length:          4,
			Source:          "23456789abcdefghjkmnpqrstuvwxyz", // 排除容易混淆的字符
			Fonts:           []string{"wqy-microhei.ttc"},
		}

		captchaService = &CaptchaService{
			store:  &customStore{},
			driver: driver,
		}
	})
	return captchaService
}

// CaptchaResponse 验证码响应
type CaptchaResponse struct {
	CaptchaID string `json:"captcha_id"`
	Image     string `json:"image"` // base64 图片
}

// Generate 生成验证码
func (s *CaptchaService) Generate() (*CaptchaResponse, error) {
	captcha := base64Captcha.NewCaptcha(s.driver, s.store)
	id, b64s, _, err := captcha.Generate()
	if err != nil {
		return nil, err
	}
	return &CaptchaResponse{
		CaptchaID: id,
		Image:     b64s,
	}, nil
}

// Verify 验证验证码
func (s *CaptchaService) Verify(id, answer string) bool {
	// 转小写比较，因为验证码不区分大小写
	return s.store.Verify(id, answer, true)
}
