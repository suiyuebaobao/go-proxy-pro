/*
 * 文件作用：告警通知服务，负责告警触发、发送和静默控制
 * 负责功能：
 *   - 根据条件类型匹配已启用规则并触发告警
 *   - 支持 Telegram Bot API / Webhook / SMTP 邮件三种通知渠道
 *   - 相同告警在静默期内不重复发送
 *   - 提供测试发送功能
 * 重要程度：⭐⭐⭐⭐ 重要（运维告警核心逻辑）
 * 依赖模块：repository, model
 */
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
)

type AlertService struct {
	db         *repository.AlertRepository
	silenceMap sync.Map // map[string]time.Time — key: "ruleID:conditionType"
}

var (
	alertService *AlertService
	alertOnce    sync.Once
)

func GetAlertService() *AlertService {
	alertOnce.Do(func() {
		alertService = &AlertService{
			db: repository.NewAlertRepository(),
		}
	})
	return alertService
}

// TriggerAlert fires an alert for the given condition type and message.
// It respects silence windows and sends to all matching enabled rules.
func (s *AlertService) TriggerAlert(conditionType, message string) {
	log := logger.GetLogger("alert")
	rules, err := s.db.GetEnabledRulesByCondition(conditionType)
	if err != nil || len(rules) == 0 {
		return
	}

	for _, rule := range rules {
		silenceKey := fmt.Sprintf("%d:%s", rule.ID, conditionType)
		if lastFired, ok := s.silenceMap.Load(silenceKey); ok {
			if time.Since(lastFired.(time.Time)) < time.Duration(rule.SilenceMinutes)*time.Minute {
				continue
			}
		}

		var sendErr error
		switch rule.ChannelType {
		case "telegram":
			sendErr = s.sendTelegram(rule.ChannelConfig, rule.Name, message)
		case "webhook":
			sendErr = s.sendWebhook(rule.ChannelConfig, rule.Name, conditionType, message)
		case "email":
			sendErr = s.sendEmail(rule.ChannelConfig, rule.Name, message)
		default:
			sendErr = fmt.Errorf("unknown channel: %s", rule.ChannelType)
		}

		status := "sent"
		errMsg := ""
		if sendErr != nil {
			status = "failed"
			errMsg = sendErr.Error()
			log.Error("告警发送失败 | Rule: %s | Channel: %s | Error: %v", rule.Name, rule.ChannelType, sendErr)
		} else {
			s.silenceMap.Store(silenceKey, time.Now())
			log.Info("告警发送成功 | Rule: %s | Channel: %s", rule.Name, rule.ChannelType)
		}

		_ = s.db.CreateLog(&model.AlertLog{
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			AlertType: conditionType,
			Message:   message,
			Channel:   rule.ChannelType,
			Status:    status,
			Error:     errMsg,
		})
	}
}

// TestSend sends a test alert for a specific rule
func (s *AlertService) TestSend(rule *model.AlertRule) error {
	msg := fmt.Sprintf("[测试告警] 规则 \"%s\" 的通知测试", rule.Name)
	switch rule.ChannelType {
	case "telegram":
		return s.sendTelegram(rule.ChannelConfig, rule.Name, msg)
	case "webhook":
		return s.sendWebhook(rule.ChannelConfig, rule.Name, "test", msg)
	case "email":
		return s.sendEmail(rule.ChannelConfig, rule.Name, msg)
	default:
		return fmt.Errorf("unknown channel: %s", rule.ChannelType)
	}
}

type telegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

func (s *AlertService) sendTelegram(configJSON, ruleName, message string) error {
	var cfg telegramConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return fmt.Errorf("invalid telegram config: %w", err)
	}
	if cfg.BotToken == "" || cfg.ChatID == "" {
		return fmt.Errorf("missing bot_token or chat_id")
	}

	text := fmt.Sprintf("🚨 *Go-AIProxy 告警*\n\n*规则:* %s\n*详情:* %s\n*时间:* %s", ruleName, message, time.Now().Format("2006-01-02 15:04:05"))
	payload, _ := json.Marshal(map[string]string{
		"chat_id":    cfg.ChatID,
		"text":       text,
		"parse_mode": "Markdown",
	})

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

type webhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (s *AlertService) sendWebhook(configJSON, ruleName, conditionType, message string) error {
	var cfg webhookConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("missing webhook url")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"rule_name":      ruleName,
		"condition_type": conditionType,
		"message":        message,
		"timestamp":      time.Now().Format(time.RFC3339),
		"source":         "Go-AIProxy",
	})

	req, _ := http.NewRequest("POST", cfg.URL, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

type emailConfig struct {
	SMTPHost string `json:"smtp_host"`
	SMTPPort string `json:"smtp_port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"` // comma-separated
}

func (s *AlertService) sendEmail(configJSON, ruleName, message string) error {
	var cfg emailConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return fmt.Errorf("invalid email config: %w", err)
	}
	if cfg.SMTPHost == "" || cfg.To == "" {
		return fmt.Errorf("missing smtp_host or recipients")
	}

	subject := fmt.Sprintf("Go-AIProxy 告警: %s", ruleName)
	body := fmt.Sprintf("规则: %s\n详情: %s\n时间: %s", ruleName, message, time.Now().Format("2006-01-02 15:04:05"))

	from := cfg.From
	if from == "" {
		from = cfg.Username
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, cfg.To, subject, body)

	addr := cfg.SMTPHost + ":" + cfg.SMTPPort
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	recipients := strings.Split(cfg.To, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	return smtp.SendMail(addr, auth, from, recipients, []byte(msg))
}
