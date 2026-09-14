"use client";

import React, { useEffect, useRef } from "react";

const EXPAND_ICON_SVG = `
  <svg class="w-4 h-4 text-slate-300 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
    <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" />
  </svg>
`;

const COLLAPSE_ICON_SVG = `
  <svg class="w-4 h-4 text-slate-300 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
    <path stroke-linecap="round" stroke-linejoin="round" d="M9 9V4.5M9 9H4.5M9 9L3.75 3.75M9 15v4.5M9 15H4.5M9 15l-5.25 5.25M15 9h4.5M15 9V4.5M15 9l5.25-5.25M15 15h4.5M15 15v4.5m0-4.5l5.25 5.25" />
  </svg>
`;

export default function DocContent({ html }: { html: string }) {
  const containerRef = useRef<HTMLDivElement>(null);

  /**
   * Parses and syntax-highlights ASCII AST trees and execution graphs.
   */
  const setupAstHighlighting = () => {
    if (!containerRef.current) return;
    const astBlocks = containerRef.current.querySelectorAll<HTMLElement>(
      "code.language-ast, code.language-tree, code.language-lifetree",
    );

    astBlocks.forEach((block) => {
      if (block.getAttribute("data-highlighted") === "true") return;

      const raw = block.textContent || "";
      const formatted = raw
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(
          /^([◆▶]\s*.*)$/gm,
          '<span class="text-pink-500 font-bold">$1</span>',
        )
        .replace(
          /\[HTTP\]/g,
          '<span class="text-sky-400 font-semibold">[HTTP]</span>',
        )
        .replace(
          /\bGET\b/g,
          '<span class="text-emerald-400 font-bold">GET</span>',
        )
        .replace(
          /\b(POST|PUT|PATCH)\b/g,
          '<span class="text-amber-400 font-bold">$1</span>',
        )
        .replace(
          /\bDELETE\b/g,
          '<span class="text-rose-500 font-bold">DELETE</span>',
        )
        .replace(
          /\[(BRANCH|MATCH|LOOP|POLL)\]/gi,
          '<span class="text-yellow-400 font-bold">[$1]</span>',
        )
        .replace(
          /\[(TRUE|FALSE)\]/gi,
          '<span class="text-cyan-400 font-semibold">[$1]</span>',
        )
        .replace(
          /(Case:\s*[a-zA-Z0-9_-]+)/g,
          '<span class="text-amber-300 font-medium">$1</span>',
        )
        .replace(
          /\[(BEFORE|AFTER)\]/gi,
          '<span class="text-fuchsia-400 font-semibold">[$1]</span>',
        )
        .replace(
          /→\s*(res_assert|script|req_mutate|barrier|log)/g,
          '→ <span class="text-slate-300 font-medium">$1</span>',
        )
        .replace(
          /(\[id:\s*[^\]]+\])/g,
          '<span class="text-slate-500">$1</span>',
        )
        .replace(/(├──|└──|│)/g, '<span class="text-slate-600">$1</span>');

      block.innerHTML = formatted;
      block.setAttribute("data-highlighted", "true");
      block.classList.add("leading-relaxed", "font-mono");
    });
  };

  /**
   * Initializes Mermaid diagrams with horizontal touch scroll and zoom controls.
   */
  const setupMermaid = async () => {
    if (!containerRef.current) return;
    const mermaidElements = containerRef.current.querySelectorAll<HTMLElement>(
      ".mermaid:not([data-processed='true'])",
    );

    if (mermaidElements.length === 0) return;

    mermaidElements.forEach((el) => {
      el.setAttribute("translate", "no");
      el.classList.add("notranslate");

      if (el.parentElement) {
        el.parentElement.setAttribute("translate", "no");
        el.parentElement.classList.add("notranslate");
      }

      const savedRaw = el.getAttribute("data-raw-code");
      if (savedRaw) {
        el.textContent = savedRaw;
      } else {
        const cleanText = el.textContent?.trim() || "";
        el.setAttribute("data-raw-code", cleanText);
        el.textContent = cleanText;
      }
    });

    const { default: mermaid } = await import("mermaid");
    const isDark = document.documentElement.classList.contains("dark");

    mermaid.initialize({
      startOnLoad: false,
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

    try {
      await mermaid.run({ nodes: Array.from(mermaidElements) });
    } catch (err) {
      console.error("Failed to render Mermaid diagram:", err);
    }

    mermaidElements.forEach((el) => {
      el.setAttribute("data-processed", "true");
      if (el.parentElement?.classList.contains("mermaid-wrapper")) return;

      const wrapper = document.createElement("div");
      // Use overflow-x-auto and touch-pan-x for fluid mobile swiping
      wrapper.className =
        "mermaid-wrapper notranslate relative group mt-8 rounded-2xl bg-white/80 dark:bg-slate-900/60 p-4 shadow-2xs overflow-x-auto touch-pan-x";
      (
        wrapper.style as HTMLElement["style"] & {
          webkitOverflowScrolling?: string;
        }
      ).webkitOverflowScrolling = "touch";
      wrapper.setAttribute("translate", "no");

      el.parentNode?.insertBefore(wrapper, el);
      wrapper.appendChild(el);

      const toolbar = document.createElement("div");
      toolbar.className =
        "sticky top-3 float-right flex items-center gap-1 bg-slate-100/90 dark:bg-slate-800/90 backdrop-blur-sm border border-slate-200 dark:border-slate-700 rounded-lg p-1 opacity-0 group-hover:opacity-100 transition-opacity z-10 text-xs font-mono notranslate";
      toolbar.setAttribute("translate", "no");

      toolbar.innerHTML = `
        <button type="button" class="zoom-out w-7 h-7 flex items-center justify-center hover:bg-slate-200 dark:hover:bg-slate-700 rounded text-slate-600 dark:text-slate-300 font-bold text-base cursor-pointer transition-colors" title="Zoom out">-</button>
        <span class="zoom-level px-1.5 text-slate-500 min-w-11 text-center font-semibold">100%</span>
        <button type="button" class="zoom-in w-7 h-7 flex items-center justify-center hover:bg-slate-200 dark:hover:bg-slate-700 rounded text-slate-600 dark:text-slate-300 font-bold text-base cursor-pointer transition-colors" title="Zoom in">+</button>
        <button type="button" class="zoom-reset w-7 h-7 flex items-center justify-center hover:bg-slate-200 dark:hover:bg-slate-700 rounded text-slate-600 dark:text-slate-300 ml-1 border-l border-slate-300 dark:border-slate-700 text-sm cursor-pointer transition-colors" title="Reset">↺</button>
      `;
      wrapper.appendChild(toolbar);

      let scale = 1;
      const svg = el.querySelector("svg");
      if (svg) {
        svg.style.transition = "transform 0.15s ease-out";
        svg.style.transformOrigin = "center top";
      }

      const updateZoom = (newScale: number) => {
        scale = Math.min(Math.max(newScale, 0.4), 2.5);
        if (svg) svg.style.transform = `scale(${scale})`;
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
    });
  };

  /**
   * Configures terminal expandable blocks while strictly preserving horizontal code scrolling.
   */
  const setupCodeExpand = () => {
    if (!containerRef.current) return;
    const boxes =
      containerRef.current.querySelectorAll<HTMLDivElement>(".terminal-box");

    boxes.forEach((box) => {
      box.setAttribute("translate", "no");
      box.classList.add("notranslate");

      const pre = box.querySelector("pre");
      if (!pre) return;

      // Always guarantee horizontal touch scrolling on mobile
      pre.style.overflowX = "auto";
      (
        pre.style as HTMLElement["style"] & { webkitOverflowScrolling?: string }
      ).webkitOverflowScrolling = "touch";

      if (box.querySelector(".expand-toggle-btn")) return;

      const COLLAPSED_HEIGHT = 320;

      if (pre.scrollHeight > COLLAPSED_HEIGHT + 40) {
        box.classList.add("relative");
        pre.style.maxHeight = `${COLLAPSED_HEIGHT}px`;
        // Only clamp vertical height, NEVER clamp overflow-x
        pre.style.overflowY = "hidden";
        pre.style.transition = "max-height 0.35s cubic-bezier(0.4, 0, 0.2, 1)";

        const fadeOverlay = document.createElement("div");
        fadeOverlay.className =
          "expand-fade absolute bottom-0 left-0 right-0 h-24 bg-gradient-to-t from-[#16181d] via-[#16181d]/80 to-transparent pointer-events-none transition-opacity duration-300 z-1";
        box.appendChild(fadeOverlay);

        const toggleBtn = document.createElement("button");
        toggleBtn.type = "button";
        toggleBtn.className =
          "expand-toggle-btn absolute bottom-3 right-3 z-10 flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-[#212631]/95 hover:bg-[#2c3342] text-slate-200 border border-slate-700/80 hover:border-slate-500 shadow-md hover:shadow-lg backdrop-blur-sm transition-all duration-200 cursor-pointer text-xs font-medium";
        toggleBtn.innerHTML = `
          ${EXPAND_ICON_SVG}
          <span class="btn-text">Expand</span>
        `;
        box.appendChild(toggleBtn);

        const toggleState = () => {
          const isCollapsed = pre.style.maxHeight === `${COLLAPSED_HEIGHT}px`;
          if (isCollapsed) {
            pre.style.maxHeight = `${pre.scrollHeight}px`;
            pre.style.overflowY = "visible";
            fadeOverlay.classList.add("opacity-0");
            toggleBtn.innerHTML = `${COLLAPSE_ICON_SVG}<span class="btn-text">Collapse</span>`;
          } else {
            pre.style.maxHeight = `${COLLAPSED_HEIGHT}px`;
            pre.style.overflowY = "hidden";
            fadeOverlay.classList.remove("opacity-0");
            toggleBtn.innerHTML = `${EXPAND_ICON_SVG}<span class="btn-text">Expand</span>`;
            box.scrollIntoView({ behavior: "smooth", block: "nearest" });
          }
        };

        toggleBtn.addEventListener("click", toggleState);
      }
    });
  };

  /**
   * Wraps markdown tables with a responsive horizontal scroll container.
   */
  const setupTableScroll = () => {
    if (!containerRef.current) return;
    const tables = containerRef.current.querySelectorAll("table");
    tables.forEach((table) => {
      if (table.parentElement?.classList.contains("table-scroll-wrapper"))
        return;
      const wrapper = document.createElement("div");
      wrapper.className =
        "table-scroll-wrapper my-8 w-full max-w-full overflow-x-auto touch-pan-x rounded-xl border border-slate-300 dark:border-slate-800 shadow-xs";
      (
        wrapper.style as HTMLElement["style"] & {
          webkitOverflowScrolling?: string;
        }
      ).webkitOverflowScrolling = "touch";
      table.parentNode?.insertBefore(wrapper, table);
      wrapper.appendChild(table);
    });
  };

  useEffect(() => {
    setupAstHighlighting();
    setupMermaid();
    setupCodeExpand();
    setupTableScroll();

    const handleThemeChange = () => {
      if (containerRef.current) {
        containerRef.current
          .querySelectorAll(".mermaid")
          .forEach((el) => el.removeAttribute("data-processed"));
      }
      setTimeout(setupMermaid, 60);
    };

    window.addEventListener("theme-change", handleThemeChange);
    return () => window.removeEventListener("theme-change", handleThemeChange);
  }, [html]);

  const handleClick = (e: React.MouseEvent<HTMLDivElement>) => {
    const btn = (e.target as HTMLElement).closest(
      ".copy-btn",
    ) as HTMLButtonElement | null;
    if (!btn) return;

    const rawCode = btn.getAttribute("data-code");
    if (!rawCode) return;

    navigator.clipboard.writeText(decodeURIComponent(rawCode)).then(() => {
      const textSpan = btn.querySelector(".copy-text");
      const originalText = textSpan?.textContent || "Copy";
      if (textSpan) textSpan.textContent = "Copied!";
      btn.classList.add("text-emerald-400");

      setTimeout(() => {
        if (textSpan) textSpan.textContent = originalText;
        btn.classList.remove("text-emerald-400");
      }, 2000);
    });
  };

  return (
    <div
      ref={containerRef}
      onClick={handleClick}
      suppressHydrationWarning
      className="prose-content w-full min-w-0 max-w-full wrap-break-word text-[15.5px] leading-[1.8] text-slate-600 dark:text-slate-300 font-sans
        [&_h1]:text-3xl sm:[&_h1]:text-4xl [&_h1]:font-extrabold [&_h1]:tracking-tight [&_h1]:text-slate-900 dark:[&_h1]:text-white [&_h1]:mt-10 [&_h1]:mb-5
        [&_h2]:text-2xl [&_h2]:font-bold [&_h2]:tracking-tight [&_h2]:text-slate-900 dark:[&_h2]:text-slate-100 [&_h2]:mt-12 [&_h2]:mb-4 [&_h2]:border-b [&_h2]:border-slate-200/60 dark:[&_h2]:border-slate-800 [&_h2]:pb-2.5
        [&_h3]:text-lg [&_h3]:font-semibold [&_h3]:text-slate-800 dark:[&_h3]:text-slate-200 [&_h3]:mt-8 [&_h3]:mb-3
        [&_p]:my-4 [&_p]:leading-[1.8]
        [&_strong]:text-slate-900 dark:[&_strong]:text-slate-100
        [&_ul]:my-4 [&_ul]:pl-6 [&_ul]:space-y-2.5
        [&_ol]:my-4 [&_ol]:pl-6 [&_ol]:space-y-2.5
        [&_code]:text-[13.5px] [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:rounded-md [&_code]:bg-slate-100 dark:[&_code]:bg-slate-800 [&_code]:text-slate-800 dark:[&_code]:text-slate-200 [&_code]:border [&_code]:border-slate-200 dark:[&_code]:border-slate-700
        [&_.terminal-box]:w-full [&_.terminal-box]:max-w-full
        [&_.terminal-box_pre]:w-full [&_.terminal-box_pre]:max-w-full [&_.terminal-box_pre]:overflow-x-auto [&_.terminal-box_pre]:touch-pan-x
        [&_.terminal-box_code]:bg-transparent [&_.terminal-box_code]:border-0 [&_.terminal-box_code]:p-0 [&_.terminal-box_code]:text-slate-200
        [&_table]:w-full [&_table]:min-w-140 [&_table]:text-left [&_table]:text-[13.5px] [&_table]:border-separate [&_table]:border-spacing-0
        [&_thead]:bg-slate-200/75 dark:[&_thead]:bg-[#181b22]
        [&_th]:py-3.5 [&_th]:px-4 [&_th]:font-bold [&_th]:text-slate-950 dark:[&_th]:text-white [&_th]:text-[12px] [&_th]:uppercase [&_th]:tracking-wider [&_th]:border-b-2 [&_th]:border-slate-300 dark:[&_th]:border-slate-700 [&_th]:border-r last:[&_th]:border-r-0 [&_th]:whitespace-nowrap
        [&_tbody_tr]:transition-colors [&_tbody_tr]:hover:bg-slate-100/50 dark:[&_tbody_tr]:hover:bg-slate-800/40
        [&_td]:py-3 [&_td]:px-4 [&_td]:border-b [&_td]:border-r last:[&_td]:border-r-0 [&_td]:border-slate-200 dark:[&_td]:border-slate-800 [&_td]:text-slate-700 dark:[&_td]:text-slate-300 [&_td]:leading-relaxed
        [&_tbody_tr:last-child_td]:border-b-0
        [&_blockquote]:border-l-2 [&_blockquote]:border-indigo-500 [&_blockquote]:bg-indigo-50/30 dark:[&_blockquote]:bg-indigo-950/20 [&_blockquote]:py-2.5 [&_blockquote]:px-4 [&_blockquote]:rounded-r-lg [&_blockquote]:text-slate-700 dark:[&_blockquote]:text-slate-300 [&_blockquote]:italic [&_blockquote]:my-5"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
