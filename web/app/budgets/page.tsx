"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { formatCents, parseBRLToCents } from "@/lib/money";
import { ProgressBar } from "@/components/Charts";

interface Budget {
  id: string;
  category_id: string;
  month: string;
  limit_cents: number;
}

interface Category {
  id: string;
  name: string;
  kind: string;
}

interface CategoryTotal {
  category_id: string | null;
  category_name: string | null;
  kind: string;
  total_cents: number;
}

export default function BudgetsPage() {
  const [items, setItems] = useState<Budget[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [spentByCat, setSpentByCat] = useState<Record<string, number>>({});
  const [error, setError] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [month, setMonth] = useState("");
  const [limit, setLimit] = useState("");
  const [notice, setNotice] = useState("");

  function load(): void {
    api<Budget[]>("/budgets")
      .then((list) => {
        setItems(list);
        const months = [...new Set(list.map((b) => b.month.slice(0, 7)))];
        Promise.all(
          months.map((m) => {
            const [y, mm] = m.split("-").map(Number);
            const last = new Date(y, mm, 0).getDate();
            return api<{ by_category: CategoryTotal[] }>(
              `/reports/summary?from=${y}-${String(mm).padStart(2, "0")}-01&to=${y}-${String(mm).padStart(2, "0")}-${last}`,
            ).catch(() => ({ by_category: [] }));
          }),
        ).then((summaries) => {
          const acc: Record<string, number> = {};
          for (const s of summaries) {
            for (const c of s.by_category) {
              if (c.kind === "expense" && c.category_id) {
                acc[c.category_id] = (acc[c.category_id] ?? 0) + c.total_cents;
              }
            }
          }
          setSpentByCat(acc);
        });
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erro"));
    api<Category[]>("/categories")
      .then(setCategories)
      .catch(() => setCategories([]));
  }
  useEffect(load, []);

  async function create(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    try {
      await api("/budgets", {
        method: "POST",
        body: JSON.stringify({ category_id: categoryId, month: `${month}-01`, limit_cents: parseBRLToCents(limit) }),
      });
      setCategoryId(""); setMonth(""); setLimit("");
      setNotice("Orçamento criado.");
      load();
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Erro");
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Orçamentos mensais</h1>
      {error !== "" && <p role="alert" className="text-sm text-red-600">{error}</p>}
      {items.length === 0 ? (
        <p className="text-sm text-stone-500">Nenhum orçamento ainda. Crie o primeiro abaixo.</p>
      ) : (
        <ul className="space-y-3">
          {items.map((b) => {
            const spent = spentByCat[b.category_id] ?? 0;
            const cat = categories.find((c) => c.id === b.category_id)?.name ?? "Categoria";
            return (
              <li key={b.id} className="rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
                <div className="flex justify-between text-sm">
                  <span className="font-semibold">{cat} <span className="font-normal text-stone-500">· {b.month.slice(0, 7)}</span></span>
                  <span>{formatCents(spent)} / {formatCents(b.limit_cents)}</span>
                </div>
                <div className="mt-2"><ProgressBar value={spent} max={b.limit_cents} /></div>
              </li>
            );
          })}
        </ul>
      )}
      <form onSubmit={create} aria-label="Novo orçamento" className="max-w-md space-y-2 rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <h2 className="font-semibold">Novo orçamento</h2>
        <label className="block text-sm">Categoria
          <select required value={categoryId} onChange={(e) => setCategoryId(e.target.value)} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800">
            <option value="">—</option>
            {categories.filter((c) => c.kind === "expense").map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </label>
        <label className="block text-sm">Mês<input required type="month" value={month} onChange={(e) => setMonth(e.target.value)} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <label className="block text-sm">Limite (R$)<input required value={limit} onChange={(e) => setLimit(e.target.value)} placeholder="800,00" inputMode="decimal" className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <button type="submit" className="rounded bg-teal-600 px-3 py-2 text-sm text-white">Criar</button>
      </form>
      {notice !== "" && <p role="status" className="text-sm">{notice}</p>}
    </div>
  );
}
