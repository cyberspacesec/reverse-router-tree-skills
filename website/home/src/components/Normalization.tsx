import { Card, Col, Row, Table, Tag, Typography } from 'antd'
import { API_LAYERS } from '../content'
import { SectionTitle } from './Problems'

/** 归一化 API 三个问题三组能力 */
const API_GROUPS = [
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

const LAYER_COLUMNS = [
  {
    title: 'API 层',
    dataIndex: 'name',
    width: 180,
    render: (v: string) => <Typography.Text code strong>{v}</Typography.Text>,
  },
  { title: '适用场景', dataIndex: 'scope', width: 180 },
  { title: '说明', dataIndex: 'desc' },
]

export default function Normalization() {
  return (
    <section id="normalize" style={{ padding: '88px 24px', background: '#fafafa' }}>
      <div style={{ maxWidth: 1120, margin: '0 auto' }}>
        <SectionTitle
          title="URL 资产归一化 API"
          subtitle="归一化是本项目的核心输出：把同一接口的不同 URL 收成一条稳定的路由资产，支撑测绘 URL 资产的去重、聚合与检索。"
        />
        <Row gutter={[20, 20]}>
          {API_GROUPS.map((g) => (
            <Col xs={24} md={8} key={g.q}>
              <Card hoverable style={{ height: '100%' }}>
                <Tag color="green" style={{ fontSize: 13, marginBottom: 12 }}>
                  {g.q}
                </Tag>
                <Typography.Paragraph strong style={{ fontFamily: 'Consolas, monospace', fontSize: 13, marginBottom: 8 }}>
                  {g.apis}
                </Typography.Paragraph>
                <Typography.Paragraph style={{ color: '#555', fontSize: 13.5, marginBottom: 0 }}>
                  {g.d}
                </Typography.Paragraph>
              </Card>
            </Col>
          ))}
        </Row>

        <Card style={{ marginTop: 32 }} styles={{ body: { padding: '24px 28px' } }}>
          <Typography.Title level={4} style={{ marginTop: 0, marginBottom: 18 }}>
            三层 API，入口对称
          </Typography.Title>
          <Table
            dataSource={API_LAYERS}
            columns={LAYER_COLUMNS}
            rowKey="name"
            pagination={false}
            size="middle"
          />
          <Typography.Paragraph style={{ color: '#888', marginTop: 16, marginBottom: 0 }}>
            归一化是只读操作：未知 host / 项目只返回失败原因，不懒建空桶，不污染资产清单与配额。
          </Typography.Paragraph>
        </Card>
      </div>
    </section>
  )
}
