"use client";

import React, { useState, useSyncExternalStore } from "react";
import Link from "next/link";
import Image from "next/image";
import packageJson from "@/package.json";
import ThemeToggle from "./ThemeToggle";
import SearchDialog from "./SearchDialog";

const subscribeOS = () => () => {};
const getIsMac = () =>
  typeof navigator !== "undefined" &&
  navigator.userAgent.toUpperCase().includes("MAC");
const getServerIsMac = () => false;

export default function Header() {
  const [searchOpen, setSearchOpen] = useState(false);
  const isMac = useSyncExternalStore(subscribeOS, getIsMac, getServerIsMac);
  const currentVersion = `v${packageJson.version || "0.0.0"}`;

  return (
    <>
      <header className="sticky top-0 z-40 w-full border-b border-slate-200/80 dark:border-slate-800/80 bg-white/90 dark:bg-[#0c0d12]/90 backdrop-blur-md transition-colors">
        <div className="w-full px-4 sm:px-6 h-14 flex items-center justify-between gap-4">
          {/* Brand & navigation */}
          <div className="flex items-center gap-3.5 shrink-0">
            <Link href="/" className="flex items-center gap-2 p-0 m-0 group">
              {/* Scaled container to offset internal SVG padding */}
              <div className="w-8.5 h-8.5 p-0 m-0 shrink-0 flex items-center justify-center">
                <Image
                  src="/logo.svg"
                  alt="Budment Logo"
                  width={32}
                  height={32}
                  className="w-full h-full object-contain scale-125 transition-transform group-hover:scale-130"
                  priority
                />
              </div>

              <div className="flex items-baseline gap-1.5">
                <span className="font-mono font-bold text-sm tracking-wider text-slate-900 dark:text-white">
                  BUDMENT
                </span>
                <span className="text-xs font-semibold text-slate-400 dark:text-slate-500">
                  /docs
                </span>
              </div>
            </Link>

            <a
              href="https://github.com/budment/budment/releases"
              target="_blank"
              rel="noreferrer"
              title="Release notes"
              className="hidden sm:inline-block font-mono text-[11px] font-semibold text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200 bg-slate-100 dark:bg-slate-800/80 px-1.5 py-0.5 rounded-none transition-colors"
            >
              {currentVersion}
            </a>

            <nav className="hidden lg:flex items-center gap-4 text-xs font-medium text-slate-600 dark:text-slate-300 ml-1">
              <Link
                href="/"
                className="hover:text-slate-950 dark:hover:text-white transition-colors"
              >
                Home
              </Link>
              <Link
                href="/docs/guide/01-lifecycle"
                className="hover:text-slate-950 dark:hover:text-white transition-colors"
              >
                Guide
              </Link>
              <Link
                href="/docs/reference/cli"
                className="hover:text-slate-950 dark:hover:text-white transition-colors"
              >
                CLI Spec
              </Link>
            </nav>
          </div>

          {/* Desktop search trigger */}
          <div className="flex-1 min-w-0 hidden sm:flex">
            <button
              type="button"
              onClick={() => setSearchOpen(true)}
              className="w-full h-9 px-3.5 flex items-center gap-3 text-xs text-slate-400 bg-slate-100/70 dark:bg-slate-900/60 hover:bg-slate-100 dark:hover:bg-slate-900 border border-slate-200/80 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700 rounded-none transition-all cursor-pointer text-left group"
            >
              <svg
                className="w-5 h-5 text-slate-400 group-hover:text-blue-500 transition-colors shrink-0"
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
              <span className="flex-1 text-slate-500 dark:text-slate-400 truncate">
                Search documentation, CLI flags, DAG...
              </span>
              <kbd className="font-mono text-[10px] font-medium bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700/80 px-1.5 py-0.5 rounded-none text-slate-500 dark:text-slate-400 shrink-0">
                {isMac ? "⌘K" : "Ctrl K"}
              </kbd>
            </button>
          </div>

          {/* Right actions */}
          <div className="flex items-center gap-2 shrink-0">
            {/* Mobile search button */}
            <button
              type="button"
              onClick={() => setSearchOpen(true)}
              className="sm:hidden p-2 text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white rounded-none hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors cursor-pointer"
              aria-label="Search"
            >
              <svg
                className="w-5 h-5"
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
            </button>

            {/* GitHub repository */}
            <a
              href="https://github.com/budment/budment"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1.5 h-9 px-2.5 text-xs font-medium text-slate-600 dark:text-slate-300 hover:text-slate-950 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800/70 rounded-none transition-colors"
            >
              <svg
                className="w-4.5 h-4.5 fill-current shrink-0"
                viewBox="0 0 16 16"
              >
                <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z" />
              </svg>
              <span className="hidden sm:inline">GitHub</span>
              <span className="text-[10px] text-slate-400 opacity-60">↗</span>
            </a>

            <div className="w-px h-3.5 bg-slate-200 dark:bg-slate-800 mx-0.5" />

            <ThemeToggle />
          </div>
        </div>
      </header>

      <SearchDialog open={searchOpen} setOpen={setSearchOpen} />
    </>
  );
}
