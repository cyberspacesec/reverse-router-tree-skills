import { Button, Card, Col, Row, Space, Typography } from 'antd'
import { BookOutlined, CodeOutlined } from '@ant-design/icons'
import { DOCS_URL, QUICKSTART_CODE, REPO_URL } from '../content'
import { SectionTitle } from './Problems'

/** 极简 Go 语法高亮：关键字 / 字符串 / 注释 着色 */
function highlight(code: string) {
  const kw = /^(package|import|func|for|range|return|var|const|if|else)$/
  return code.split('\n').map((line, li) => (
    <div key={li}>
      {line.split(/(\s+)/).map((tok, ti) => {
        if (kw.test(tok)) return <span key={ti} className="kw">{tok}</span>
        if (tok.startsWith('//')) return <span key={ti} className="cmt">{line.slice(line.indexOf('//'))}</span>
        if (/^".*"$/.test(tok)) return <span key={ti} className="str">{tok}</span>
        if (/^\d+$/.test(tok)) return <span key={ti} className="num">{tok}</span>
        return <span key={ti}>{tok}</span>
      })}
      {'\n'}
    </div>
  ))
}

const STEPS = [
  {
    title: '1 · 安装',
    code: 'go get github.com/cyberspacesec/reverse-router-tree-skills',
  },
  {
    title: '2 · 喂数据 → 拿路由树 → 导出规范',
    code: null,
  },
]

export default function QuickStart() {
  return (
    <section id="quickstart" style={{ padding: '88px 24px' }}>
      <div style={{ maxWidth: 960, margin: '0 auto' }}>
        <SectionTitle
          title="快速开始"
          subtitle="三步接入：批量喂入抓包流量，拿到还原好的路由树，导出 OpenAPI 3.0.3 或归一化资产清单。"
        />
        <Space direction="vertical" size={16} style={{ display: 'flex' }}>
          <Card size="small">
            <Typography.Text strong>{STEPS[0].title}</Typography.Text>
            <pre className="code-block" style={{ marginTop: 12 }}>
              <span className="fn">go get</span> github.com/cyberspacesec/reverse-router-tree-skills
            </pre>
          </Card>

          <Card size="small">
            <Typography.Text strong>{STEPS[1].title}</Typography.Text>
            <pre className="code-block" style={{ marginTop: 12 }}>
              {highlight(QUICKSTART_CODE)}
            </pre>
          </Card>

          <Card size="small">
            <Typography.Text strong>3 · 或者直接跑示例 CLI</Typography.Text>
            <pre className="code-block" style={{ marginTop: 12 }}>
              <span className="cmt"># 仓库内置 quickstart 演示：喂数据 → 路由树 → OpenAPI → 资产归一化</span>
              {'\n'}
              <span className="fn">go run</span> ./examples/quickstart
            </pre>
          </Card>
        </Space>

        <Row gutter={16} justify="center" style={{ marginTop: 40 }}>
          <Col>
            <Button type="primary" size="large" icon={<BookOutlined />} href={DOCS_URL}>
              阅读教学文档
            </Button>
          </Col>
          <Col>
            <Button size="large" icon={<CodeOutlined />} href={`${REPO_URL}/tree/main/examples/quickstart`} target="_blank">
              查看示例源码
            </Button>
          </Col>
        </Row>
      </div>
    </section>
  )
}
