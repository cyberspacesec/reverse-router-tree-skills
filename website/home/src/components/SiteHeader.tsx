import { DOCS_URL, NAV_ITEMS, REPO_URL } from '../content'

/** 滚动到锚点 section（站点为单页无路由，锚点导航即可） */
function scrollTo(key: string) {
  document.getElementById(key)?.scrollIntoView({ behavior: 'smooth' })
}

export default function SiteHeader() {
  return (
    <header className="site-header">
      <div className="brand" onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}>
        <span className="dot" />
        reverse-<b>router</b>-tree
      </div>
      <nav className="nav-links">
        {NAV_ITEMS.map((it) => (
          <button
            key={it.key}
            onClick={() => (it.href ? (window.location.href = it.href) : scrollTo(it.key))}
          >
            {it.label}
          </button>
        ))}
      </nav>
      <div className="header-actions">
        <a className="btn btn-ghost nav-docs" href={DOCS_URL}>
          教学文档
        </a>
        <a className="btn btn-primary" href={REPO_URL} target="_blank" rel="noreferrer">
          GitHub
        </a>
      </div>
    </header>
  )
}
