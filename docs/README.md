# reverse-router-tree-skills 项目文档

本目录存放项目的设计文档、分析结论和知识库。

## 📚 在线文档站

项目有一个基于 VitePress 的教学文档站，内容更系统、配图更丰富：

- **源码**：[`website/`](../website/) 目录
- **本地预览**：`cd website && npm install && npm run dev`
- **部署**：推送到 `main` 后由 [.github/workflows/deploy-docs.yml](../.github/workflows/deploy-docs.yml) 自动构建并部署到 GitHub Pages

## 文档索引

> 以下为本目录内的设计文档（文档站的早期素材，内容已被文档站吸收整合）。

- [项目初衷与核心问题](01-project-purpose.md) — 项目要解决什么问题，为什么重要
- [当前实现状态](02-current-status.md) — 各模块完成度、编译状态、已知问题
- [架构设计](03-architecture.md) — 项目整体架构和模块关系
- [待修复问题清单](04-issues-to-fix.md) — 编译错误、bug、设计缺陷
- [核心算法设计](05-core-algorithm.md) — ReverseHttpRequest 的算法思路和实现规划
