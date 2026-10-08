import { NORMALIZE_DEMO, PAIN_POINTS } from '../content'
import SectionTitle from './SectionTitle'

export default function Problems() {
  return (
    <section className="section" id="problems">
      <div className="wrap">
        <SectionTitle
          eyebrow="WHY IT MATTERS"
          title="解决什么问题"
          subtitle="三大典型场景，同一个根因：抓到的流量按「值」散开，没人知道「这两条 URL 是不是同一个接口」。"
        />
        <div className="pull-quote">
          散乱的 URL 不是资产，<b>还原成真实路由结构</b>才算资产。
        </div>

        <div className="grid-3">
          {PAIN_POINTS.map((p, i) => (
            <div className="card" key={p.title}>
              <div className="mark">0{i + 1}</div>
              <h3>{p.title}</h3>
              <div className="loss">{p.loss}</div>
              <p>{p.desc}</p>
            </div>
          ))}
        </div>

        <div className="diff">
          <h3>关键区别：是「还原」而不是「相似度」</h3>
          {NORMALIZE_DEMO.map((row) => (
            <div className={`diff-row${row.kind === 'fixed' ? ' fixed' : ''}`} key={row.raw + row.note}>
              <span className="raw">{row.raw}</span>
              <span className="arrow">→</span>
              <span className="res">{row.result}</span>
              <span className="note">{row.note}</span>
            </div>
          ))}
          <p className="foot">
            左边是爬虫眼里的一条条 URL，右边是目标服务器真实的路由结构——
            反推出来的不是猜测的相似度，而是真实存在的接口形态。
            <br />
            /api/users/profile 是真实存在的固定路径，不会被吞进 {'{users_id}'} 模板。
          </p>
        </div>
      </div>
    </section>
  )
}
