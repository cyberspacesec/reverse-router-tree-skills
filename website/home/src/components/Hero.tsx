import { DOCS_URL, HERO, REPO_URL } from '../content'

/**
 * 首屏右侧的还原示意：手写 SVG，不依赖运行时引擎。
 * 坐标按 520×250 视口手摆，保证窄屏缩放后标签仍不重叠。
 */
function StageTree() {
  const nodes: { x: number; y: number; label: string; fill: string }[] = [
    { x: 260, y: 26, label: '/', fill: '#34d399' },
    { x: 260, y: 84, label: 'api', fill: '#34d399' },
    { x: 150, y: 146, label: 'users', fill: '#34d399' },
    { x: 392, y: 146, label: 'orders', fill: '#34d399' },
    { x: 78, y: 214, label: '{users_id}', fill: '#38bdf8' },
    { x: 222, y: 214, label: 'profile', fill: '#34d399' },
    { x: 392, y: 214, label: '?status*', fill: '#fbbf24' },
  ]
  const edges: [number, number][] = [
    [0, 1],
    [1, 2],
    [1, 3],
    [2, 4],
    [2, 5],
    [3, 6],
  ]
  return (
    <svg className="tree" viewBox="0 0 520 250" role="img" aria-label="还原后的路由树示意">
      {edges.map(([a, b]) => (
        <path
          key={`${a}-${b}`}
          className="edge"
          d={`M${nodes[a].x} ${nodes[a].y + 7} C ${nodes[a].x} ${(nodes[a].y + nodes[b].y) / 2}, ${nodes[b].x} ${(nodes[a].y + nodes[b].y) / 2}, ${nodes[b].x} ${nodes[b].y - 9}`}
        />
      ))}
      {nodes.map((n) => (
        <g key={n.label}>
          <text x={n.x} y={n.y - 12} textAnchor="middle" fill={n.fill === '#fbbf24' ? '#fde68a' : n.fill === '#38bdf8' ? '#7dd3fc' : '#a7f3d0'}>
            {n.label}
          </text>
          <circle cx={n.x} cy={n.y} r={5} fill="#0b141d" stroke={n.fill} strokeWidth={1.8} />
        </g>
      ))}
    </svg>
  )
}

export default function Hero() {
  return (
    <section className="hero">
      <div className="hero-grid">
        <div>
          <div className="kicker">{HERO.tag}</div>
          <h1>
            {HERO.titleTop}
            <br />
            <span className="accent">{HERO.titleHighlight}</span>
          </h1>
          <p className="sub">{HERO.subtitle}</p>
          <div className="hero-cta">
            <button
              className="btn btn-primary btn-lg"
              onClick={() => document.getElementById('quickstart')?.scrollIntoView({ behavior: 'smooth' })}
            >
              快速开始 →
            </button>
            <a className="btn btn-ghost btn-lg" href={DOCS_URL}>
              阅读文档
            </a>
            <a className="btn btn-ghost btn-lg" href={REPO_URL} target="_blank" rel="noreferrer">
              GitHub
            </a>
          </div>
          <div className="hero-trust">
            <span><i />纯 Go · 零外部依赖</span>
            <span><i />OpenAPI 3.0.3 导出</span>
            <span><i />并发安全 · 生产就绪</span>
          </div>
        </div>

        <div className="stage" aria-hidden="true">
          <div className="stage-label"><span className="n">1</span>爬虫眼里的散落 URL</div>
          <div className="url-strip">
            <div>/api/users/<span className="hl">123</span></div>
            <div>/api/users/<span className="hl">456</span></div>
            <div>/api/users/<span className="hl">789</span></div>
            <div>/api/users/profile</div>
            <div>/api/orders?status=paid</div>
          </div>
          <div className="stage-gate">▼ 反向推理还原 ▼</div>
          <div className="stage-label"><span className="n">2</span>还原后的真实路由树</div>
          <StageTree />
          <div className="stage-label" style={{ marginTop: 12 }}>
            <span className="n">3</span>收敛成资产<b>5 条 URL → 3 条资产</b>
          </div>
          <div className="asset-rows">
            <div className="ar"><span className="m">GET</span><span className="t">/api/users/&#123;users_id&#125;</span></div>
            <div className="ar"><span className="m">GET</span><span className="t">/api/users/profile</span></div>
            <div className="ar"><span className="m">GET</span><span className="t">/api/orders</span><span className="q">?status</span></div>
          </div>
        </div>
      </div>
    </section>
  )
}
