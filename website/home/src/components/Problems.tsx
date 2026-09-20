import { Card, Col, Row, Typography } from 'antd'
import { BugOutlined, CloudServerOutlined, GlobalOutlined } from '@ant-design/icons'
import { NORMALIZE_DEMO, PAIN_POINTS } from '../content'

const ICONS = [BugOutlined, CloudServerOutlined, GlobalOutlined]

/** 标题区（各 section 复用） */
export function SectionTitle({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <div style={{ textAlign: 'center', marginBottom: 48 }}>
      <Typography.Title level={2} style={{ marginBottom: 12 }}>
        {title}
      </Typography.Title>
      <Typography.Paragraph style={{ fontSize: 16, color: '#666', maxWidth: 760, margin: '0 auto' }}>
        {subtitle}
      </Typography.Paragraph>
    </div>
  )
}

export default function Problems() {
  return (
    <section id="problems" style={{ padding: '88px 24px', background: '#fafafa' }}>
      <div style={{ maxWidth: 1120, margin: '0 auto' }}>
        <SectionTitle
          title="解决什么问题"
          subtitle="散乱的 URL 不是资产，还原成真实路由结构才是。三大典型场景，同一个根因：没人知道「这两条 URL 是不是同一个接口」。"
        />
        <Row gutter={[24, 24]}>
          {PAIN_POINTS.map((p, i) => {
            const Icon = ICONS[i % ICONS.length]
            return (
              <Col xs={24} md={8} key={p.title}>
                <Card hoverable style={{ height: '100%' }}>
                  <Icon style={{ fontSize: 34, color: '#3aa676', marginBottom: 16 }} />
                  <Typography.Title level={4} style={{ marginTop: 0 }}>
                    {p.title}
                  </Typography.Title>
                  <Typography.Paragraph style={{ color: '#555' }}>{p.desc}</Typography.Paragraph>
                </Card>
              </Col>
            )
          })}
        </Row>

        {/* 输入 → 输出 对照 */}
        <Card style={{ marginTop: 40 }} styles={{ body: { padding: '28px 32px' } }}>
          <Typography.Title level={4} style={{ marginTop: 0, textAlign: 'center' }}>
            本项目做的事
          </Typography.Title>
          <div style={{ maxWidth: 720, margin: '0 auto' }}>
            {NORMALIZE_DEMO.map((row) => (
              <div
                key={row.raw + row.note}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '9px 0',
                  borderBottom: '1px dashed #eee',
                  flexWrap: 'wrap',
                  fontFamily: 'SFMono-Regular, Consolas, monospace',
                  fontSize: 13.5,
                }}
              >
                <span style={{ color: '#d46b08', flex: '1 1 260px' }}>{row.raw}</span>
                <span style={{ color: '#3aa676' }}>──▶</span>
                <span style={{ color: '#1677ff', flex: '1 1 260px' }}>{row.result}</span>
                <span style={{ color: '#999', flex: '1 1 160px', fontFamily: 'inherit', fontSize: 12.5 }}>
                  {row.note}
                </span>
              </div>
            ))}
          </div>
          <Typography.Paragraph style={{ textAlign: 'center', color: '#888', marginTop: 20, marginBottom: 0 }}>
            左边是爬虫/扫描器眼里的一条条 URL，右边是目标服务器真实的路由结构。
          </Typography.Paragraph>
        </Card>
      </div>
    </section>
  )
}
