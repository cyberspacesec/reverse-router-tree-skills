import { Space, Typography } from 'antd'
import { GithubOutlined } from '@ant-design/icons'
import { DOCS_URL, REPO_URL } from '../content'

export default function SiteFooter() {
  return (
    <footer
      style={{
        padding: '40px 24px',
        textAlign: 'center',
        background: '#14181f',
        color: 'rgba(255,255,255,0.65)',
      }}
    >
      <Space direction="vertical" size={8}>
        <Space size={24}>
          <a href={REPO_URL} target="_blank" rel="noreferrer" style={{ color: 'rgba(255,255,255,0.85)' }}>
            <GithubOutlined /> GitHub 仓库
          </a>
          <a href={DOCS_URL} style={{ color: 'rgba(255,255,255,0.85)' }}>
            教学文档
          </a>
          <a href={`${REPO_URL}/releases`} target="_blank" rel="noreferrer" style={{ color: 'rgba(255,255,255,0.85)' }}>
            Releases
          </a>
        </Space>
        <Typography.Text style={{ color: 'rgba(255,255,255,0.45)', fontSize: 13 }}>
          reverse-router-tree-skills · MIT License · 从黑盒流量还原 Web 路由树
        </Typography.Text>
      </Space>
    </footer>
  )
}
