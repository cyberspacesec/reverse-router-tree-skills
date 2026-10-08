import { DOCS_URL, QUICKSTART_CODE, REPO_URL } from '../content'
import SectionTitle from './SectionTitle'

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

export default function QuickStart() {
  return (
    <section className="section section-alt" id="quickstart">
      <div className="wrap" style={{ maxWidth: 960 }}>
        <SectionTitle
          eyebrow="GET STARTED"
          title="快速开始"
          subtitle="三步接入：批量喂入抓包流量，拿到还原好的路由树与归一化资产清单，按需导出 OpenAPI 3.0.3。"
        />
        <div className="qs-card">
          <h3>1 · 安装</h3>
          <pre className="code-block">
            <span className="fn">go get</span> github.com/cyberspacesec/reverse-router-tree-skills
          </pre>
        </div>
        <div className="qs-card">
          <h3>2 · 喂数据 → 拿路由树 → 导出规范</h3>
          <pre className="code-block">{highlight(QUICKSTART_CODE)}</pre>
        </div>
        <div className="qs-card">
          <h3>3 · 或者直接跑示例 CLI</h3>
          <pre className="code-block">
            <span className="cmt"># 仓库内置 quickstart 演示：喂数据 → 路由树 → OpenAPI → 资产归一化</span>
            {'\n'}
            <span className="fn">go run</span> ./examples/quickstart
          </pre>
        </div>
        <div className="qs-actions">
          <a className="btn btn-primary btn-lg" href={DOCS_URL}>阅读教学文档</a>
          <a className="btn btn-ghost btn-lg" href={`${REPO_URL}/tree/main/examples/quickstart`} target="_blank" rel="noreferrer">
            查看示例源码
          </a>
        </div>
      </div>
    </section>
  )
}
