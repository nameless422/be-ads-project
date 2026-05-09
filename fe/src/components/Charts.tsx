type TrendPoint = {
  label: string;
  value: number;
};

type TrendChartProps = {
  title: string;
  caption: string;
  points: TrendPoint[];
  tone?: "green" | "blue" | "amber";
};

export function TrendChart({ title, caption, points, tone = "green" }: TrendChartProps) {
  const max = Math.max(1, ...points.map((point) => point.value));
  const width = 360;
  const height = 112;
  const barWidth = points.length > 0 ? width / points.length : width;

  return (
    <section className="panel chart-panel">
      <div className="panel-head">
        <h2>{title}</h2>
        <span>{caption}</span>
      </div>
      {points.length > 0 ? (
        <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" className={`trend-chart ${tone}`} aria-label={title}>
          {points.map((point, index) => {
            const barHeight = Math.max(4, (point.value / max) * 82);
            const x = index * barWidth + 5;
            const y = height - barHeight - 18;
            const w = Math.max(5, barWidth - 10);
            return <rect key={`${point.label}-${index}`} x={x} y={y} width={w} height={barHeight} rx="5" />;
          })}
        </svg>
      ) : (
        <div className="empty-chart">No trend data</div>
      )}
    </section>
  );
}
