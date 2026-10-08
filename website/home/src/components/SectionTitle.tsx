/** 各 section 复用的标题区 */
export default function SectionTitle({ eyebrow, title, subtitle }: { eyebrow?: string; title: string; subtitle: string }) {
  return (
    <div className="sec-head">
      {eyebrow && <div className="eyebrow">{eyebrow}</div>}
      <h2>{title}</h2>
      <p>{subtitle}</p>
    </div>
  )
}
