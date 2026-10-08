import { FEATURE_GROUPS, METRICS } from '../content'
import SectionTitle from './SectionTitle'

export default function Features() {
  return (
    <section className="section section-alt" id="features">
      <div className="wrap">
        <SectionTitle
          eyebrow="CAPABILITIES"
          title="核心能力"
          subtitle="四步闭环，缺一不可：先把散落的 URL 还原成结构，给结构装上语义，把语义沉淀成资产，再把资产送进生产。"
        />
        {FEATURE_GROUPS.map((g, gi) => (
          <div className="feature-block" key={g.key}>
            <div className="group-head">
              <span className="idx">0{gi + 1}</span>
              <h3>{g.title}</h3>
            </div>
            <p className="group-sub">{g.subtitle}</p>
            <div className="grid-2">
              {g.items.map((f) => (
                <div className="card" key={f.title}>
                  <h3>
                    <span className="diamond">◆</span>
                    {f.title}
                  </h3>
                  <p>{f.desc}</p>
                </div>
              ))}
            </div>
          </div>
        ))}

        <div className="metrics">
          {METRICS.map((m) => (
            <div className="metric" key={m.label}>
              <b>{m.value}</b>
              <span>{m.label}</span>
            </div>
          ))}
        </div>
        <p className="metrics-note">覆盖率与测试数来自路由包持续测试基线，详见仓库 CI。</p>
      </div>
    </section>
  )
}
