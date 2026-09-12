"use client";

import { useMemo } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { DOCS_NAV } from "@/config/docs-nav";

const GITHUB_REPO_DOCS_URL =
  "https://github.com/budment/budment/edit/main/docs";

export default function DocFooter() {
  const pathname = usePathname();

  const allItems = useMemo(
    () => DOCS_NAV.flatMap((section) => section.items),
    [],
  );

  const currentIndex = allItems.findIndex((item) => item.href === pathname);
  const prevItem = currentIndex > 0 ? allItems[currentIndex - 1] : null;
  const nextItem =
    currentIndex !== -1 && currentIndex < allItems.length - 1
      ? allItems[currentIndex + 1]
      : null;

  const cleanPath = pathname.replace(/^\/docs\/?/, "").replace(/\/$/, "");
  const docSlug = cleanPath || "getting-started/01-introduction";
  const githubEditUrl = `${GITHUB_REPO_DOCS_URL}/${docSlug}.md`;

  return (
    <footer className="mt-14 pt-6 border-t border-slate-200/70 dark:border-slate-800/80">
      {/* Edit on GitHub */}
      <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400 mb-6">
        <a
          href={githubEditUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 hover:text-slate-900 dark:hover:text-slate-100 transition-colors group"
        >
          <svg
            className="w-3.5 h-3.5 text-slate-400 group-hover:text-blue-500 transition-colors"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L6.832 19.82a4.5 4.5 0 01-1.897 1.13l-2.685.8.8-2.685a4.5 4.5 0 011.13-1.897L16.863 4.487zm0 0L19.5 7.125"
            />
          </svg>
          <span className="font-medium">Edit this page on GitHub</span>
          <span className="text-[11px] text-slate-400 opacity-60 group-hover:opacity-100 group-hover:translate-x-0.5 transition-all">
            ↗
          </span>
        </a>
      </div>

      {/* Previous / Next */}
      <nav
        aria-label="Documentation Pagination"
        className="grid grid-cols-1 sm:grid-cols-2 gap-4"
      >
        {prevItem ? (
          <Link
            href={prevItem.href}
            className="group relative flex flex-col justify-between p-4 rounded-xl border border-slate-200/80 dark:border-slate-800/80 bg-white/50 dark:bg-slate-900/30 hover:bg-slate-50 dark:hover:bg-slate-800/50 hover:border-slate-300 dark:hover:border-slate-700 transition-all duration-200 shadow-2xs hover:shadow-xs"
          >
            <div className="flex items-center gap-1.5 text-xs font-medium text-slate-400 dark:text-slate-500 group-hover:text-slate-600 dark:group-hover:text-slate-300 transition-colors mb-2">
              <svg
                className="w-3.5 h-3.5 transition-transform duration-200 group-hover:-translate-x-1"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M15 19l-7-7 7-7"
                />
              </svg>
              <span>Previous</span>
            </div>
            <span className="text-[14.5px] font-semibold text-slate-800 dark:text-slate-200 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors line-clamp-1">
              {prevItem.title}
            </span>
          </Link>
        ) : (
          <div className="hidden sm:block" />
        )}

        {nextItem && (
          <Link
            href={nextItem.href}
            className="group relative flex flex-col justify-between p-4 rounded-xl border border-slate-200/80 dark:border-slate-800/80 bg-white/50 dark:bg-slate-900/30 hover:bg-slate-50 dark:hover:bg-slate-800/50 hover:border-slate-300 dark:hover:border-slate-700 transition-all duration-200 shadow-2xs hover:shadow-xs text-right sm:col-start-2"
          >
            <div className="flex items-center justify-end gap-1.5 text-xs font-medium text-slate-400 dark:text-slate-500 group-hover:text-slate-600 dark:group-hover:text-slate-300 transition-colors mb-2">
              <span>Next</span>
              <svg
                className="w-3.5 h-3.5 transition-transform duration-200 group-hover:translate-x-1"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M9 5l7 7-7 7"
                />
              </svg>
            </div>
            <span className="text-[14.5px] font-semibold text-slate-800 dark:text-slate-200 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors line-clamp-1">
              {nextItem.title}
            </span>
          </Link>
        )}
      </nav>
    </footer>
  );
}
