// 官网静态内容：文案与演示数据集中在这里，改文案不动组件。

export const REPO_URL = 'https://github.com/cyberspacesec/reverse-router-tree-skills'
export const DOCS_URL = '/reverse-router-tree-skills/docs/'

/** 导航锚点 */
export const NAV_ITEMS = [
  { key: 'problems', label: '解决什么问题' },
  { key: 'features', label: '核心能力' },
  { key: 'normalize', label: 'URL 归一化' },
  { key: 'quickstart', label: '快速开始' },
  { key: 'docs', label: '教学文档', href: DOCS_URL },
]

/** 痛点：输入散乱 URL，输出真实路由结构 */
export const PAIN_POINTS = [
  {
    title: '爬虫越爬越多，接口全貌越来越糊',
    desc: '爬虫把 /api/users/123 和 /api/users/456 当成两个 URL 各请求一遍，几十万条 URL 里看不出目标应用到底有多少接口。',
  },
  {
    title: '安全扫描器对同一接口重复测试',
    desc: '扫描器无法识别"这两条 URL 是同一个接口"，测试项成倍膨胀，还漏掉真正没测过的路径。',
  },
  {
    title: '测绘平台 URL 资产散乱、无法聚合',
    desc: '同一目标应用的接口在资产库里散落成海量碎片，去重靠字符串相似度，检索与资产清点无从下手。',
  },
]

/** 归一化对照演示：左原始 URL → 右归一化结果 */
export interface NormalizeDemoRow {
  raw: string
  result: string
  note: string
}

export const NORMALIZE_DEMO: NormalizeDemoRow[] = [
  { raw: '/api/users/123', result: '/api/users/{users_id}', note: '纯数字 → 变量 (integer)' },
  { raw: '/api/users/456', result: '/api/users/{users_id}', note: '归入同一模板' },
  { raw: '/api/users/789', result: '/api/users/{users_id}', note: '归入同一模板' },
  {
    raw: '/api/users?page=1&size=20',
    result: '/api/users?page & size',
    note: '查询参数还原 + 必需性推断',
  },
  {
    raw: 'POST /api/users (json)',
    result: 'requestBody: name, age',
    note: '请求体参数解析',
  },
]

/** 核心能力卡片 */
export const FEATURES = [
  {
    title: '路径变量识别',
    desc: '纯数字 / UUID / 手机号 / 身份证号 / 银行卡号 / 车牌号 / 前缀后缀模式自动合并为 {var}，只合并匹配模式的兄弟节点，list、create 等固定路径不误合并。',
  },
  {
    title: '查询参数与请求体还原',
    desc: 'JSON / 表单 / multipart 请求体解析，JSON 嵌套点号扁平化，参数名大小写不敏感；基于出现频率推断必需参数，阈值可配。',
  },
  {
    title: '两层类型推断',
    desc: '物理类型（integer / string / ...）+ 逻辑类型（uuid / phone / idcard / ...），变量不只是占位符，还知道它是什么。',
  },
  {
    title: 'URL 资产归一化',
    desc: 'Detailed 变体返回机器可读失败原因，NormalizeReport 批量明细替代静默丢弃，ListAssets 枚举全量资产，curl / 裸 URL 直达入口，读操作不建桶。',
  },
  {
    title: '多目标 Host 隔离',
    desc: 'RouterSet 按 host 分桶，每个目标应用独立还原路由树，避免跨目标污染；HostAssetKey 支撑跨目标资产清单。',
  },
  {
    title: '项目级隔离管理',
    desc: 'ProjectManager 按安全测试项目管理多个 RouterSet，同 host 在不同项目间互不污染，配额与配置向已有项目传播。',
  },
  {
    title: '可续喂 + 生产护栏',
    desc: 'JSON 序列化保留样本计数，分批采集导入后可继续推断；资源上限、敏感参数脱敏、fail-soft 护栏默认开启。',
  },
  {
    title: 'OpenAPI 3.0.3 导出',
    desc: '路径 / 参数 / 请求体 / 安全方案（从 Authorization 推断 Bearer / Basic / Digest），黑盒流量直接产出"Swagger"。',
  },
]

/** 性能 / 质量指标 */
export const METRICS = [
  { value: '146万+', label: '单核 URL 处理吞吐 /秒' },
  { value: '5', label: '归一化失败机器可读原因' },
  { value: '16', label: 'atomic 统计指标' },
  { value: '0', label: '外部依赖（纯 Go 标准库）' },
]

/** 三层 API 对照 */
export const API_LAYERS = [
  {
    name: 'ReverseRouter',
    scope: '单目标',
    desc: '一棵路由树的还原与归一化：ReverseCurls 批量喂数据，ListAssets / NormalizeURLString 直达资产。',
  },
  {
    name: 'RouterSet',
    scope: '多目标（host 分桶）',
    desc: '按 host 自动分桶，每桶一棵树；NormalizeAssetsDetailed 返回带 host 的多目标资产键。',
  },
  {
    name: 'ProjectManager',
    scope: '多项目隔离',
    desc: '按测试项目管理多个 RouterSet，同 host 项目间互不污染；三层便捷入口对称，项目级直达归一化。',
  },
]

/** 快速开始代码（带简单高亮标记） */
export const QUICKSTART_CODE = `package main

import (
    "fmt"

    "github.com/cyberspacesec/reverse-router-tree-skills/pkg/exporter"
    "github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
)

func main() {
    r := router.NewReverseRouter()

    // 1. 批量喂入抓包流量（curl 命令），单条坏样本不中断整批
    result := r.ReverseCurls(curlCommands)
    fmt.Println(result.Processed, result.Failed)

    // 2. 导出 OpenAPI 3.0.3 —— 黑盒版 Swagger
    exp := exporter.NewOpenAPIExporter()
    doc, _ := exp.Export(r.Tree)

    // 3. URL 资产归一化：资产清单 + 新 URL 直达归一化
    for _, a := range r.ListAssets() {
        fmt.Println(a.AssetKey())   // GET /api/users/{users_id}
    }
    route, ok := r.NormalizeURLString(
        "http://target/api/users/999", "GET")
    fmt.Println(route.Template, ok) // /api/users/{users_id} true
}`
