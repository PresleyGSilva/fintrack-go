"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { formatCents, parseBRLToCents } from "@/lib/money";

interface Recurrence {
  id: string;
  account_id: string;
  category_id: string | null;
  kind: string;
  amount_cents: number;
  frequency: string;
  next_run: string;
  active: boolean;
  note: string;
}

interface Account {
  id: string;
  name: string;
}

interface Category {
  id: string;
  name: string;
  kind: string;
}

const FREQUENCIES: Record<string, string> = {
  daily: "Diária",
  weekly: "Semanal",
  monthly: "Mensal",
};

export default function RecurrencesPage() {
  const [items, setItems] = useState<Recurrence[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [error, setError] = useState("");
  const [form, setForm] = useState({ account_id: "", category_id: "", kind: "expense", amount: "", frequency: "monthly", next_run: "", note: "" });
  const [notice, setNotice] = useState("");

  function load(): void {
    api<Recurrence[]>("/recurrences")
      .then(setItems)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erro"));
    api<Account[]>("/accounts")
      .then(setAccounts)
      .catch(() => setAccounts([]));
    api<Category[]>("/categories")
      .then(setCategories)
      .catch(() => setCategories([]));
  }
  useEffect(load, []);

  async function create(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    try {
      await api("/recurrences", {
        method: "POST",
        body: JSON.stringify({
          account_id: form.account_id,
          category_id: form.category_id || undefined,
          kind: form.kind,
          amount_cents: parseBRLToCents(form.amount),
          frequency: form.frequency,
          next_run: form.next_run,
          active: true,
          note: form.note || undefined,
        }),
      });
      setForm({ account_id: "", category_id: "", kind: "expense", amount: "", frequency: "monthly", next_run: "", note: "" });
      setNotice("Recorrência criada.");
      load();
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Erro");
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Recorrências</h1>
      {error !== "" && <p role="alert" className="text-sm text-red-600">{error}</p>}
      {items.length === 0 ? (
        <p className="text-sm text-stone-500">Nenhuma recorrência ainda. Crie a primeira abaixo.</p>
      ) : (
        <ul className="space-y-3">
          {items.map((r) => (
            <li key={r.id} className="rounded-lg border border-stone-200 bg-white p-4 text-sm dark:border-stone-800 dark:bg-stone-900">
              <div className="flex flex-wrap justify-between gap-2">
                <span className="font-semibold">{r.note || (r.kind === "income" ? "Receita" : "Despesa")}</span>
                <span className={r.kind === "income" ? "text-teal-700" : "text-amber-700"}>{formatCents(r.amount_cents)}</span>
              </div>
              <p className="mt-1 text-xs text-stone-500">
                {FREQUENCIES[r.frequency] ?? r.frequency} · próxima em {r.next_run} · {r.active ? "ativa" : "pausada"}
              </p>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={create} aria-label="Nova recorrência" className="max-w-md space-y-2 rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <h2 className="font-semibold">Nova recorrência</h2>
        <label className="block text-sm">Conta
          <select required value={form.account_id} onChange={(e) => setForm({ ...form, account_id: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800">
            <option value="">—</option>
            {accounts.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
          </select>
        </label>
        <label className="block text-sm">Tipo
          <select value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800">
            <option value="expense">Despesa</option><option value="income">Receita</option>
          </select>
        </label>
        <label className="block text-sm">Categoria
          <select value={form.category_id} onChange={(e) => setForm({ ...form, category_id: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800">
            <option value="">—</option>
            {categories.filter((c) => c.kind === form.kind).map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </label>
        <label className="block text-sm">Valor (R$)<input required value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} placeholder="100,00" inputMode="decimal" className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <label className="block text-sm">Frequência
          <select value={form.frequency} onChange={(e) => setForm({ ...form, frequency: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800">
            {Object.entries(FREQUENCIES).map(([v, l]) => <option key={v} value={v}>{l}</option>)}
          </select>
        </label>
        <label className="block text-sm">Próxima execução<input required type="date" value={form.next_run} onChange={(e) => setForm({ ...form, next_run: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <label className="block text-sm">Descrição<input value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <button type="submit" className="rounded bg-teal-600 px-3 py-2 text-sm text-white">Criar</button>
      </form>
      {notice !== "" && <p role="status" className="text-sm">{notice}</p>}
    </div>
  );
}
