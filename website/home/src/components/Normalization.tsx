import { API_GROUPS, API_LAYERS, ASSET_SCENARIOS } from '../content'
import SectionTitle from './SectionTitle'

export default function Normalization() {
  return (
    <section className="section" id="normalize">
      <div className="wrap">
        <SectionTitle
          eyebrow="THE DELIVERABLE"
          title="URL 资产归一化 API"
          subtitle="归一化是本项目的核心输出：把同一接口的不同 URL 收成一条稳定的路由资产，支撑测绘 URL 资产的去重、聚合与检索。"
        />

        <div className="grid-3">
          {ASSET_SCENARIOS.map((s) => (
            <div className="card" key={s.title}>
              <div className="mark">{s.title}</div>
              <div className="mono-line">{s.how}</div>
              <p>{s.desc}</p>
            </div>
          ))}
        </div>

        <div className="grid-3" style={{ marginTop: 18 }}>
          {API_GROUPS.map((g) => (
            <div className="card" key={g.q}>
              <div className="mark">{g.q}</div>
              <div className="mono-line">{g.apis}</div>
              <p>{g.d}</p>
            </div>
          ))}
        </div>

        <div className="panel">
          <h3>三层 API，入口对称</h3>
          <table className="api-table">
            <thead>
              <tr>
                <th>API 层</th>
                <th>适用场景</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              {API_LAYERS.map((l) => (
                <tr key={l.name}>
                  <td><code>{l.name}</code></td>
                  <td>{l.scope}</td>
                  <td>{l.desc}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="hint">
            归一化是只读操作：未知 host / 项目只返回失败原因，不懒建空桶，不污染资产清单与配额。
          </p>
        </div>
      </div>
    </section>
  )
}
