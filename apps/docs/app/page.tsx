import type { Metadata } from "next";
import Link from "next/link";
import Header from "@/components/Header";
import Footer from "@/components/Footer";
import InstallCommand from "@/components/home/InstallCommand";

export const metadata: Metadata = {
  title: "Budment — Scenarios as AST, Logic as Code",
  description:
    "Open-source API execution engine. Define workflows as code and compile them into deterministic execution plans.",
  alternates: {
    canonical: "/",
  },
};

export default function HomePage() {
  return (
    <div className="min-h-screen flex flex-col bg-white dark:bg-[#07080b] text-slate-900 dark:text-slate-100 font-sans selection:bg-blue-600 selection:text-white transition-colors duration-150">
      <Header />

      <main className="flex-1">
        {/* HERO */}
        <section className="pt-20 pb-16 px-6 max-w-7xl mx-auto border-b border-slate-200 dark:border-slate-800/80">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 items-center">
            {/* Heading & Action */}
            <div className="lg:col-span-7">
              <span className="font-mono text-xs uppercase tracking-widest text-slate-400 dark:text-slate-500 font-semibold mb-4 block">
                Open-Source API Execution Engine
              </span>

              <h1 className="text-4xl sm:text-6xl lg:text-7xl font-light tracking-tight text-slate-950 dark:text-white leading-[1.04] mb-6">
                Scenarios as AST.
                <br />
                Logic as Code.
              </h1>

              <p className="text-base sm:text-lg text-slate-600 dark:text-slate-400 font-normal leading-relaxed max-w-xl mb-8">
                Budment uses a two-phase execution model that separates scenario compilation from runtime execution. Define workflows in TypeScript, compile into an immutable Abstract Syntax Tree (AST), and run high-concurrency workloads cleanly.
              </p>

              <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 font-mono text-xs">
                <Link
                  href="/docs/getting-started/01-introduction"
                  className="px-5 py-3 bg-slate-950 hover:bg-slate-800 dark:bg-white dark:text-slate-950 dark:hover:bg-slate-200 text-white font-semibold text-center transition-colors shadow-2xs"
                >
                  Get Started
                </Link>

                <InstallCommand />
              </div>
            </div>

            {/* Architecture HUD */}
            <div className="lg:col-span-5 font-mono text-xs">
              <div className="border border-slate-200 dark:border-slate-800 bg-slate-50/70 dark:bg-[#0c0e14] p-5 shadow-2xs">
                <div className="flex items-center justify-between pb-3 mb-4 border-b border-slate-200 dark:border-slate-800 text-slate-400 text-[11px]">
                  <span className="flex items-center gap-2">
                    <span className="w-2 h-2 bg-emerald-500 rounded-full animate-pulse" />
                    RUNTIME DISPATCH
                  </span>
                  <span>v0.4 · READY</span>
                </div>

                <div className="space-y-3">
                  <div className="flex justify-between items-center py-1.5 border-b border-slate-200/60 dark:border-slate-800/60">
                    <span className="text-slate-500">Execution Model</span>
                    <span className="text-slate-900 dark:text-slate-200 font-semibold">Two-Phase (Plan → Run)</span>
                  </div>
                  <div className="flex justify-between items-center py-1.5 border-b border-slate-200/60 dark:border-slate-800/60">
                    <span className="text-slate-500">Compilation Target</span>
                    <span className="text-blue-600 dark:text-blue-400">Immutable AST Nodes</span>
                  </div>
                  <div className="flex justify-between items-center py-1.5 border-b border-slate-200/60 dark:border-slate-800/60">
                    <span className="text-slate-500">Runtime Hooks</span>
                    <span className="text-purple-600 dark:text-purple-400">On-Demand Invocation</span>
                  </div>
                  <div className="flex justify-between items-center py-1.5 border-b border-slate-200/60 dark:border-slate-800/60">
                    <span className="text-slate-500">Data Expressions</span>
                    <span className="text-emerald-600 dark:text-emerald-400">Dynamic DSL Primitives</span>
                  </div>
                  <div className="flex justify-between items-center py-1.5">
                    <span className="text-slate-500">Verification</span>
                    <span className="text-slate-900 dark:text-slate-200">Deterministic CI/CD Thresholds</span>
                  </div>
                </div>

                <div className="mt-4 pt-3 border-t border-slate-200 dark:border-slate-800 text-[10px] text-slate-400 flex items-center justify-between">
                  <span>Deterministic Execution Graph</span>
                  <span className="text-slate-500">AUDITABLE</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* THREE TECHNICAL PILLARS */}
        <section className="border-b border-slate-200 dark:border-slate-800/80">
          <div className="max-w-7xl mx-auto grid grid-cols-1 md:grid-cols-3 divide-y md:divide-y-0 md:divide-x divide-slate-200 dark:divide-slate-800/80">
            <div className="p-8 lg:p-12">
              <span className="font-mono text-xs text-slate-400 dark:text-slate-500 block mb-4 uppercase tracking-wider">
                01 / ARCHITECTURE
              </span>
              <h2 className="text-xl font-bold text-slate-950 dark:text-white tracking-tight mb-3">
                Two-Phase Execution
              </h2>
              <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed font-normal">
                Scenarios are evaluated once in an isolated compiler and compiled into an immutable Abstract Syntax Tree (AST) with no network or external side effects.
              </p>
            </div>

            <div className="p-8 lg:p-12">
              <span className="font-mono text-xs text-slate-400 dark:text-slate-500 block mb-4 uppercase tracking-wider">
                02 / RESOURCE EFFICIENCY
              </span>
              <h2 className="text-xl font-bold text-slate-950 dark:text-white tracking-tight mb-3">
                Context-Aware SDK
              </h2>
              <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed font-normal">
                Primitives automatically adapt to their execution context. A lightweight engine architecture minimizes runtime overhead, invoking script runtimes only when custom hooks require them.
              </p>
            </div>

            <div className="p-8 lg:p-12">
              <span className="font-mono text-xs text-slate-400 dark:text-slate-500 block mb-4 uppercase tracking-wider">
                03 / AUDITABILITY
              </span>
              <h2 className="text-xl font-bold text-slate-950 dark:text-white tracking-tight mb-3">
                Test as Code &amp; Docs
              </h2>
              <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed font-normal">
                Author workflows as type-safe code with Git and IDE support. The <code className="font-mono text-slate-900 dark:text-slate-100 bg-slate-100 dark:bg-slate-800 px-1 py-0.5">budment plan</code> command renders inspectable execution plans before sending network traffic.
              </p>
            </div>
          </div>
        </section>

        {/* FOUR CONCRETE PILLARS */}
        <section className="py-16 px-6 max-w-7xl mx-auto">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8 font-mono">
            <div className="border-l-2 border-slate-200 dark:border-slate-800 pl-4">
              <span className="text-2xl sm:text-3xl font-bold text-slate-950 dark:text-white block tracking-tight mb-1">
                Isolated
              </span>
              <span className="text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Zero Side-Effect Compile
              </span>
            </div>

            <div className="border-l-2 border-slate-200 dark:border-slate-800 pl-4">
              <span className="text-2xl sm:text-3xl font-bold text-slate-950 dark:text-white block tracking-tight mb-1">
                Type-Safe
              </span>
              <span className="text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                TypeScript Scenario DSL
              </span>
            </div>

            <div className="border-l-2 border-slate-200 dark:border-slate-800 pl-4">
              <span className="text-2xl sm:text-3xl font-bold text-slate-950 dark:text-white block tracking-tight mb-1">
                Dry-Run
              </span>
              <span className="text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Inspectable AST Plan
              </span>
            </div>

            <div className="border-l-2 border-slate-200 dark:border-slate-800 pl-4">
              <span className="text-2xl sm:text-3xl font-bold text-slate-950 dark:text-white block tracking-tight mb-1">
                CI/CD
              </span>
              <span className="text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Deterministic Thresholds
              </span>
            </div>
          </div>
        </section>
      </main>

      <Footer />
    </div>
  );
}
