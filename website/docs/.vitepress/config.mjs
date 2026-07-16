import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

// 仓库为 cyberspacesec/reverse-router-tree-skills，部署到 GitHub Pages 项目站点
// 访问地址：https://cyberspacesec.github.io/reverse-router-tree-skills/
const base = '/reverse-router-tree-skills/'

export default withMermaid(defineConfig({
  lang: 'zh-CN',
  title: 'reverse-router-tree-skills',
  description: '从黑盒流量逆向还原 Web 路由树 —— 网络空间测绘教学站',
  base,
  cleanDist: true,

  head: [
    ['meta', { name: 'theme-color', content: '#3aa676' }]
  ],

  themeConfig: {
    logo: '/logo.svg',

    nav: [
      { text: '指南', link: '/guide/what-is-this', activeMatch: '/guide/' },
      { text: '架构', link: '/architecture/overview', activeMatch: '/architecture/' },
      { text: '功能原理', link: '/features/path-variable', activeMatch: '/features/' },
      { text: '项目源码', link: 'https://github.com/cyberspacesec/reverse-router-tree-skills' }
    ],

    sidebar: {
      '/guide/': [
        {
          text: '入门',
          collapsed: false,
          items: [
            { text: '这是什么', link: '/guide/what-is-this' },
            { text: '为什么重要', link: '/guide/why-important' },
            { text: '它能做什么', link: '/guide/capabilities' },
            { text: '快速上手', link: '/guide/quick-start' },
            { text: '一个完整示例', link: '/guide/full-example' },
            { text: '从抓包到路由树', link: '/guide/packet-to-tree' }
          ]
        }
      ],
      '/architecture/': [
        {
          text: '架构设计',
          collapsed: false,
          items: [
            { text: '整体架构', link: '/architecture/overview' },
            { text: '分层与数据流', link: '/architecture/data-flow' },
            { text: '路由树结构', link: '/architecture/tree-structure' },
            { text: '节点类型体系', link: '/architecture/node-types' },
            { text: '类型推断体系', link: '/architecture/type-inference' },
            { text: '并发设计', link: '/architecture/concurrency' }
          ]
        }
      ],
      '/features/': [
        {
          text: '核心算法',
          collapsed: false,
          items: [
            { text: '9 步逆向流程', link: '/features/reverse-flow' },
            { text: 'IsNeedRequest 去重', link: '/features/is-need-request' }
          ]
        },
        {
          text: '功能点原理',
          collapsed: false,
          items: [
            { text: '路径变量识别', link: '/features/path-variable' },
            { text: '选择性合并策略', link: '/features/selective-merge' },
            { text: '前缀/后缀合并', link: '/features/prefix-suffix-merge' },
            { text: '相似串合并突破', link: '/features/similar-strings' },
            { text: '自定义合并规则', link: '/features/custom-merge-rule' },
            { text: '查询参数处理', link: '/features/query-params' },
            { text: '请求体解析', link: '/features/body-parser' },
            { text: 'Header 路由', link: '/features/header-routing' },
            { text: 'Cookie 路由', link: '/features/cookie-routing' },
            { text: '必需参数推断', link: '/features/required-params' },
            { text: '中国特有格式', link: '/features/china-formats' },
            { text: '长数字串降级', link: '/features/long-number' },
            { text: '路径边界条件', link: '/features/path-edge-cases' }
          ]
        },
        {
          text: '输出与可观测',
          collapsed: false,
          items: [
            { text: '路由树序列化', link: '/features/serialization' },
            { text: 'OpenAPI 导出', link: '/features/openapi-export' },
            { text: '日志与统计', link: '/features/observability' }
          ]
        }
      ]
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/cyberspacesec/reverse-router-tree-skills' }
    ],

    outline: {
      level: [2, 3],
      label: '本页导航'
    },

    docFooter: {
      prev: '上一页',
      next: '下一页'
    },

    lastUpdated: {
      text: '最后更新于',
      formatOptions: {
        dateStyle: 'yyyy-MM-dd'
      }
    },

    search: {
      provider: 'local',
      options: {
        translations: {
          button: { buttonText: '搜索文档', buttonTitle: '搜索', buttonAriaLabel: '搜索' },
          modal: {
            displayDetails: '显示详情',
            resetButtonTitle: '清除',
            backButtonTitle: '返回',
            noResultsText: '没有结果',
            startScreen: { recentSearchesTitle: '最近搜索', noRecentSearchesText: '无' },
            footer: {
              selectText: '选择',
              navigateText: '切换',
              closeText: '关闭',
              navigateUpKey: '↑',
              navigateDownKey: '↓',
              closeKey: 'esc'
            }
          }
        }
      }
    },

    footer: {
      message: '基于 MIT 协议发布',
      copyright: 'Copyright © 2026 cyberspacesec'
    }
  },

  markdown: {
    lineNumbers: false,
    theme: { light: 'github-light', dark: 'github-dark' }
  }
}))
