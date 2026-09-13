import Link from "next/link";

export default function Footer() {
  return (
    <footer className="w-full border-t border-slate-200/80 dark:border-slate-800/80 bg-white/70 dark:bg-[#090a0f]/80 backdrop-blur-md transition-colors mt-auto">
      <div className="w-full px-6 sm:px-10 lg:px-16 py-14">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-10 lg:gap-12 mb-12">
          {/* Column 1 & 2: Platform Info (occupies 2 columns) */}
          <div className="lg:col-span-2 space-y-4">
            <div className="flex items-center gap-2.5">
              <span className="font-mono font-bold text-xs tracking-wider text-white bg-blue-600 px-2.5 py-1 rounded-md shadow-xs">
                BUDMENT
              </span>
              <span className="text-sm font-semibold text-slate-600 dark:text-slate-300">
                Execution Engine
              </span>
            </div>
            <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed max-w-md">
              An open-source declarative API execution engine. Compiles
              TypeScript workflows into deterministic execution plans for load
              testing.
            </p>
          </div>

          {/* Column 3: Documentation */}
          <div className="space-y-3.5">
            <div className="font-bold text-slate-900 dark:text-slate-100 uppercase tracking-wider text-[12px]">
              Documentation
            </div>
            <ul className="space-y-2.5 text-sm text-slate-600 dark:text-slate-400">
              <li>
                <Link
                  href="/docs/getting-started/01-introduction"
                  className="hover:text-slate-950 dark:hover:text-white transition-colors"
                >
                  Introduction
                </Link>
              </li>
              <li>
                <Link
                  href="/docs/getting-started/02-installation"
                  className="hover:text-slate-950 dark:hover:text-white transition-colors"
                >
                  Installation
                </Link>
              </li>
              <li>
                <Link
                  href="/docs/getting-started/03-quickstart"
                  className="hover:text-slate-950 dark:hover:text-white transition-colors"
                >
                  Quickstart Guide
                </Link>
              </li>
              <li>
                <Link
                  href="/docs/guide/01-lifecycle"
                  className="hover:text-slate-950 dark:hover:text-white transition-colors"
                >
                  Lifecycle DAG
                </Link>
              </li>
              <li>
                <Link
                  href="/docs/reference/cli"
                  className="hover:text-slate-950 dark:hover:text-white transition-colors"
                >
                  CLI Reference
                </Link>
              </li>
            </ul>
          </div>

          {/* Column 4: Ecosystem & Community */}
          <div className="space-y-3.5">
            <div className="font-bold text-slate-900 dark:text-slate-100 uppercase tracking-wider text-[12px]">
              Ecosystem
            </div>
            <ul className="space-y-2.5 text-sm text-slate-600 dark:text-slate-400">
              <li>
                <a
                  href="https://github.com/budment/budment"
                  target="_blank"
                  rel="noreferrer"
                  className="hover:text-slate-950 dark:hover:text-white transition-colors flex items-center gap-1.5"
                >
                  GitHub Repository ↗
                </a>
              </li>
              <li>
                <a
                  href="https://github.com/budment/budment/releases"
                  target="_blank"
                  rel="noreferrer"
                  className="hover:text-slate-950 dark:hover:text-white transition-colors flex items-center gap-1.5"
                >
                  Releases & Changelog ↗
                </a>
              </li>
              <li>
                <a
                  href="https://github.com/budment/budment/issues"
                  target="_blank"
                  rel="noreferrer"
                  className="hover:text-slate-950 dark:hover:text-white transition-colors flex items-center gap-1.5"
                >
                  Issues & Discussions ↗
                </a>
              </li>
            </ul>
          </div>

          {/* Column 5: Open Source & License */}
          <div className="space-y-3.5">
            <div className="font-bold text-slate-900 dark:text-slate-100 uppercase tracking-wider text-[12px]">
              Open Source
            </div>
            <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed">
              Released under the{" "}
              <strong className="text-slate-900 dark:text-slate-200">
                MIT License
              </strong>
              . Community contributions and
              feedback are welcome.
            </p>
          </div>
        </div>

        <div className="pt-8 border-t border-slate-200/80 dark:border-slate-800/80 flex flex-col sm:flex-row items-center justify-between gap-4 text-sm text-slate-500 dark:text-slate-400">
          <p>© 2026 Budment Project by vunas. All rights reserved.</p>
          <div className="flex items-center gap-6 font-medium">
            <Link
              href="/"
              className="hover:text-slate-900 dark:hover:text-white transition-colors"
            >
              Home
            </Link>
            <Link
              href="/docs/getting-started/01-introduction"
              className="hover:text-slate-900 dark:hover:text-white transition-colors"
            >
              Docs
            </Link>
            <a
              href="https://github.com/budment/budment/blob/main/LICENSE"
              target="_blank"
              rel="noreferrer"
              className="hover:text-slate-900 dark:hover:text-white transition-colors"
            >
              License
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
