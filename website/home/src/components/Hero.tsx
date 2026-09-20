import { Button, Space, Tag } from 'antd'
import { ArrowRightOutlined, GithubOutlined, RocketOutlined } from '@ant-design/icons'
import { DOCS_URL, REPO_URL } from '../content'

export default function Hero() {
  return (
    <section
      style={{
        padding: '160px 24px 96px',
        textAlign: 'center',
        background:
          'linear-gradient(180deg, #f0faf5 0%, #ffffff 78%)',
      }}
    >
      <Space direction="vertical" size={20} style={{ display: 'flex', alignItems: 'center' }}>
        <Tag color="green" style={{ fontSize: 14, padding: '4px 14px' }}>
          网络空间测绘 · URL 资产归一化引擎
        </Tag>
        <h1 style={{ fontSize: 44, margin: 0, lineHeight: 1.25 }}>
          从黑盒抓包流量
          <br />
          还原 Web 应用的<b style={{ color: '#3aa676' }}>真实路由树</b>
        </h1>
        <p style={{ fontSize: 18, color: '#555', maxWidth: 720, margin: 0 }}>
          给一组抓到的 HTTP 请求，还你一棵还原好的路由树——识别路径变量、推断参数类型、
          把同一接口的散乱 URL 归一化为稳定的路由模板，并导出 OpenAPI 3.0.3（黑盒版 Swagger）。
        </p>
        <Space size="middle" style={{ marginTop: 12 }}>
          <Button
            type="primary"
            size="large"
            icon={<RocketOutlined />}
            onClick={() =>
              document.getElementById('quickstart')?.scrollIntoView({ behavior: 'smooth' })
            }
          >
            快速开始
          </Button>
          <Button size="large" href={DOCS_URL}>
            教学文档 <ArrowRightOutlined />
          </Button>
          <Button size="large" icon={<GithubOutlined />} href={REPO_URL} target="_blank">
            GitHub
          </Button>
        </Space>
      </Space>
    </section>
  )
}
