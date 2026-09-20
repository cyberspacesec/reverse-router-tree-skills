import { Card, Col, Row, Typography } from 'antd'
import { FEATURES, METRICS } from '../content'
import { SectionTitle } from './Problems'
import { Statistic } from 'antd'

export default function Features() {
  return (
    <section id="features" style={{ padding: '88px 24px' }}>
      <div style={{ maxWidth: 1120, margin: '0 auto' }}>
        <SectionTitle
          title="核心能力"
          subtitle="从流量还原、类型推断到资产归一化、生产护栏——一个纯 Go 标准库实现的完整链路。"
        />
        <Row gutter={[20, 20]}>
          {FEATURES.map((f) => (
            <Col xs={24} md={12} lg={6} key={f.title}>
              <Card hoverable style={{ height: '100%' }} styles={{ body: { padding: '22px 20px' } }}>
                <Typography.Title level={5} style={{ marginTop: 0, marginBottom: 10, fontSize: 16 }}>
                  <span style={{ color: '#3aa676', marginRight: 8 }}>◆</span>
                  {f.title}
                </Typography.Title>
                <Typography.Paragraph style={{ color: '#555', fontSize: 13.5, marginBottom: 0 }}>
                  {f.desc}
                </Typography.Paragraph>
              </Card>
            </Col>
          ))}
        </Row>

        {/* 指标 */}
        <Row gutter={[24, 24]} style={{ marginTop: 56 }} justify="center">
          {METRICS.map((m) => (
            <Col xs={12} md={6} key={m.label}>
              <div style={{ textAlign: 'center' }}>
                <Statistic value={m.value} valueStyle={{ color: '#3aa676', fontSize: 34, fontWeight: 700 }} />
                <div style={{ color: '#888', marginTop: 6 }}>{m.label}</div>
              </div>
            </Col>
          ))}
        </Row>
      </div>
    </section>
  )
}
