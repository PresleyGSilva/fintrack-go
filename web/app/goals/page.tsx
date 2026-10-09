"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { formatCents, parseBRLToCents } from "@/lib/money";
import { ProgressBar } from "@/components/Charts";

interface Goal {
  id: string;
  name: string;
  target_cents: number;
  current_cents: number;
}

export default function GoalsPage() {
  const [items, setItems] = useState<Goal[]>([]);
  const [error, setError] = useState("");
  const [name, setName] = useState("");
  const [target, setTarget] = useState("");
  const [current, setCurrent] = useState("");
  const [notice, setNotice] = useState("");

  function load(): void {
    api<Goal[]>("/goals")
      .then(setItems)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erro"));
  }
  useEffect(load, []);

  async function create(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    try {
      await api("/goals", {
        method: "POST",
        body: JSON.stringify({
          name,
          target_cents: parseBRLToCents(target),
          current_cents: parseBRLToCents(current || "0"),
        }),
      });
      setName(""); setTarget(""); setCurrent("");
      setNotice("Meta criada.");
      load();
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Erro");
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Metas</h1>
      {error !== "" && <p role="alert" className="text-sm text-red-600">{error}</p>}
      {items.length === 0 ? (
        <p className="text-sm text-stone-500">Nenhuma meta ainda. Crie a primeira abaixo.</p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {items.map((g) => (
            <div key={g.id} className="rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
              <p className="font-semibold">{g.name}</p>
              <p className="mt-1 text-sm">{formatCents(g.current_cents)} de {formatCents(g.target_cents)}</p>
              <div className="mt-2"><ProgressBar value={g.current_cents} max={g.target_cents} /></div>
            </div>
          ))}
        </div>
      )}
      <form onSubmit={create} aria-label="Nova meta" className="max-w-md space-y-2 rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <h2 className="font-semibold">Nova meta</h2>
        <label className="block text-sm">Nome<input required value={name} onChange={(e) => setName(e.target.value)} placeholder="Viagem" className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <label className="block text-sm">Objetivo (R$)<input required value={target} onChange={(e) => setTarget(e.target.value)} placeholder="5000,00" inputMode="decimal" className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <label className="block text-sm">Valor atual (R$)<input value={current} onChange={(e) => setCurrent(e.target.value)} placeholder="0,00" inputMode="decimal" className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <button type="submit" className="rounded bg-teal-600 px-3 py-2 text-sm text-white">Criar meta</button>
      </form>
      {notice !== "" && <p role="status" className="text-sm">{notice}</p>}
    </div>
  );
}
