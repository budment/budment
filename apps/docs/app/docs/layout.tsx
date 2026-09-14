import React from "react";
import Header from "@/components/Header";
import Sidebar from "@/components/Sidebar";
import Footer from "@/components/Footer";

export default function DocsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-[#fbfbfa] dark:bg-[#0c0d12] text-slate-800 dark:text-slate-200 flex flex-col font-sans transition-colors">
      <Header />
      <div className="flex-1 w-full flex">
        <Sidebar />
        <main className="flex-1 min-w-0 py-8 sm:py-10 px-4 sm:px-10 lg:px-14 flex justify-center">
          <div className="w-full max-w-6xl">
            {children}
          </div>
        </main>
      </div>
      {/* Global Page Footer */}
      <Footer />
    </div>
  );
}