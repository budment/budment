"use client";

import React, { useState, useEffect, useMemo, useRef } from "react";
import { useRouter } from "next/navigation";
import { DOCS_NAV } from "@/config/docs-nav";

interface SearchDialogProps {
  open: boolean;
  setOpen: (open: boolean | ((prev: boolean) => boolean)) => void;
}

function HighlightMatch({ text, query }: { text: string; query: string }) {
  if (!query.trim()) return <>{text}</>;

  const escaped = query.trim().replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const parts = text.split(new RegExp(`(${escaped})`, "gi"));

  return (
    <>
      {parts.map((part, i) =>
        part.toLowerCase() === query.toLowerCase().trim() ? (
          <span
            key={i}
            className="text-blue-600 dark:text-blue-400 font-semibold bg-blue-100/80 dark:bg-blue-950/90 px-1 py-0.5 rounded-sm"
          >
            {part}
          </span>
        ) : (
          part
        )
      )}
    </>
  );
}

export default function SearchDialog({ open, setOpen }: SearchDialogProps) {
  const router = useRouter();

  const [query, setQuery] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(0);

  const listContainerRef = useRef<HTMLDivElement>(null);
  const listContentRef = useRef<HTMLDivElement>(null);
  const isFirstExpandRef = useRef(true);
  const activeItemRef = useRef<HTMLDivElement>(null);

  // Flatten navigation items
  const allDocs = useMemo(
    () =>
      DOCS_NAV.flatMap((section) =>
        section.items.map((item) => ({
          ...item,
          category: section.title,
          keywords: item.keywords ?? "",
        }))
      ),
    []
  );

  const filteredDocs = useMemo(() => {
    const clean = query.toLowerCase().trim();
    if (!clean) return allDocs;
    return allDocs.filter(
      (doc) =>
        doc.title.toLowerCase().includes(clean) ||
        doc.category.toLowerCase().includes(clean) ||
        doc.keywords.toLowerCase().includes(clean)
    );
  }, [query, allDocs]);

  // Lock body scroll and compensate for scrollbar shift
  useEffect(() => {
    if (!open) return;

    const originalOverflow = document.body.style.overflow;
    const originalPaddingRight = document.body.style.paddingRight;
    const scrollbarWidth = window.innerWidth - document.documentElement.clientWidth;

    document.body.style.overflow = "hidden";
    if (scrollbarWidth > 0) {
      document.body.style.paddingRight = `${scrollbarWidth}px`;
    }

    return () => {
      document.body.style.overflow = originalOverflow;
      document.body.style.paddingRight = originalPaddingRight;
    };
  }, [open]);

  // Animate dynamic height transitions
  useEffect(() => {
    if (!open) {
      isFirstExpandRef.current = true;
      return;
    }
    if (!listContentRef.current || !listContainerRef.current) return;

    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const targetHeight = Math.min(
          (entry.target as HTMLElement).offsetHeight,
          420
        );

        if (listContainerRef.current) {
          if (isFirstExpandRef.current) {
            listContainerRef.current.style.transition = "none";
            listContainerRef.current.style.height = `${targetHeight}px`;

            requestAnimationFrame(() => {
              if (listContainerRef.current) {
                listContainerRef.current.style.transition =
                  "height 0.28s cubic-bezier(0.16, 1, 0.3, 1)";
              }
            });
            isFirstExpandRef.current = false;
          } else {
            listContainerRef.current.style.height = `${targetHeight}px`;
          }
        }
      }
    });

    observer.observe(listContentRef.current);
    return () => observer.disconnect();
  }, [open]);

  // Auto-scroll selected item into view
  useEffect(() => {
    if (activeItemRef.current) {
      activeItemRef.current.scrollIntoView({
        behavior: "smooth",
        block: "nearest",
      });
    }
  }, [selectedIndex]);

  // Global keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || (e.key === "/" && !open)) {
        if (["INPUT", "TEXTAREA"].includes((e.target as HTMLElement)?.tagName) && e.key === "/") {
          return;
        }
        e.preventDefault();
        setOpen((prev) => !prev);
      } else if (e.key === "Escape" && open) {
        e.preventDefault();
        setOpen(false);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, setOpen]);

  const handleSelect = (href: string) => {
    setOpen(false);
    router.push(href);
  };

  const handleInputKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      e.preventDefault();
      setOpen(false);
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((prev) => (prev + 1 < filteredDocs.length ? prev + 1 : 0));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((prev) => (prev - 1 >= 0 ? prev - 1 : filteredDocs.length - 1));
    } else if (e.key === "Enter" && filteredDocs[selectedIndex]) {
      e.preventDefault();
      handleSelect(filteredDocs[selectedIndex].href);
    }
  };

  if (!open) return null;

  return (
    <>
      <style
        dangerouslySetInnerHTML={{
          __html: `
            @keyframes searchDialogBackdrop {
              from { opacity: 0; backdrop-filter: blur(0px); }
              to { opacity: 1; backdrop-filter: blur(12px); }
            }
            @keyframes searchDialogModal {
              from { opacity: 0; transform: scale(0.96) translateY(-12px); }
              to { opacity: 1; transform: scale(1) translateY(0); }
            }
            @keyframes searchCascadeIn {
              0% { opacity: 0; transform: translateY(12px) scale(0.97); }
              100% { opacity: 1; transform: translateY(0) scale(1); }
            }
            .search-dialog-backdrop {
              animation: searchDialogBackdrop 0.2s ease-out forwards;
            }
            .search-dialog-card {
              animation: searchDialogModal 0.22s cubic-bezier(0.16, 1, 0.3, 1) forwards;
            }
            .search-cascade-item {
              animation: searchCascadeIn 0.22s cubic-bezier(0.16, 1, 0.3, 1) both;
            }
          `,
        }}
      />

      <div className="fixed inset-0 z-50 flex items-start justify-center pt-12 sm:pt-20 px-4 select-none">
        <div
          className="search-dialog-backdrop fixed inset-0 bg-slate-950/70 dark:bg-black/80"
          onClick={() => setOpen(false)}
        />

        <div className="search-dialog-card relative w-full max-w-3xl lg:max-w-4xl bg-white dark:bg-[#12141a] rounded-2xl shadow-2xl border border-slate-200/90 dark:border-slate-800 overflow-hidden font-sans flex flex-col max-h-[82vh]">
          <div className="flex items-center px-5 border-b border-slate-200/80 dark:border-slate-800/80 bg-slate-50/50 dark:bg-slate-900/30 shrink-0">
            <svg
              className="w-5 h-5 text-slate-400 shrink-0"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            <input
              autoFocus
              type="text"
              value={query}
              onChange={(e) => {
                setQuery(e.target.value);
                setSelectedIndex(0);
              }}
              onKeyDown={handleInputKeyDown}
              placeholder="Search documentation, guides, architecture, CLI..."
              className="w-full py-4 px-4 bg-transparent text-[15px] sm:text-base text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none"
            />
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="text-[11px] font-mono text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded cursor-pointer transition-colors shadow-2xs"
            >
              ESC
            </button>
          </div>

          {query.trim() && (
            <div className="px-5 py-2 bg-slate-50/90 dark:bg-slate-900/50 border-b border-slate-200/60 dark:border-slate-800/60 text-xs font-medium text-slate-500 dark:text-slate-400 shrink-0 flex items-center justify-between">
              <span>
                <strong>{filteredDocs.length}</strong> results found for &quot;
                <span className="text-slate-800 dark:text-slate-200">{query}</span>
                &quot;
              </span>
              <span className="text-[11px] text-slate-400 uppercase tracking-wider">
                Quick Navigation
              </span>
            </div>
          )}

          <div
            ref={listContainerRef}
            className="overflow-y-auto scroll-smooth will-change-[height]"
            style={{
              height: "auto",
              transition: "height 0.28s cubic-bezier(0.16, 1, 0.3, 1)",
            }}
          >
            <div ref={listContentRef} className="p-3 space-y-1.5">
              {filteredDocs.length > 0 ? (
                filteredDocs.map((doc, idx) => {
                  const isSelected = selectedIndex === idx;
                  return (
                    <div
                      key={`${query.trim()}-${doc.href}`}
                      className="search-cascade-item"
                      style={{ animationDelay: `${Math.min(idx * 28, 180)}ms` }}
                    >
                      <div
                        ref={isSelected ? activeItemRef : null}
                        onClick={() => handleSelect(doc.href)}
                        onMouseEnter={() => setSelectedIndex(idx)}
                        className={`flex items-center justify-between px-4 py-3 rounded-xl cursor-pointer transition-[background-color,border-color,box-shadow] duration-150 ${
                          isSelected
                            ? "bg-blue-50/90 dark:bg-blue-950/40 text-blue-600 dark:text-blue-400 border border-blue-200/80 dark:border-blue-800/60 shadow-2xs"
                            : "text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800/50 border border-transparent"
                        }`}
                      >
                        <div className="flex items-center gap-3.5 min-w-0">
                          <div
                            className={`p-2 rounded-lg shrink-0 transition-colors duration-150 ${
                              isSelected
                                ? "bg-blue-600 text-white shadow-xs"
                                : "bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400"
                            }`}
                          >
                            <svg
                              className="w-4 h-4"
                              fill="none"
                              viewBox="0 0 24 24"
                              stroke="currentColor"
                            >
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={2}
                                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                              />
                            </svg>
                          </div>
                          <div className="truncate">
                            <p className="text-[14px] font-semibold text-slate-900 dark:text-slate-100">
                              <HighlightMatch text={doc.title} query={query} />
                            </p>
                            <div className="flex items-center gap-2 mt-0.5">
                              <span className="text-[11px] font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wider">
                                <HighlightMatch text={doc.category} query={query} />
                              </span>
                              <span className="text-slate-300 dark:text-slate-700">•</span>
                              <span className="text-[11px] text-slate-400 dark:text-slate-500 font-mono truncate">
                                {doc.href}
                              </span>
                            </div>
                          </div>
                        </div>

                        <div className="flex items-center gap-2 shrink-0">
                          {isSelected && (
                            <span className="hidden sm:inline-block text-[11px] font-medium text-blue-600 dark:text-blue-400">
                              Jump to
                            </span>
                          )}
                          <span
                            className={`text-xs px-2 py-0.5 rounded font-mono transition-colors ${
                              isSelected
                                ? "bg-blue-100 dark:bg-blue-900/60 text-blue-600 dark:text-blue-300 font-bold"
                                : "text-slate-400"
                            }`}
                          >
                            ↵
                          </span>
                        </div>
                      </div>
                    </div>
                  );
                })
              ) : (
                <div className="search-cascade-item py-12 text-center text-slate-400">
                  <svg
                    className="w-8 h-8 mx-auto mb-2 text-slate-300 dark:text-slate-600"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={1.5}
                      d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  <p className="text-sm">No matching documentation found for</p>
                  <p className="text-sm font-semibold text-slate-800 dark:text-slate-200 mt-0.5">
                    &quot;{query}&quot;
                  </p>
                </div>
              )}
            </div>
          </div>

          <div className="px-5 py-2.5 bg-slate-50/80 dark:bg-[#0c0d12]/90 border-t border-slate-200/80 dark:border-slate-800 text-xs text-slate-400 flex items-center justify-between shrink-0">
            <div className="flex items-center gap-4">
              <span className="flex items-center gap-1.5">
                <kbd className="font-mono bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 px-1.5 py-0.5 rounded text-[11px] shadow-2xs">↑</kbd>
                <kbd className="font-mono bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 px-1.5 py-0.5 rounded text-[11px] shadow-2xs">↓</kbd>
                <span>navigate</span>
              </span>
              <span className="flex items-center gap-1.5">
                <kbd className="font-mono bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 px-1.5 py-0.5 rounded text-[11px] shadow-2xs">↵</kbd>
                <span>select</span>
              </span>
            </div>
            <span className="flex items-center gap-1.5">
              <kbd className="font-mono bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 px-1.5 py-0.5 rounded text-[11px] shadow-2xs">esc</kbd>
              <span>close</span>
            </span>
          </div>
        </div>
      </div>
    </>
  );
}