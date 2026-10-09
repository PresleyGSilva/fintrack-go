"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { formatCents, parseBRLToCents } from "@/lib/money";

interface Account {
  id: string;
  name: string;
  type: string;
  balance_cents: number;
}

const TYPES = [
  { value: "checking", label: "Conta corrente" },
  { value: "savings", label: "Poupança" },
  { value: "cash", label: "Dinheiro" },
  { value: "credit", label: "Cartão de crédito" },
  { value: "investment", label: "Investimento" },
  { value: "other", label: "Outra" },
];

export default function AccountsPage() {
  const [items, setItems] = useState<Account[]>([]);
  const [error, setError] = useState("");
  const [name, setName] = useState("");
  const [type, setType] = useState("checking");
  const [balance, setBalance] = useState("");
  const [notice, setNotice] = useState("");

  function load(): void {
    api<Account[]>("/accounts")
      .then(setItems)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erro"));
  }
  useEffect(load, []);

  async function create(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    try {
      await api("/accounts", {
        method: "POST",
        body: JSON.stringify({ name, type, balance_cents: parseBRLToCents(balance || "0") }),
      });
      setName(""); setType("checking"); setBalance("");
      setNotice("Conta criada.");
      load();
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Erro");
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Contas e carteiras</h1>
      {error !== "" && <p role="alert" className="text-sm text-red-600">{error} — nenhuma conta carregada.</p>}
      {items.length === 0 && error === "" ? (
        <p className="text-sm text-stone-500">Nenhuma conta ainda. Crie a primeira abaixo.</p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {items.map((a) => (
            <div key={a.id} className="rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
              <p className="font-semibold">{a.name}</p>
              <p className="text-xs text-stone-500">{TYPES.find((t) => t.value === a.type)?.label ?? a.type}</p>
              <p className="mt-2 text-xl font-bold text-teal-700 dark:text-teal-400">{formatCents(a.balance_cents)}</p>
            </div>
          ))}
        </div>
      )}
      <form onSubmit={create} aria-label="Nova conta" className="max-w-md space-y-2 rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <h2 className="font-semibold">Nova conta</h2>
        <label className="block text-sm">Nome<input required value={name} onChange={(e) => setName(e.target.value)} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <label className="block text-sm">Tipo
          <select value={type} onChange={(e) => setType(e.target.value)} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800">
            {TYPES.map((t) => <option key={t.value} value={t.value}>{t.label}</option>)}
          </select>
        </label>
        <label className="block text-sm">Saldo inicial (R$)<input value={balance} onChange={(e) => setBalance(e.target.value)} placeholder="0,00" inputMode="decimal" className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <button type="submit" className="rounded bg-teal-600 px-3 py-2 text-sm text-white">Criar conta</button>
      </form>
      {notice !== "" && <p role="status" className="text-sm">{notice}</p>}
    </div>
  );
}
