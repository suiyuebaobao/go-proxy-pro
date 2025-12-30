/*
 * 文件作用：模型映射数据模型，定义模型名称转换配置
 * 负责功能：
 *   - 源模型到目标模型的映射定义
 *   - 映射优先级控制
 *   - 映射启用/禁用状态
 *   - 创建/更新请求结构
 * 重要程度：⭐⭐⭐ 一般（模型映射数据结构）
 * 依赖模块：gorm
 */
package model

import (
	"time"

	"gorm.io/gorm"
)

// ModelMapping 模型映射配置
// 用于将请求中的模型名映射到实际使用的模型名
type ModelMapping struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	SourceModel string         `json:"source_model" gorm:"type:varchar(100);index;not null;comment:源模型名（请求中的模型名）"`
	TargetModel string         `json:"target_model" gorm:"type:varchar(100);not null;comment:目标模型名（实际使用的模型名）"`
	Enabled     bool           `json:"enabled" gorm:"default:true;comment:是否启用"`
	Priority    int            `json:"priority" gorm:"default:0;comment:优先级，数值越大优先级越高"`
	Description string         `json:"description" gorm:"type:varchar(500);comment:描述"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (ModelMapping) TableName() string {
	return "model_mappings"
}

// CreateModelMappingRequest 创建模型映射请求
type CreateModelMappingRequest struct {
	SourceModel string `json:"source_model" binding:"required"`
	TargetModel string `json:"target_model" binding:"required"`
	Enabled     *bool  `json:"enabled"`
	Priority    int    `json:"priority"`
	Description string `json:"description"`
}

// UpdateModelMappingRequest 更新模型映射请求
type UpdateModelMappingRequest struct {
	SourceModel string `json:"source_model"`
	TargetModel string `json:"target_model"`
	Enabled     *bool  `json:"enabled"`
	Priority    *int   `json:"priority"`
	Description string `json:"description"`
}
