"use client";

import { useState } from "react";
import { API_URL, PREFIX } from "@/lib/api";

interface ImportResult {
  inserted: number;
  skipped: number;
  errors: string[] | null;
}

export default function ImportPage() {
  const [file, setFile] = useState<File | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent): Promise<void> {
    e.preventDefault();
    if (!file) {
      setError("Selecione um arquivo CSV.");
      return;
    }
    setError("");
    setLoading(true);
    try {
      const body = new FormData();
      body.append("file", file);
      const headers: Record<string, string> = {};
      const token = localStorage.getItem("fintrack_token");
      if (token) headers.Authorization = `Bearer ${token}`;
      const orgId = localStorage.getItem("fintrack_org_id");
      if (orgId) headers["X-Org-ID"] = orgId;
      const res = await fetch(`${API_URL}${PREFIX}/import/csv`, {
        method: "POST",
        headers,
        credentials: "include",
        body,
      });
      if (!res.ok) throw new Error((await res.text().catch(() => "")) || `Erro ${res.status}`);
      const parsed: unknown = await res.json();
      const payload =
        parsed !== null && typeof parsed === "object" && "data" in parsed
          ? (parsed.data as ImportResult)
          : (parsed as ImportResult);
      setResult(payload);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha na importação");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Importar CSV</h1>
      <p className="text-sm text-stone-500">
        Envie um CSV com colunas data, descrição, valor em centavos e categoria.
        Linhas já importadas são ignoradas (dedup por hash).
      </p>
      <form onSubmit={onSubmit} className="max-w-md space-y-3 rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
        <div>
          <label htmlFor="csv-file" className="text-sm font-medium">Arquivo CSV</label>
          <input
            id="csv-file"
            type="file"
            accept=".csv,text/csv"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            className="mt-1 w-full text-sm"
          />
        </div>
        {error !== "" && <p role="alert" className="text-sm text-red-600">{error}</p>}
        <button type="submit" disabled={loading} className="rounded bg-teal-600 px-3 py-2 text-sm text-white disabled:opacity-50">
          {loading ? "Importando…" : "Importar"}
        </button>
      </form>
      {result && (
        <section aria-label="Resultado" className="max-w-md rounded-lg border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900">
          <h2 className="font-semibold">Resultado</h2>
          <ul className="mt-2 text-sm">
            <li>Importadas: <strong>{result.inserted}</strong></li>
            <li>Duplicadas ignoradas: <strong>{result.skipped}</strong></li>
            <li>Erros: <strong>{result.errors?.length ?? 0}</strong></li>
          </ul>
          {result.errors && result.errors.length > 0 && (
            <ul className="mt-2 max-h-48 space-y-1 overflow-auto text-xs">
              {result.errors.map((msg, i) => (
                <li key={i}>{msg}</li>
              ))}
            </ul>
          )}
        </section>
      )}
    </div>
  );
}
