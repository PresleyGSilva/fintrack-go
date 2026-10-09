"use client";

import { useEffect, useState } from "react";
import { api, API_URL, PREFIX } from "@/lib/api";
import { formatCents } from "@/lib/money";

interface Row {
  month: string;
  income_cents: number;
  expense_cents: number;
}

export default function ReportsPage() {
  const [year, setYear] = useState(String(new Date().getFullYear()));
  const [rows, setRows] = useState<Row[]>([]);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  function load(): void {
    api<Row[]>(`/reports/monthly?year=${encodeURIComponent(year)}`)
      .then(setRows)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erro"));
  }
  useEffect(load, []);

  async function exportCSV(): Promise<void> {
    setNotice("");
    try {
      const headers = new Headers();
      const token = localStorage.getItem("fintrack_token");
      if (token) headers.set("Authorization", `Bearer ${token}`);
      const orgId = localStorage.getItem("fintrack_org_id");
      if (orgId) headers.set("X-Org-ID", orgId);
      const res = await fetch(`${API_URL}${PREFIX}/export/csv`, {
        headers,
        credentials: "include",
      });
      if (!res.ok) throw new Error(`Erro ${res.status}`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "fintrack-transacoes.csv";
      a.click();
      URL.revokeObjectURL(url);
      setNotice("CSV exportado.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Falha na exportação");
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Relatórios</h1>
      {error !== "" && <p role="alert" className="text-sm text-red-600">{error}</p>}
      <div className="flex flex-wrap items-end gap-2 rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <div>
          <label htmlFor="rep-year" className="text-sm">Ano</label>
          <input id="rep-year" type="number" value={year} onChange={(e) => setYear(e.target.value)} className="ml-2 w-28 rounded border px-2 py-1 dark:bg-stone-800" />
        </div>
        <button type="button" onClick={load} className="rounded bg-teal-600 px-3 py-1 text-sm text-white">Atualizar</button>
        <button type="button" onClick={exportCSV} className="rounded border border-teal-600 px-3 py-1 text-sm text-teal-700 dark:text-teal-300">Exportar CSV</button>
      </div>
      {rows.length === 0 ? (
        <p className="text-sm text-stone-500">Sem dados para o período. Registre transações para ver o resumo.</p>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-stone-200 bg-white dark:border-stone-800 dark:bg-stone-900">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left dark:border-stone-800">
                <th className="p-2">Mês</th><th className="p-2 text-right">Receitas</th><th className="p-2 text-right">Despesas</th><th className="p-2 text-right">Saldo</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.month} className="border-b last:border-0 dark:border-stone-800">
                  <td className="p-2">{r.month.slice(0, 7)}</td>
                  <td className="p-2 text-right">{formatCents(r.income_cents)}</td>
                  <td className="p-2 text-right">{formatCents(r.expense_cents)}</td>
                  <td className="p-2 text-right font-semibold">{formatCents(r.income_cents - r.expense_cents)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {notice !== "" && <p role="status" className="text-sm">{notice}</p>}
    </div>
  );
}
