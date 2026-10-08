// Package main 提供一个可交互的 URL 归一化演示服务：
// 在网页上粘贴一批 CURL 或 URL，后端用 reverse-router-tree-skills 还原路由树，
// 返回树 JSON + 归一化资产清单 + 统计，前端用 D3 渲染成可交互的树形图。
//
// 运行：go run ./demo
// 打开：http://localhost:47177/（默认监听所有网卡，局域网内可通过 http://<本机IP>:47177/ 访问）
//
// 这是纯标准库实现（配合 SDK 本身零依赖），一个 go run 即可跑起整个演示。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
)

var (
	addr      = flag.String("addr", ":47177", "监听地址")
	prefix    = flag.String("prefix", "/", "前端页面与 API 的前缀（部署到子路径时使用）")
	apiPrefix string
)

// exitOn 是进程退出点。测试里换成记录函数，避免 log.Fatal 直接结束测试进程。
var osExit = os.Exit

var exitOn = func(v ...any) {
	if shouldExit(v) {
		log.Print(v...)
		osExit(1)
	}
}

// shouldExit 判断退出信息是否真的要结束进程。空信息表示正常结束。
func shouldExit(v []any) bool {
	return len(v) > 0 && v[0] != ""
}

func main() {
	exitOn(boot(flag.CommandLine, flag.Args()))
}

// boot 解析参数并启动监听。监听失败返回退出信息，而不是直接结束进程。
// 命令行已经解析过时不再重复解析，避免 flag 二次解析 panic。
func boot(fs *flag.FlagSet, args []string) string {
	if fs != nil && !fs.Parsed() {
		_ = fs.Parse(args)
	}
	apiPrefix = strings.TrimSuffix(*prefix, "/")
	if err := run(apiPrefix, *addr); err != nil {
		return serveError(err)
	}
	return ""
}

// run 组装路由并监听。返回监听错误，便于测试覆盖退出路径。
// listenAndServe 是监听点。测试替换它，避免真的占端口。
var listenAndServe = http.ListenAndServe

func run(prefix, addr string) error {
	log.Printf("URL 归一化演示已启动： http://localhost%s/", addr)
	log.Printf("API： POST %s/api/normalize", prefix)
	return listenAndServe(addr, newMux(prefix))
}

// newMux 组装 API 与静态页。前缀为空时页面挂在根路径，否则去掉前缀再找文件。
func newMux(prefix string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(prefix+"/api/normalize", handleNormalize)
	mux.HandleFunc(prefix+"/api/showcase", handleShowcase)
	mux.HandleFunc(prefix+"/api/health", handleHealth)

	fs := http.FileServer(http.Dir("demo/web"))
	if prefix != "" {
		fs = http.StripPrefix(prefix, fs)
	}
	mux.Handle(prefix+"/", fs)
	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"engine": "reverse-router-tree-skills",
	})
}

// showcaseLines 是首页「散落 URL → 一棵树」的演示样本，和首页文案同一批数据。
var showcaseLines = []string{
	"curl 'http://api.example.com/api/users/123'",
	"curl 'http://api.example.com/api/users/456'",
	"curl 'http://api.example.com/api/users/789'",
	"GET http://api.example.com/api/users/profile",
	"curl 'http://api.example.com/api/orders?status=paid'",
	"curl 'http://api.example.com/api/orders?status=shipped'",
}

// handleShowcase 只读演示：用固定样本跑一遍引擎，供首页画出真实还原结果。
func handleShowcase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持 GET / POST"})
		return
	}
	rRouter, result := reduceLines(showcaseLines)
	assets := collectAssets(rRouter)
	stats := rRouter.Tree.Stats()
	writeJSON(w, http.StatusOK, NormalizeResponse{
		Tree:   rRouter.Tree.ExportRoot(),
		Assets: assets,
		Stats:  &stats,
		Summary: Summary{
			InputLines: len(showcaseLines),
			Parsed:     result.Processed,
			Failed:     result.Failed,
			Assets:     len(assets),
		},
		Errors: toBatchErrors(result.Errors),
	})
}

// serveError 把监听失败整理成退出信息。
func serveError(err error) string {
	return fmt.Sprintf("服务启动失败: %v", err)
}

// NormalizeRequest 请求体：一行一个 CURL 或 URL（带可选的 method 前缀）。
type NormalizeRequest struct {
	Lines []string `json:"lines"`
}

// NormalizeResponse 响应：树 + 资产 + 统计。
type NormalizeResponse struct {
	Tree    *tree.RouteNodeJSON `json:"tree"`
	Assets  []AssetEntry        `json:"assets"`
	Stats   *tree.RouteStats    `json:"stats"`
	Summary Summary             `json:"summary"`
	Errors  []batchErrorJSON    `json:"errors"`
}

// batchErrorJSON 是失败样本的对外形态。router.BatchError.Err 是 error 接口，
// 默认序列化成空对象，前端读不到失败原因，这里改写成字符串。
type batchErrorJSON struct {
	Index int    `json:"index"`
	Raw   string `json:"raw"`
	Error string `json:"error"`
}

func toBatchErrors(in []router.BatchError) []batchErrorJSON {
	out := make([]batchErrorJSON, 0, len(in))
	for _, e := range in {
		msg := ""
		if e.Err != nil {
			msg = e.Err.Error()
		}
		out = append(out, batchErrorJSON{Index: e.Index, Raw: e.Raw, Error: msg})
	}
	return out
}

// AssetEntry 归一化资产条目（含 host，便于多目标展示）。
type AssetEntry struct {
	Host           string   `json:"host"`
	Method         string   `json:"method"`
	Template       string   `json:"template"`
	AssetKey       string   `json:"asset_key"`
	PathParams     []string `json:"path_params"`
	QueryParams    []string `json:"query_params"`
	RequiredParams []string `json:"required_params"`
}

// Summary 输入输出的量化总结。
type Summary struct {
	InputLines int `json:"input_lines"`
	Parsed     int `json:"parsed"`
	Failed     int `json:"failed"`
	Assets     int `json:"assets"`
}

func handleNormalize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持 POST"})
		return
	}
	var req NormalizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求体解析失败: " + err.Error()})
		return
	}
	if len(req.Lines) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "至少提供一行 URL 或 CURL"})
		return
	}

	rRouter, result := reduceLines(req.Lines)

	rootNode := rRouter.Tree.ExportRoot()

	// 资产清单（按 host 分组，稳定排序）
	assets := collectAssets(rRouter)
	stats := rRouter.Tree.Stats()

	writeJSON(w, http.StatusOK, NormalizeResponse{
		Tree:   rootNode,
		Assets: assets,
		Stats:  &stats,
		Summary: Summary{
			InputLines: len(req.Lines),
			Parsed:     result.Processed,
			Failed:     result.Failed,
			Assets:     len(assets),
		},
		Errors: toBatchErrors(result.Errors),
	})
}

// reduceLines 把一批混排的 CURL / "METHOD URL" / 裸 URL 还原成一棵树。
// CURL 与普通请求分开喂入，坏样本只记失败、不中断整批。
func reduceLines(lines []string) (*router.ReverseRouter, router.BatchResult) {
	var curls []string
	var reqs []*request.HttpRequest
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if looksLikeCurl(line) {
			curls = append(curls, line)
			continue
		}
		// 支持 "METHOD url" 与裸 url 两种写法
		method := "GET"
		raw := line
		if fields := strings.Fields(line); len(fields) >= 2 && isMethod(fields[0]) {
			method = strings.ToUpper(fields[0])
			raw = fields[1]
		}
		reqs = append(reqs, &request.HttpRequest{Method: method, Url: raw})
	}
	rRouter := router.NewReverseRouter()
	result := rRouter.ReverseCurls(curls)
	if len(reqs) > 0 {
		r2 := rRouter.ReverseRequests(reqs)
		result.Processed += r2.Processed
		result.Failed += r2.Failed
		result.Errors = append(result.Errors, r2.Errors...)
	}
	return rRouter, result
}

// collectAssets 从树中枚举归一化资产（去重后稳定排序）。
func collectAssets(r *router.ReverseRouter) []AssetEntry {
	return assetsFrom(r.ListAssets())
}

func assetsFrom(list []router.NormalizedRoute) []AssetEntry {
	seen := map[string]bool{}
	var out []AssetEntry
	for _, a := range list {
		key := a.AssetKey()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, AssetEntry{
			Host:           a.Host,
			Method:         a.Method,
			Template:       a.Template,
			AssetKey:       key,
			PathParams:     a.PathParams,
			QueryParams:    a.QueryParams,
			RequiredParams: a.RequiredParams,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AssetKey < out[j].AssetKey })
	return out
}

func looksLikeCurl(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "curl") || strings.HasPrefix(line, "curl ")
}

func isMethod(s string) bool {
	switch strings.ToUpper(s) {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("写 JSON 响应失败: %v", err)
	}
}
