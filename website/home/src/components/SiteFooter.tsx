import { DOCS_URL, REPO_URL } from '../content'

export default function SiteFooter() {
  return (
    <footer className="site-footer">
      <nav>
        <a href={REPO_URL} target="_blank" rel="noreferrer">GitHub 仓库</a>
        <a href={DOCS_URL}>教学文档</a>
        <a href={`${REPO_URL}/releases`} target="_blank" rel="noreferrer">Releases</a>
      </nav>
      <div>reverse-router-tree-skills · MIT License · 从黑盒流量还原真实路由结构，归一化 URL 资产</div>
    </footer>
  )
}
