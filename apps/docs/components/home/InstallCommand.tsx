"use client";

import { useState } from "react";

const INSTALL_CMD = "curl -fsSL https://budment.com/install.sh | bash";

export default function InstallCommand() {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(INSTALL_CMD);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="flex items-center justify-between border border-slate-300 dark:border-slate-800 bg-slate-50 dark:bg-[#0d0f14] px-3.5 py-2.5 min-w-75 sm:min-w-95 font-mono text-xs">
      <div className="flex items-center gap-2 text-slate-600 dark:text-slate-400 overflow-hidden">
        <span className="text-slate-400 dark:text-slate-600 select-none">$</span>
        <span className="truncate">{INSTALL_CMD}</span>
      </div>
      <button
        type="button"
        onClick={handleCopy}
        className="ml-3 uppercase text-[11px] font-bold text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 cursor-pointer shrink-0"
      >
        {copied ? "COPIED" : "COPY"}
      </button>
    </div>
  );
}