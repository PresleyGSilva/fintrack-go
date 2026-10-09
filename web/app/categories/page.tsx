"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

interface Category {
  id: string;
  name: string;
  kind: string;
  color: string;
}

const KIND_LABEL: Record<string, string> = { income: "Receita", expense: "Despesa" };

export default function CategoriesPage() {
  const [items, setItems] = useState<Category[]>([]);
  const [error, setError] = useState("");
  const [name, setName] = useState("");
  const [kind, setKind] = useState("expense");
  const [color, setColor] = useState("#0D9488");
  const [notice, setNotice] = useState("");

  function load(): void {
    api<Category[]>("/categories")
      .then(setItems)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erro"));
  }
  useEffect(load, []);

  async function create(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    try {
      await api("/categories", {
        method: "POST",
        body: JSON.stringify({ name, kind, color }),
      });
      setName(""); setKind("expense"); setColor("#0D9488");
      setNotice("Categoria criada.");
      load();
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Erro");
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Categorias</h1>
      {error !== "" && <p role="alert" className="text-sm text-red-600">{error}</p>}
      {items.length === 0 ? (
        <p className="text-sm text-stone-500">Nenhuma categoria ainda. Crie a primeira abaixo.</p>
      ) : (
        <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {items.map((c) => (
            <li key={c.id} className="flex items-center gap-3 rounded-lg border border-stone-200 bg-white p-3 dark:border-stone-800 dark:bg-stone-900">
              <span aria-hidden className="h-6 w-6 rounded-full" style={{ backgroundColor: c.color || "#0D9488" }} />
              <div>
                <p className="text-sm font-semibold">{c.name}</p>
                <p className="text-xs text-stone-500">{KIND_LABEL[c.kind] ?? c.kind}</p>
              </div>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={create} aria-label="Nova categoria" className="max-w-md space-y-2 rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <h2 className="font-semibold">Nova categoria</h2>
        <label className="block text-sm">Nome<input required value={name} onChange={(e) => setName(e.target.value)} placeholder="Mercado" className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800" /></label>
        <label className="block text-sm">Tipo
          <select value={kind} onChange={(e) => setKind(e.target.value)} className="mt-1 w-full rounded border px-2 py-1 dark:bg-stone-800">
            <option value="expense">Despesa</option><option value="income">Receita</option>
          </select>
        </label>
        <label className="block text-sm">Cor<input type="color" value={color} onChange={(e) => setColor(e.target.value)} className="mt-1 h-9 w-16 rounded border px-1 py-1 dark:bg-stone-800" /></label>
        <button type="submit" className="rounded bg-teal-600 px-3 py-2 text-sm text-white">Criar</button>
      </form>
      {notice !== "" && <p role="status" className="text-sm">{notice}</p>}
    </div>
  );
}
