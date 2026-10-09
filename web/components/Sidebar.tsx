"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import ThemeToggle from "./ThemeToggle";
import { logout } from "@/lib/api";

const NAV = [
  { href: "/", label: "Dashboard" },
  { href: "/transactions", label: "Transações" },
  { href: "/accounts", label: "Contas" },
  { href: "/categories", label: "Categorias" },
  { href: "/recurrences", label: "Recorrências" },
  { href: "/budgets", label: "Orçamentos" },
  { href: "/goals", label: "Metas" },
  { href: "/reports", label: "Relatórios" },
  { href: "/import", label: "Importar" },
];

export default function Sidebar() {
  const pathname = usePathname();
  return (
    <aside className="flex w-full flex-col gap-4 bg-stone-900 p-4 text-stone-100 dark:bg-black md:min-h-screen md:w-60">
      <div className="flex items-center justify-between">
        <span className="text-lg font-bold text-teal-400">FinTrack</span>
        <ThemeToggle />
      </div>
      <nav aria-label="Navegação principal" className="flex flex-row flex-wrap gap-2 md:flex-col">
        {NAV.map((item) => {
          const active = pathname === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              aria-current={active ? "page" : undefined}
              className={`rounded px-3 py-2 text-sm ${
                active
                  ? "bg-teal-600 text-white"
                  : "text-stone-300 hover:bg-stone-800 hover:text-white"
              }`}
            >
              {item.label}
            </Link>
          );
        })}
      </nav>
      <button
        type="button"
        onClick={logout}
        className="mt-auto rounded border border-stone-700 px-3 py-2 text-sm text-stone-300 hover:bg-stone-800"
      >
        Sair
      </button>
    </aside>
  );
}
