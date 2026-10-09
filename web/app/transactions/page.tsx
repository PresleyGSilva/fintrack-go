"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { formatCents, parseBRLToCents } from "@/lib/money";

interface Tx {
  id: string;
  note: string;
  amount_cents: number;
  kind: string;
  category_id: string | null;
  account_id: string;
  date: string;
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

function monthRange(month: string): { from: string; to: string } {
  const [y, m] = month.split("-").map(Number);
  const last = new Date(y, m, 0).getDate();
  const mm = String(m).padStart(2, "0");
  return { from: `${y}-${mm}-01`, to: `${y}-${mm}-${last}` };
}

export default function TransactionsPage() {
  const [items, setItems] = useState<Tx[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [error, setError] = useState("");
  const [filterKind, setFilterKind] = useState("");
  const [filterCat, setFilterCat] = useState("");
  const [filterPeriod, setFilterPeriod] = useState("");
  const [form, setForm] = useState({ note: "", amount: "", kind: "expense", category_id: "", account_id: "", date: "" });
  const [transfer, setTransfer] = useState({ from: "", to: "", amount: "", memo: "" });
  const [notice, setNotice] = useState("");

  function catName(id: string | null): string {
    if (!id) return "—";
    return categories.find((c) => c.id === id)?.name ?? "—";
  }

  function load(): void {
    const q = new URLSearchParams();
    if (filterKind) q.set("kind", filterKind);
    if (filterCat) q.set("category_id", filterCat);
    if (filterPeriod) {
      const { from, to } = monthRange(filterPeriod);
      q.set("from", from);
      q.set("to", to);
    }
    api<Tx[]>(`/transactions?${q.toString()}`)
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

  async function createTx(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    setNotice("");
    try {
      await api("/transactions", {
        method: "POST",
        body: JSON.stringify({
          note: form.note,
          amount_cents: parseBRLToCents(form.amount),
          kind: form.kind,
          category_id: form.category_id || undefined,
          account_id: form.account_id,
          date: form.date || new Date().toISOString().slice(0, 10),
        }),
      });
      setForm({ note: "", amount: "", kind: "expense", category_id: "", account_id: "", date: "" });
      setNotice("Transação criada.");
      load();
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Erro ao criar");
    }
  }

  async function doTransfer(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    setNotice("");
    try {
      await api("/transfers", {
        method: "POST",
        body: JSON.stringify({
          from_account_id: transfer.from,
          to_account_id: transfer.to,
          amount_cents: parseBRLToCents(transfer.amount),
          memo: transfer.memo || undefined,
        }),
      });
      setTransfer({ from: "", to: "", amount: "", memo: "" });
      setNotice("Transferência realizada.");
      load();
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Erro na transferência");
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Transações</h1>
      {error !== "" && <p role="alert" className="text-sm text-red-600">{error}</p>}
      <section aria-label="Filtros" className="flex flex-wrap gap-2 rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <div>
          <label htmlFor="f-kind" className="text-xs">Tipo</label>
          <select id="f-kind" value={filterKind} onChange={(e) => setFilterKind(e.target.value)} className="ml-1 rounded border px-2 py-1 dark:bg-stone-800">
            <option value="">Todos</option>
            <option value="income">Receita</option>
            <option value="expense">Despesa</option>
          </select>
        </div>
        <div>
          <label htmlFor="f-cat" className="text-xs">Categoria</label>
          <select id="f-cat" value={filterCat} onChange={(e) => setFilterCat(e.target.value)} className="ml-1 rounded border px-2 py-1 dark:bg-stone-800">
            <option value="">Todas</option>
            {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </div>
        <div>
          <label htmlFor="f-period" className="text-xs">Período</label>
          <input id="f-period" type="month" value={filterPeriod} onChange={(e) => setFilterPeriod(e.target.value)} className="ml-1 rounded border px-2 py-1 dark:bg-stone-800" />
        </div>
        <button type="button" onClick={load} className="rounded bg-teal-600 px-3 py-1 text-sm text-white">Filtrar</button>
      </section>

      <section aria-label="Lista" className="overflow-x-auto rounded-lg border border-stone-200 bg-white dark:border-stone-800 dark:bg-stone-900">
        {items.length === 0 ? (
          <p className="p-4 text-sm text-stone-500">Nenhuma transação encontrada. Crie a primeira abaixo.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left dark:border-stone-800">
                <th className="p-2">Data</th><th className="p-2">Descrição</th><th className="p-2">Categoria</th><th className="p-2">Tipo</th><th className="p-2 text-right">Valor</th>
              </tr>
            </thead>
            <tbody>
              {items.map((t) => (
                <tr key={t.id} className="border-b last:border-0 dark:border-stone-800">
                  <td className="p-2">{t.date}</td>
                  <td className="p-2">{t.note}</td>
                  <td className="p-2">{catName(t.category_id)}</td>
                  <td className="p-2">{t.kind === "income" ? "Receita" : t.kind === "transfer" ? "Transferência" : "Despesa"}</td>
                  <td className="p-2 text-right">{formatCents(t.amount_cents)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <div className="grid gap-4 md:grid-cols-2">
        <form onSubmit={createTx} aria-label="Nova transação" className="space-y-2 rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
          <h2 className="font-semibold">Nova transação</h2>
          <label className="block text-sm">Descrição<input required value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
          <label className="block text-sm">Valor (R$)<input required value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} placeholder="123,45" inputMode="decimal" className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
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
          <label className="block text-sm">Conta
            <select required value={form.account_id} onChange={(e) => setForm({ ...form, account_id: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800">
              <option value="">—</option>
              {accounts.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
            </select>
          </label>
          <label className="block text-sm">Data<input type="date" value={form.date} onChange={(e) => setForm({ ...form, date: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
          <button type="submit" className="rounded bg-teal-600 px-3 py-2 text-sm text-white">Adicionar</button>
        </form>
        <form onSubmit={doTransfer} aria-label="Transferência entre contas" className="space-y-2 rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
          <h2 className="font-semibold">Transferência entre contas</h2>
          <label className="block text-sm">De
            <select required value={transfer.from} onChange={(e) => setTransfer({ ...transfer, from: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800">
              <option value="">—</option>
              {accounts.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
            </select>
          </label>
          <label className="block text-sm">Para
            <select required value={transfer.to} onChange={(e) => setTransfer({ ...transfer, to: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800">
              <option value="">—</option>
              {accounts.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
            </select>
          </label>
          <label className="block text-sm">Valor (R$)<input required value={transfer.amount} onChange={(e) => setTransfer({ ...transfer, amount: e.target.value })} placeholder="200,00" inputMode="decimal" className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
          <label className="block text-sm">Descrição<input value={transfer.memo} onChange={(e) => setTransfer({ ...transfer, memo: e.target.value })} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
          <button type="submit" className="rounded bg-amber-600 px-3 py-2 text-sm text-white">Transferir</button>
        </form>
      </div>
      {notice !== "" && <p role="status" className="text-sm text-stone-600 dark:text-stone-300">{notice}</p>}
    </div>
  );
}
