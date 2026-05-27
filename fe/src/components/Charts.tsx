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
  const values = points.map((point) => point.value);
  const max = Math.max(1, ...values);
  const min = Math.min(0, ...values);
  const span = Math.max(1, max - min);
  const width = 360;
  const height = 112;
  const plotTop = 12;
  const plotBottom = height - 22;
  const plotHeight = plotBottom - plotTop;
  const coordinates = points.map((point, index) => {
    const x = points.length === 1 ? width / 2 : 12 + (index * (width - 24)) / (points.length - 1);
    const y = plotBottom - ((point.value - min) / span) * plotHeight;
    return { ...point, x, y };
  });
  const path = coordinates.map((point) => `${point.x},${point.y}`).join(" ");

  return (
    <section className="panel chart-panel">
      <div className="panel-head">
        <h2>{title}</h2>
        <span>{caption}</span>
      </div>
      {points.length > 0 ? (
        <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" className={`trend-chart ${tone}`} aria-label={title}>
          <line className="trend-axis" x1="8" x2={width - 8} y1={plotBottom} y2={plotBottom} />
          <polyline className="trend-line" points={path} fill="none" />
          {coordinates.map((point, index) => (
            <circle key={`${point.label}-${index}`} className="trend-point" cx={point.x} cy={point.y} r="3.5" />
          ))}
        </svg>
      ) : (
        <div className="empty-chart">No trend data</div>
      )}
    </section>
  );
}
