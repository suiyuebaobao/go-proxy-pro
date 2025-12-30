package main

import (
	"encoding/json"
	"fmt"
	"go-aiproxy/internal/config"
	"go-aiproxy/internal/repository"
)

func main() {
	// 加载配置
	if err := config.Load("configs/config.yaml"); err != nil {
		fmt.Println("Config load error:", err)
		return
	}

	// 初始化数据库
	if err := repository.InitMySQL(); err != nil {
		fmt.Println("MySQL init error:", err)
		return
	}
	
	repo := repository.NewAccountRepository()
	accounts, total, err := repo.List(1, 10, "", "")
	if err != nil {
		fmt.Println("List error:", err)
		return
	}
	
	fmt.Printf("Total: %d\n\n", total)
	
	for _, acc := range accounts {
		fmt.Printf("ID: %d, Name: %s, Type: %s\n", acc.ID, acc.Name, acc.Type)
		fmt.Printf("  FiveHourUtilization: %v\n", acc.FiveHourUtilization)
		fmt.Printf("  SevenDayUtilization: %v\n", acc.SevenDayUtilization)
		fmt.Printf("  SevenDaySonnetUtilization: %v\n", acc.SevenDaySonnetUtilization)
		
		// 测试 JSON 序列化
		data, _ := json.Marshal(acc)
		fmt.Printf("  JSON length: %d\n", len(data))
		
		// 检查 JSON 是否包含 utilization 字段
		var m map[string]interface{}
		json.Unmarshal(data, &m)
		if v, ok := m["five_hour_utilization"]; ok {
			fmt.Printf("  five_hour_utilization in JSON: %v\n", v)
		} else {
			fmt.Printf("  five_hour_utilization NOT in JSON!\n")
		}
		fmt.Println()
	}
}
