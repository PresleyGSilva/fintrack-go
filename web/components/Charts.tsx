"use client";

export interface MonthlyPoint {
  label: string;
  income: number;
  expense: number;
}

export function MonthlyBarChart({ data }: { data: MonthlyPoint[] }) {
  if (data.length === 0) {
    return <p className="text-sm text-stone-500">Sem dados mensais ainda.</p>;
  }
  const max = Math.max(1, ...data.flatMap((d) => [d.income, d.expense]));
  const W = 560;
  const H = 220;
  const pad = 28;
  const n = data.length;
  const groupW = (W - pad * 2) / n;
  const barW = Math.min(28, (groupW - 12) / 2);

  return (
    <svg
      viewBox={`0 0 ${W} ${H}`}
      role="img"
      aria-label="Gráfico de receitas e despesas por mês"
      className="w-full"
    >
      {[0.25, 0.5, 0.75, 1].map((f) => {
        const y = H - pad - (H - pad * 2) * f;
        return <line key={f} x1={pad} x2={W - 8} y1={y} y2={y} stroke="#e7e5e4" strokeWidth={1} />;
      })}
      {data.map((d, i) => {
        const cx = pad + groupW * i + groupW / 2;
        const ih = ((H - pad * 2) * d.income) / max;
        const eh = ((H - pad * 2) * d.expense) / max;
        return (
          <g key={d.label}>
            <rect
              x={cx - barW - 2}
              y={H - pad - ih}
              width={barW}
              height={ih}
              rx={3}
              fill="#0D9488"
            >
              <title>{`${d.label} receita: ${d.income}`}</title>
            </rect>
            <rect
              x={cx + 2}
              y={H - pad - eh}
              width={barW}
              height={eh}
              rx={3}
              fill="#f59e0b"
            >
              <title>{`${d.label} despesa: ${d.expense}`}</title>
            </rect>
            <text x={cx} y={H - 8} textAnchor="middle" fontSize={11} fill="currentColor">
              {d.label}
            </text>
          </g>
        );
      })}
    </svg>
  );
}

export interface Slice {
  label: string;
  value: number;
  color: string;
}

export function CategoryPie({ data }: { data: Slice[] }) {
  if (data.length === 0) {
    return <p className="text-sm text-stone-500">Sem categorias ainda.</p>;
  }
  const total = data.reduce((acc, s) => acc + s.value, 0) || 1;
  let angle = -Math.PI / 2;
  const R = 70;
  const CX = 90;
  const CY = 90;

  const paths = data.map((s) => {
    const frac = s.value / total;
    const a2 = angle + frac * Math.PI * 2;
    const x1 = CX + R * Math.cos(angle);
    const y1 = CY + R * Math.sin(angle);
    const x2 = CX + R * Math.cos(a2);
    const y2 = CY + R * Math.sin(a2);
    const large = frac > 0.5 ? 1 : 0;
    angle = a2;
    if (frac >= 1) {
      return { ...s, d: "", circle: true };
    }
    return { ...s, d: `M ${CX} ${CY} L ${x1} ${y1} A ${R} ${R} 0 ${large} 1 ${x2} ${y2} Z`, circle: false };
  });

  return (
    <div className="flex flex-wrap items-center gap-4">
      <svg viewBox="0 0 180 180" role="img" aria-label="Gastos por categoria" className="h-44 w-44">
        {paths.map((p) =>
          p.circle ? (
            <circle key={p.label} cx={CX} cy={CY} r={R} fill={p.color}>
              <title>{`${p.label}: ${p.value}`}</title>
            </circle>
          ) : (
            <path key={p.label} d={p.d} fill={p.color} stroke="#fff" strokeWidth={1}>
              <title>{`${p.label}: ${p.value}`}</title>
            </path>
          ),
        )}
      </svg>
      <ul className="space-y-1 text-sm">
        {data.map((s) => (
          <li key={s.label} className="flex items-center gap-2">
            <span aria-hidden className="inline-block h-3 w-3 rounded" style={{ background: s.color }} />
            {s.label} — {(100 * s.value) / total > 0 ? ((100 * s.value) / total).toFixed(0) : 0}%
          </li>
        ))}
      </ul>
    </div>
  );
}

export function ProgressBar({ value, max }: { value: number; max: number }) {
  const pct = max > 0 ? Math.min(100, Math.round((100 * value) / max)) : 0;
  return (
    <div
      role="progressbar"
      aria-valuenow={pct}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-label={`Progresso ${pct}%`}
      className="h-2 w-full overflow-hidden rounded bg-stone-200 dark:bg-stone-800"
    >
      <div className="h-full rounded bg-teal-600" style={{ width: `${pct}%` }} />
    </div>
  );
}
