import { Button, Menu, Space } from 'antd'
import { GithubOutlined, ReadOutlined } from '@ant-design/icons'
import { DOCS_URL, NAV_ITEMS, REPO_URL } from '../content'

/** 滚动到锚点 section（站点为单页无路由，锚点导航即可） */
function scrollTo(key: string) {
  document.getElementById(key)?.scrollIntoView({ behavior: 'smooth' })
}

export default function SiteHeader() {
  return (
    <header
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        zIndex: 100,
        display: 'flex',
        alignItems: 'center',
        padding: '0 32px',
        height: 64,
        background: 'rgba(255,255,255,0.92)',
        backdropFilter: 'blur(8px)',
        borderBottom: '1px solid #f0f0f0',
      }}
    >
      <div
        style={{
          fontWeight: 700,
          fontSize: 17,
          cursor: 'pointer',
          whiteSpace: 'nowrap',
        }}
        onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
      >
        reverse-router-tree<span style={{ color: '#3aa676' }}>-skills</span>
      </div>
      <Menu
        mode="horizontal"
        style={{ flex: 1, minWidth: 0, justifyContent: 'center', borderBottom: 'none', background: 'transparent' }}
        items={NAV_ITEMS.map((it) => ({ key: it.key, label: it.label }))}
        onClick={({ key }) => {
          const item = NAV_ITEMS.find((n) => n.key === key)
          if (item?.href) {
            window.location.href = item.href
          } else {
            scrollTo(key)
          }
        }}
      />
      <Space>
        <Button icon={<ReadOutlined />} href={DOCS_URL} target="_self">
          教学文档
        </Button>
        <Button type="primary" icon={<GithubOutlined />} href={REPO_URL} target="_blank">
          GitHub
        </Button>
      </Space>
    </header>
  )
}
