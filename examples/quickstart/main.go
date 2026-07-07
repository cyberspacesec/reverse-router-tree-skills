// Package main 演示 reverse-router-tree 的端到端用法：从一组 HTTP 请求
// 还原路由树，并导出为 OpenAPI 3.0.3 规范。
//
// 运行：go run ./examples/quickstart
//
// 这是给上层集成项目的最小参考样板：喂数据 → 拿路由树 → 导出规范。
package main

import (
	"fmt"
	"log"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/exporter"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
)

func main() {
	r := router.NewReverseRouter()

	// 模拟黑盒抓包流量：一个 RESTful 资源 /api/users 的 CRUD 序列。
	// 数字 ID 会被自动合并为 {users_id} 路径变量；query 参数会被还原；
	// Authorization 头会被推断为 Bearer 安全方案。
	samples := []struct {
		url    string
		method string
		body   string
	}{
		{"/api/users?page=1&size=20", "GET", ""},
		{"/api/users?page=2&size=20", "GET", ""},
		{"/api/users/123", "GET", ""},
		{"/api/users/456", "GET", ""},
		{"/api/users/789", "GET", ""}, // 第 3 个数字 ID → 触发合并为 {users_id}
		{"/api/users", "POST", `{"name":"alice","age":30}`},
		{"/api/users/123", "PUT", `{"name":"alice2"}`},
		{"/api/users/123", "DELETE", ""},
	}

	for _, s := range samples {
		h := request.Headers{}
		h.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiJ9...")
		h.Set("Content-Type", "application/json")
		var body []byte
		if s.body != "" {
			body = []byte(s.body)
		}
		req := request.NewHttpRequest(s.url, h, s.method, body)
		if err := r.ReverseHttpRequest(req); err != nil {
			log.Fatalf("处理请求 %s %s 失败: %v", s.method, s.url, err)
		}
	}

	// 喂完数据后推断必需参数（基于出现频率）。
	r.InferRequiredParams()

	// 打印还原出的路由树。
	fmt.Println("=== 还原路由树 ===")
	fmt.Println(r.Tree.String())

	// 导出为 OpenAPI 3.0.3。
	exp := exporter.NewOpenAPIExporter()
	exp.Title = "Users API (Reverse Engineered)"
	exp.Version = "1.0.0"
	exp.ServerURL = "https://api.example.com"
	doc, err := exp.Export(r.Tree)
	if err != nil {
		log.Fatalf("导出 OpenAPI 失败: %v", err)
	}

	fmt.Println("=== OpenAPI 3.0.3 规范 ===")
	fmt.Println(string(doc))

	// 统计指标。
	st := r.GetStats()
	fmt.Printf("=== 统计 ===\n请求数: %d  路径变量: %d  参数: %d  类型推断: %d\n",
		st.RequestsProcessed, st.PathVariablesIdentified, st.ParamsCreated, st.TypeInferences)
}
