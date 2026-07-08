---
layout: home

hero:
  name: reverse-router-tree
  text: 从黑盒流量逆向还原 Web 路由树
  tagline: 纯黑盒 · 自动识别路径变量与参数模式 · 导出 OpenAPI · 一个网络空间测绘教学项目
  actions:
    - theme: brand
      text: 从这里开始 →
      link: /guide/what-is-this
    - theme: alt
      text: 查看架构
      link: /architecture/overview
    - theme: alt
      text: GitHub
      link: https://github.com/cyberspacesec/reverse-router-tree-skills

features:
  - title: 路径变量识别
    details: /api/users/123 与 /api/users/456 自动合并为 /api/users/{id}，整数、UUID、手机号等模式智能识别。
    icon: 🔀
  - title: 选择性合并
    details: 不会把 list/create 这类固定路径误合并进变量，只合并真正符合模式的子集。
    icon: 🎯
  - title: 中国特有格式
    details: 手机号、座机号、身份证号、银行卡号、车牌号自动识别为带语义的逻辑类型。
    icon: 🇨🇳
  - title: 多维度路由
    details: 不仅是路径，查询参数、Content-Type、Header、Cookie 都是路由维度，全部纳入树结构。
    icon: 🌳
  - title: 类型推断
    details: 物理类型（integer/float/string）+ 逻辑类型（phone/email/uuid/date），两层推断协同。
    icon: 🔍
  - title: OpenAPI 导出
    details: 路由树一键导出为 OpenAPI 3.0.3，Swagger UI / Redoc 直接渲染。
    icon: 📄
  - title: 结构化日志 + 统计
    details: 五级 slog 日志 + 11 项 atomic 计数指标，黑盒过程可追踪、效果可量化。
    icon: 📊
  - title: 纯黑盒
    details: 无需目标主动暴露接口文档，从抓包流量即可还原路由结构。
    icon: 🕵️
---
