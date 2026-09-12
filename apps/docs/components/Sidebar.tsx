"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { DOCS_NAV } from "@/config/docs-nav";

export default function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 lg:w-72 shrink-0 hidden md:block border-r border-slate-200/80 dark:border-slate-800/80 bg-white/50 dark:bg-[#0d1117]/50 py-8 px-6 sticky top-14 h-[calc(100vh-3.5rem)] overflow-y-auto transition-colors">
      <div className="space-y-6">
        {DOCS_NAV.map((section) => (
          <div key={section.title}>
            <div className="text-[11px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-wider mb-2 px-2.5">
              {section.title}
            </div>
            <ul className="space-y-1">
              {section.items.map((item) => {
                const isActive = pathname === item.href;
                return (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      className={`flex items-center px-3 py-1.5 rounded-lg text-sm transition-all ${
                        isActive
                          ? "bg-blue-50 dark:bg-blue-950/40 text-blue-700 dark:text-blue-400 font-semibold border-l-2 border-blue-600 rounded-l-none"
                          : "text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100/70 dark:hover:bg-slate-800/50 font-normal"
                      }`}
                    >
                      {item.title}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </div>
    </aside>
  );
}