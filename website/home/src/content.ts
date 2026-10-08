// 官网静态内容：文案与演示数据集中在这里，改文案不动组件。
//
// 叙事主线（全站统一的"一句话价值主张"）：
//   网络空间测绘中，抓到的 URL 是按"值"散开的，不是按"接口"组织的——
//   /api/users/123 与 /api/users/456 是同一个接口的两个分身。
//   本引擎做的事：把散落 URL 反向推理成目标服务器真实的路由树，
//   再归一化成一条条稳定的路由资产（方法 + 路径模板）。
//   路由树是中间产物，归一化资产是交付物，OpenAPI 导出是附赠能力。

export const REPO_URL = 'https://github.com/cyberspacesec/reverse-router-tree-skills'
export const DOCS_URL = '/reverse-router-tree-skills/docs/'

/** 导航锚点 */
export const NAV_ITEMS = [
  { key: 'case', label: '扫一眼就懂' },
  { key: 'problems', label: '解决什么问题' },
  { key: 'features', label: '核心能力' },
  { key: 'normalize', label: 'URL 归一化' },
  { key: 'quickstart', label: '快速开始' },
  { key: 'docs', label: '教学文档', href: DOCS_URL },
]

/** Hero 区文案 */
export const HERO = {
  tag: '网络空间测绘 · URL 资产归一化引擎',
  titleTop: '散落的 URL，',
  titleHighlight: '还原成真实的接口',
  subtitle:
    '爬虫抓到的 /api/users/123、/api/users/456、/api/users/789，是同一个接口的三个分身。' +
    '本引擎把它们反向推理成目标服务器真实的路由结构，归一化成一条稳定资产 GET /api/users/{users_id}' +
    '—— 不靠字符串相似度硬凑，靠还原路由树认出"它们本来就是同一个接口"。',
}

/** 痛点：每个场景带量化损失，回答"这个项目解决什么问题" */
export interface PainPoint {
  title: string
  loss: string
  desc: string
}

export const PAIN_POINTS: PainPoint[] = [
  {
    title: '爬虫越爬越多，接口全貌越来越糊',
    loss: '1 个接口，被当成 300 个 URL 请求',
    desc: '抓包是按"值"散开的：/api/users/123、/api/users/456…… 同一接口的不同取值被一条条当成独立 URL 收藏，几十万条里看不出目标应用到底有几个接口。',
  },
  {
    title: '扫描器在同一接口上重复测试',
    loss: '1000 条 URL，真实接口也许只有 700',
    desc: '识别不出"这两条 URL 是同一个接口"，测试项成倍膨胀，配额和 QPS 浪费在重复路径上；更糟的是，真正没测过的路径反而被淹没、漏掉。',
  },
  {
    title: '测绘平台 URL 资产散乱、无法聚合',
    loss: '同源接口在资产库散落成上千碎片',
    desc: '资产去重靠字符串相似度，粒度对不齐，检索与资产清点无从下手——"这个目标应用到底有哪些接口、覆盖了多少"永远答不上来。',
  },
]

/** "扫一眼就懂"的还原 case：同一批真实流量 → 还原后资产清单 */
export interface BeforeAfterCase {
  feed: string[]
  output: string[]
  note: string
}

export const BEFORE_AFTER_CASES: BeforeAfterCase[] = [
  {
    feed: [
      'GET /api/users/123',
      'GET /api/users/456',
      'GET /api/users/789',
      'GET /api/users/profile',
      'GET /api/posts/42?author=7',
      'GET /api/posts/43?author=7&author=8',
    ],
    output: [
      'GET /api/users/{users_id}',
      'GET /api/users/profile',
      'GET /api/posts/{posts_id}',
      '?author · 可多值',
    ],
    note: '6 条散落 URL → 4 条资产。users_id 是 integer，author 被推断为可多值参数。',
  },
  {
    feed: [
      'POST /api/orders {"product_id":123,"qty":2}',
      'POST /api/orders {"product_id":456,"qty":1}',
      'POST /api/orders {"product_id":789,"qty":2,"coupon":"SAVE10"}',
    ],
    output: [
      'POST /api/orders',
      'body: product_id, qty',
      'coupon → 可选字段',
    ],
    note: '3 条请求体差异 → 1 条资产 + 字段必需性推断。',
  },
]

/** 输入 → 输出 对照演示：左原始 URL → 右归一化资产 */
export interface NormalizeDemoRow {
  raw: string
  result: string
  note: string
  /** 视觉类别：合并成模板 / 不误合并（固定词） / 查询参数 / 请求体 */
  kind?: 'merge' | 'fixed' | 'query' | 'body'
}

export const NORMALIZE_DEMO: NormalizeDemoRow[] = [
  { raw: '/api/users/123', result: '/api/users/{users_id}', note: '纯数字 → 变量 (integer)', kind: 'merge' },
  { raw: '/api/users/456', result: '/api/users/{users_id}', note: '归入同一模板', kind: 'merge' },
  { raw: '/api/users/789', result: '/api/users/{users_id}', note: '归入同一模板', kind: 'merge' },
  { raw: '/api/users/profile', result: '(独立固定路径)', note: '字母固定词 → 不误合并', kind: 'fixed' },
  {
    raw: '/api/users?page=1&size=20',
    result: '/api/users?page & size',
    note: '查询参数还原 + 必需性推断',
    kind: 'query',
  },
  {
    raw: 'POST /api/users (json)',
    result: 'requestBody: name, age',
    note: '请求体参数解析',
    kind: 'body',
  },
]

/** 核心能力：4 组 × 2 卡片，每组回答一个环节的问题 */
export interface FeatureItem {
  title: string
  desc: string
}

export interface FeatureGroup {
  key: string
  title: string
  subtitle: string
  items: FeatureItem[]
}

export const FEATURE_GROUPS: FeatureGroup[] = [
  {
    key: 'restore',
    title: '从散乱到结构',
    subtitle: '输入是散落的 URL，输出是一棵像 Spring 路由映射表一样的树。',
    items: [
      {
        title: '路径变量识别',
        desc: '纯数字 / UUID / 手机号 / 身份证号 / 银行卡号 / 车牌号 / 前缀后缀模式自动合并为 {var}，只合并匹配模式的兄弟节点；list、create 等固定路径不误合并。',
      },
      {
        title: '查询参数与请求体还原',
        desc: 'JSON / 表单 / multipart 请求体解析，JSON 嵌套点号扁平化，参数名大小写不敏感；基于出现频率推断必需参数，阈值可配。',
      },
    ],
  },
  {
    key: 'semantics',
    title: '结构带着语义',
    subtitle: '节点不只是路径片段，还知道自己是什么——这才谈得上"理解"接口。',
    items: [
      {
        title: '两层类型推断',
        desc: '物理类型（integer / string / ...）+ 逻辑类型（uuid / phone / idcard / ...），变量不只是占位符，还知道它是什么，后续 fuzz 与参数生成有据可依。',
      },
      {
        title: '多维度路由还原',
        desc: 'Content-Type / Header（Accept 等）/ Cookie 是独立的子路由维度，同一路径不同维度是不同的接口，不会被错误并成同一条模板。',
      },
    ],
  },
  {
    key: 'asset',
    title: '资产可运营',
    subtitle: '结构沉淀成可查询、可聚合、可去重的资产——这是本项目的核心交付物。',
    items: [
      {
        title: 'URL 资产归一化',
        desc: '每条 URL 收敛为「方法 + 路径模板」稳定资产键；归一化失败给 5 种机器可读原因；ListAssets 全量枚举已知资产，不靠字符串相似度去重。',
      },
      {
        title: '多目标 Host 隔离',
        desc: 'RouterSet 按 host 分桶，每个目标应用独立还原一棵树，跨目标流量不会混成"缝合怪"；HostAssetKey 支撑跨目标资产清单。',
      },
    ],
  },
  {
    key: 'production',
    title: '生产可落地',
    subtitle: '不为玩具：多项目隔离、分批续喂、资源护栏、规范导出，缺一不可。',
    items: [
      {
        title: '项目级隔离与生产护栏',
        desc: 'ProjectManager 按安全测试项目管理多个 RouterSet，同 host 项目间互不污染；资源上限、敏感参数脱敏、fail-soft 默认开启；JSON 序列化保留样本计数，分批采集可续喂。',
      },
      {
        title: 'OpenAPI 3.0.3 导出',
        desc: '路径 / 参数 / 请求体 / 安全方案（从 Authorization 推断 Bearer / Basic / Digest），黑盒流量直接产出"Swagger"——一份结构复用，多种消费方式。',
      },
    ],
  },
]

/** 性能 / 质量指标 */
export const METRICS = [
  { value: '146万+', label: '单核 URL 处理吞吐 /秒' },
  { value: '92%', label: '路由包测试覆盖率' },
  { value: '532', label: '自动化测试（含 -race）' },
  { value: '0', label: '外部依赖（纯 Go 标准库）' },
]

/** 归一化资产拿到手之后能干什么：三个运营场景 */
export interface AssetScenario {
  title: string
  how: string
  desc: string
}

export const ASSET_SCENARIOS: AssetScenario[] = [
  {
    title: '资产去重',
    how: '/api/users/123 + /api/users/456 → 1 条资产',
    desc: '同一接口的不同 URL 收敛到同一条「方法 + 路径模板」，资产库里不再有"长得不一样、其实是同一个"的碎片，清点数量即真实接口数。',
  },
  {
    title: '资产检索',
    how: 'ListAssets 全量枚举 · NormalizeURLString 单条直达',
    desc: '想知道"这个目标应用有哪些接口"，一条 ListAssets 全量枚举、稳定排序；手里新抓到一条 URL，想知道它归到哪个已知接口，单条归一化直达模板。',
  },
  {
    title: '聚合统计',
    how: '按 host / 项目分组 · Health / Stats 聚合',
    desc: '资产按 host、按项目分组聚合，配合统计指标与健康报告，能回答"每个目标应用到底有多少接口、已覆盖哪些、还有哪些没测"。',
  },
]

/** 归一化 API 三个问题三组能力 */
export const API_GROUPS = [
  {
    q: '这条流量归到哪？',
    apis: 'NormalizeURL / NormalizeCurl / NormalizeURLString',
    d: '单条直达归一化，返回方法 + 路径模板资产键。',
  },
  {
    q: '归不上卡在哪？',
    apis: 'NormalizeURLDetailed / NormalizeReport',
    d: '5 种机器可读失败原因（unknown_path / unknown_method / unknown_host / unknown_project / invalid_request），批量明细替代静默丢弃。',
  },
  {
    q: '树里有哪些资产？',
    apis: 'ListAssets / ProjectAssets',
    d: '遍历树枚举全部已知资产，稳定排序输出，按 host / 项目分组，不用逐条请求试探。',
  },
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
