import { BEFORE_AFTER_CASES } from '../content'
import SectionTitle from './SectionTitle'

/** 两个具体 case：喂进去什么 → 还原成什么。让人扫一眼就懂"解决到什么程度"。 */
export default function CaseDemo() {
  return (
    <section className="section section-alt" id="case">
      <div className="wrap">
        <SectionTitle
          eyebrow="BEFORE / AFTER"
          title="同一批流量，还原成什么"
          subtitle="下面不是示意图，是真实跑一遍引擎得到的输入输出。左边是抓到的散落 URL，右边是还原后的稳定资产。"
        />
        <div className="grid-2">
          {BEFORE_AFTER_CASES.map((c, i) => (
            <div className="case-card" key={i}>
              <div className="case-cols">
                <div className="case-col feed">
                  <div className="cap">喂入 · 爬虫眼里</div>
                  {c.feed.map((f) => (
                    <div className="pill" key={f}>{f}</div>
                  ))}
                </div>
                <div className="case-col out">
                  <div className="cap">还原后 · 稳定资产</div>
                  {c.output.map((o) => (
                    <div className="pill" key={o}>{o}</div>
                  ))}
                </div>
              </div>
              <div className="case-note">
                <span className="tag">CASE {i + 1}</span>
                {c.note}
              </div>
            </div>
          ))}
        </div>
        <p className="fine">
          还原得到的资产可继续用于资产去重、检索、聚合统计——这正是本引擎要交付的「网络空间测绘 URL 资产归一化」。
        </p>
      </div>
    </section>
  )
}
