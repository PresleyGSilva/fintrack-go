"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { formatCents } from "@/lib/money";
import { MonthlyBarChart, CategoryPie } from "@/components/Charts";

interface CategoryTotal {
  category_id: string | null;
  category_name: string | null;
  kind: string;
  total_cents: number;
}

interface Summary {
  income_cents: number;
  expense_cents: number;
  balance_cents: number;
  by_category: CategoryTotal[];
}

interface MonthPoint {
  month: string;
  income_cents: number;
  expense_cents: number;
}

interface Tx {
  id: string;
  note: string;
  amount_cents: number;
  kind: string;
  date: string;
}

const PALETTE = ["#0D9488", "#f59e0b", "#6366f1", "#ec4899", "#84cc16", "#64748b"];

export default function DashboardPage() {
  const [summary, setSummary] = useState<Summary | null>(null);
  const [monthly, setMonthly] = useState<MonthPoint[]>([]);
  const [recent, setRecent] = useState<Tx[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    const year = new Date().getFullYear();
    Promise.all([
      api<Summary>("/reports/summary"),
      api<MonthPoint[]>(`/reports/monthly?year=${year}`).catch(() => []),
      api<Tx[]>("/transactions").catch(() => []),
    ])
      .then(([s, m, txs]) => {
        setSummary(s);
        setMonthly(m);
        setRecent(txs.slice(0, 8));
      })
      .catch((err: unknown) =>
        setError(err instanceof Error ? err.message : "Erro"),
      );
  }, []);

  if (error !== "") {
    return (
      <div>
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="mt-4 rounded border border-amber-300 bg-amber-50 p-4 text-sm dark:bg-stone-900">
          Não foi possível carregar o resumo ({error}). Verifique se o backend está rodando e faça login novamente.
        </p>
      </div>
    );
  }

  if (!summary) {
    return (
      <div>
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="mt-4 text-sm text-stone-500">Carregando…</p>
      </div>
    );
  }

  const cats = summary.by_category.map((c, i) => ({
    label: c.category_name ?? (c.kind === "income" ? "Receitas" : "Despesas"),
    value: c.total_cents,
    color: PALETTE[i % PALETTE.length],
  }));

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
          <p className="text-sm text-stone-500">Receitas</p>
          <p className="text-2xl font-bold text-teal-700 dark:text-teal-400">{formatCents(summary.income_cents)}</p>
        </div>
        <div className="rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
          <p className="text-sm text-stone-500">Despesas</p>
          <p className="text-2xl font-bold text-amber-600">{formatCents(summary.expense_cents)}</p>
        </div>
        <div className="rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
          <p className="text-sm text-stone-500">Saldo</p>
          <p className="text-2xl font-bold">{formatCents(summary.balance_cents)}</p>
        </div>
      </div>
      <section aria-label="Evolução mensal" className="rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <h2 className="mb-2 font-semibold">Evolução mensal</h2>
        <MonthlyBarChart
          data={monthly.map((m) => ({
            label: m.month.slice(5, 7),
            income: m.income_cents,
            expense: m.expense_cents,
          }))}
        />
      </section>
      <div className="grid gap-4 md:grid-cols-2">
        <section aria-label="Por categoria" className="rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
          <h2 className="mb-2 font-semibold">Por categoria</h2>
          <CategoryPie data={cats} />
        </section>
        <section aria-label="Transações recentes" className="rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
          <h2 className="mb-2 font-semibold">Recentes</h2>
          {recent.length === 0 ? (
            <p className="text-sm text-stone-500">Nenhuma transação ainda. Comece em Transações.</p>
          ) : (
            <ul className="divide-y divide-stone-200 text-sm dark:divide-stone-800">
              {recent.map((t) => (
                <li key={t.id} className="flex justify-between py-2">
                  <span>{t.note} <span className="text-stone-400">· {t.date}</span></span>
                  <span className={t.kind === "income" ? "text-teal-700" : "text-amber-700"}>{formatCents(t.amount_cents)}</span>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </div>
  );
}
