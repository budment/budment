"use client";

import React, { useEffect, useRef } from "react";

const EXPAND_ICON_SVG = `<svg class="w-4 h-4 text-slate-300 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" /></svg>`;
const COLLAPSE_ICON_SVG = `<svg class="w-4 h-4 text-slate-300 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 9V4.5M9 9H4.5M9 9L3.75 3.75M9 15v4.5M9 15H4.5M9 15l-5.25 5.25M15 9h4.5M15 9V4.5M15 9l5.25-5.25M15 15h4.5M15 15v4.5m0-4.5l5.25 5.25" /></svg>`;

export default function DocContent({ html }: { html: string }) {
  const containerRef = useRef<HTMLDivElement>(null);

  const setupMermaid = async () => {
    if (!containerRef.current) return;
    const blocks = containerRef.current.querySelectorAll<HTMLDivElement>(
      ".mermaid-block:not([data-rendered='true'])",
    );
    if (blocks.length === 0) return;

    const { default: mermaid } = await import("mermaid");
    const isDark = document.documentElement.classList.contains("dark");

    mermaid.initialize({
      startOnLoad: false,
      htmlLabels: true,
      theme: isDark ? "dark" : "neutral",
      securityLevel: "loose",
      fontFamily: "var(--font-sans)",
      themeVariables: isDark
        ? {
            darkMode: true,
            background: "#16181d",
            primaryColor: "#1e293b",
            primaryTextColor: "#f1f5f9",
            lineColor: "#64748b",
          }
        : undefined,
    });

    for (const block of Array.from(blocks)) {
      const rawCode = decodeURIComponent(block.getAttribute("data-code") || "");
      const container =
        block.querySelector<HTMLDivElement>(".mermaid-container");
      if (!rawCode || !container) continue;

      try {
        const id = `mermaid-${Math.random().toString(36).slice(2, 9)}`;
        const { svg } = await mermaid.render(id, rawCode);

        container.innerHTML = svg;
        block.setAttribute("data-rendered", "true");

        const toolbar = document.createElement("div");
        toolbar.className =
          "absolute top-3 right-3 flex items-center gap-1 bg-white/95 dark:bg-slate-800/95 backdrop-blur-md border border-slate-200 dark:border-slate-700 rounded-lg p-1 opacity-0 group-hover:opacity-100 transition-opacity z-10 text-xs font-mono shadow-sm";
        toolbar.innerHTML = `
          <button type="button" class="zoom-out w-7 h-7 flex items-center justify-center hover:bg-slate-100 dark:hover:bg-slate-700 rounded text-slate-700 dark:text-slate-200 font-bold transition-colors cursor-pointer" title="Zoom out">-</button>
          <span class="zoom-level px-1.5 text-slate-600 dark:text-slate-300 min-w-11 text-center font-semibold">100%</span>
          <button type="button" class="zoom-in w-7 h-7 flex items-center justify-center hover:bg-slate-100 dark:hover:bg-slate-700 rounded text-slate-700 dark:text-slate-200 font-bold transition-colors cursor-pointer" title="Zoom in">+</button>
          <button type="button" class="zoom-reset w-7 h-7 flex items-center justify-center hover:bg-slate-100 dark:hover:bg-slate-700 rounded text-slate-700 dark:text-slate-200 ml-1 border-l border-slate-200 dark:border-slate-700 transition-colors cursor-pointer" title="Reset">↺</button>
        `;
        container.appendChild(toolbar);

        let scale = 1;
        const svgEl = container.querySelector("svg");
        if (svgEl) {
          svgEl.style.transition = "transform 0.15s ease-out";
          svgEl.style.transformOrigin = "center top";
        }

        const updateZoom = (newScale: number) => {
          scale = Math.min(Math.max(newScale, 0.4), 2.5);
          if (svgEl) svgEl.style.transform = `scale(${scale})`;
          const label = toolbar.querySelector(".zoom-level");
          if (label) label.textContent = `${Math.round(scale * 100)}%`;
        };

        toolbar
          .querySelector(".zoom-in")
          ?.addEventListener("click", () => updateZoom(scale + 0.2));
        toolbar
          .querySelector(".zoom-out")
          ?.addEventListener("click", () => updateZoom(scale - 0.2));
        toolbar
          .querySelector(".zoom-reset")
          ?.addEventListener("click", () => updateZoom(1));
      } catch (err) {
        console.error("Mermaid Render Error:", err);
      }
    }
  };

  useEffect(() => {
    setupMermaid();

    const handleThemeChange = () => {
      if (containerRef.current) {
        containerRef.current
          .querySelectorAll(".mermaid-block")
          .forEach((el) => el.removeAttribute("data-rendered"));
      }
      setTimeout(setupMermaid, 60);
    };

    window.addEventListener("theme-change", handleThemeChange);
    return () => window.removeEventListener("theme-change", handleThemeChange);
  }, [html]);

  const handleClick = (e: React.MouseEvent<HTMLDivElement>) => {
    const target = e.target as HTMLElement;
    const copyBtn = target.closest(".copy-btn") as HTMLButtonElement | null;
    if (copyBtn) {
      const rawCode = copyBtn.getAttribute("data-code");
      if (!rawCode) return;
      navigator.clipboard.writeText(decodeURIComponent(rawCode)).then(() => {
        const textSpan = copyBtn.querySelector(".copy-text");
        const originalText = textSpan?.textContent || "Copy";
        if (textSpan) textSpan.textContent = "Copied!";
        copyBtn.classList.add("text-emerald-400");
        setTimeout(() => {
          if (textSpan) textSpan.textContent = originalText;
          copyBtn.classList.remove("text-emerald-400");
        }, 2000);
      });
      return;
    }

    const expandBtn = target.closest(
      ".expand-toggle-btn",
    ) as HTMLButtonElement | null;
    if (expandBtn) {
      const box = expandBtn.closest(".terminal-box");
      const pre = box?.querySelector("pre");
      const fade = box?.querySelector(".expand-fade");
      if (!pre) return;

      const isCollapsed = pre.classList.contains("max-h-[320px]");
      if (isCollapsed) {
        pre.classList.remove("max-h-[320px]");
        fade?.classList.add("opacity-0");
        expandBtn.innerHTML = `${COLLAPSE_ICON_SVG}<span class="btn-text">Collapse</span>`;
      } else {
        pre.classList.add("max-h-[320px]");
        fade?.classList.remove("opacity-0");
        expandBtn.innerHTML = `${EXPAND_ICON_SVG}<span class="btn-text">Expand</span>`;
        box?.scrollIntoView({ behavior: "smooth", block: "nearest" });
      }
    }
  };

  return (
    <div
      ref={containerRef}
      onClick={handleClick}
      suppressHydrationWarning
      className="prose-content w-full min-w-0 max-w-full wrap-break-word text-[15.5px] leading-[1.8] text-slate-600 dark:text-slate-300 font-sans
        [&_h1]:text-3xl sm:[&_h1]:text-4xl [&_h1]:font-extrabold [&_h1]:text-slate-900 dark:[&_h1]:text-white [&_h1]:mt-10 [&_h1]:mb-5
        [&_h2]:text-2xl [&_h2]:font-bold [&_h2]:text-slate-900 dark:[&_h2]:text-slate-100 [&_h2]:mt-12 [&_h2]:mb-4 [&_h2]:border-b [&_h2]:border-slate-200/60 dark:[&_h2]:border-slate-800 [&_h2]:pb-2.5
        [&_h3]:text-lg [&_h3]:font-semibold [&_h3]:text-slate-800 dark:[&_h3]:text-slate-200 [&_h3]:mt-8 [&_h3]:mb-3
        [&_p]:my-4 [&_p]:leading-[1.8]
        [&_strong]:text-slate-900 dark:[&_strong]:text-slate-100
        [&_ul]:my-4 [&_ul]:pl-6 [&_ul]:space-y-2.5
        [&_ol]:my-4 [&_ol]:pl-6 [&_ol]:space-y-2.5
        [&_code]:text-[13.5px] [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:rounded-md [&_code]:bg-slate-100 dark:[&_code]:bg-slate-800 [&_code]:text-slate-800 dark:[&_code]:text-slate-200
        [&_.terminal-box]:w-full [&_.terminal-box]:max-w-full [&_.terminal-box]:bg-[#16181d]
        [&_.terminal-box_pre]:w-full [&_.terminal-box_pre]:max-w-full [&_.terminal-box_pre]:bg-[#16181d] [&_.terminal-box_pre]:overflow-x-auto
        [&_.terminal-box_code]:bg-transparent! [&_.terminal-box_code]:border-0! [&_.terminal-box_code]:p-0! [&_.terminal-box_code]:text-slate-200!
        [&_table]:block [&_table]:w-full [&_table]:overflow-x-auto [&_table]:my-8 [&_table]:text-left [&_table]:text-[13.5px] [&_table]:border-collapse [&_table]:rounded-xl [&_table]:border [&_table]:border-slate-300 dark:[&_table]:border-slate-800 [&_table]:shadow-2xs
        [&_thead]:bg-slate-100 dark:[&_thead]:bg-[#181b22]
        [&_th]:py-3.5 [&_th]:px-4 [&_th]:font-bold [&_th]:text-slate-950 dark:[&_th]:text-white [&_th]:text-[12px] [&_th]:uppercase [&_th]:tracking-wider [&_th]:border-b-2 [&_th]:border-slate-300 dark:[&_th]:border-slate-700 [&_th]:whitespace-nowrap
        [&_tbody_tr]:transition-colors [&_tbody_tr]:hover:bg-slate-100/50 dark:[&_tbody_tr]:hover:bg-slate-800/40
        [&_td]:py-3 [&_td]:px-4 [&_td]:border-b [&_td]:border-slate-200 dark:[&_td]:border-slate-800 [&_td]:text-slate-700 dark:[&_td]:text-slate-300 [&_td]:leading-relaxed
        [&_tbody_tr:last-child_td]:border-b-0
        [&_blockquote]:border-l-2 [&_blockquote]:border-indigo-500 [&_blockquote]:bg-indigo-50/30 dark:[&_blockquote]:bg-indigo-950/20 [&_blockquote]:py-2.5 [&_blockquote]:px-4 [&_blockquote]:rounded-r-lg [&_blockquote]:text-slate-700 dark:[&_blockquote]:text-slate-300 [&_blockquote]:italic [&_blockquote]:my-5"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
