import type { Metadata } from "next";
import Link from "next/link";
import Header from "@/components/Header";
import Footer from "@/components/Footer";
import InstallCommand from "@/components/home/InstallCommand";
import HeroAnimation from "@/components/home/HeroAnimation";

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
        {/* HERO SECTION */}
        <section className="pt-10 sm:pt-16 pb-14 sm:pb-20 px-4 sm:px-6 max-w-7xl mx-auto border-b border-slate-200 dark:border-slate-800/80">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-12 items-center">
            {/* HERO CONTENT */}
            <div className="lg:col-span-7 flex flex-col justify-center text-left">
              <span className="font-mono text-xs uppercase tracking-[0.2em] text-slate-400 dark:text-slate-500 font-semibold mb-3.5 block">
                Open-Source API Execution Engine
              </span>

              <h1 className="text-4xl sm:text-6xl lg:text-[64px] font-light tracking-[-0.035em] text-slate-950 dark:text-white leading-[1.05] mb-6">
                Scenarios as{" "}
                <span className="font-semibold bg-linear-to-r bg-blue-600 bg-clip-text text-transparent">
                  AST.
                </span>
                <br />
                <span className="font-semibold text-slate-900 dark:text-slate-100">
                  Logic as Code.
                </span>
              </h1>

              <p className="text-base sm:text-lg text-slate-600 dark:text-slate-400 font-normal leading-relaxed mb-8 max-w-xl">
                Budment separates scenario compilation from runtime execution.
                Define workflows in TypeScript, compile into an immutable
                Abstract Syntax Tree (AST).
              </p>

              <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3.5 font-mono text-xs">
                <Link
                  href="/docs/getting-started/01-introduction"
                  className="px-6 py-3 bg-slate-950 hover:bg-slate-800 dark:bg-white dark:text-slate-950 dark:hover:bg-slate-200 text-white font-semibold text-center transition-colors shadow-2xs shrink-0"
                >
                  Get Started
                </Link>

                <InstallCommand />
              </div>
            </div>

            {/* Right side: Animation (5 cols) */}
            <div className="lg:col-span-5 w-full flex items-center justify-center lg:justify-end">
              <HeroAnimation src="/Ship.lottie" />
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
                Scenarios are evaluated once in an isolated compiler and
                compiled into an immutable Abstract Syntax Tree (AST) with no
                network or external side effects.
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
                Primitives automatically adapt to their execution context. A
                lightweight engine architecture minimizes runtime overhead,
                invoking script runtimes only when custom hooks require them.
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
                Author workflows as type-safe code with Git and IDE support. The{" "}
                <code className="font-mono text-slate-900 dark:text-slate-100 bg-slate-100 dark:bg-slate-800 px-1 py-0.5">
                  budment plan
                </code>{" "}
                command renders inspectable execution plans before sending
                network traffic.
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
