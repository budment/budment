"use client";

import { useEffect, useState, useRef, useCallback } from "react";

export interface HeadingItem {
  id: string;
  text: string;
  level: number;
}

interface RightSidebarProps {
  headings: HeadingItem[];
  rawMarkdown: string;
}

interface Point {
  x: number;
  y: number;
}

export default function RightSidebar({
  headings,
  rawMarkdown,
}: RightSidebarProps) {
  const [prevHeadings, setPrevHeadings] = useState(headings);
  const [userActiveId, setUserActiveId] = useState<string | null>(null);

  // Sync active heading state during navigation without effects
  if (headings !== prevHeadings) {
    setPrevHeadings(headings);
    setUserActiveId(null);
  }

  const activeId = userActiveId ?? headings[0]?.id ?? "";

  const [copied, setCopied] = useState(false);
  const [aiFeedback, setAiFeedback] = useState<string | null>(null);

  const copyTimerRef = useRef<NodeJS.Timeout | null>(null);
  const feedbackTimerRef = useRef<NodeJS.Timeout | null>(null);

  const listRef = useRef<HTMLUListElement>(null);
  const itemRefs = useRef<Record<string, HTMLLIElement | null>>({});

  const svgRef = useRef<SVGSVGElement>(null);
  const trackRef = useRef<SVGPathElement>(null);
  const progressPathRef = useRef<SVGPathElement>(null);
  const activeDotRef = useRef<SVGCircleElement>(null);
  const activeDotGlowRef = useRef<SVGCircleElement>(null);
  const lastDrawnHeadingsRef = useRef<HeadingItem[]>([]);

  const headingsRef = useRef(headings);
  const activeIdRef = useRef(activeId);

  useEffect(() => {
    return () => {
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current);
      if (feedbackTimerRef.current) clearTimeout(feedbackTimerRef.current);
    };
  }, []);

  const updateIndicator = useCallback(
    (
      currentHeadings: HeadingItem[],
      currentActiveId: string,
      animate = true,
    ) => {
      const list = listRef.current;
      const svg = svgRef.current;
      const track = trackRef.current;
      const progress = progressPathRef.current;
      const dot = activeDotRef.current;
      const glow = activeDotGlowRef.current;

      if (
        !list ||
        !svg ||
        !track ||
        !progress ||
        !dot ||
        currentHeadings.length === 0
      ) {
        if (svg) svg.style.display = "none";
        return;
      }

      const listRect = list.getBoundingClientRect();
      const points: Point[] = [];

      currentHeadings.forEach((item) => {
        const el = itemRefs.current[item.id];
        if (!el) return;
        const rect = el.getBoundingClientRect();
        const depth = item.level === 3 ? 1 : 0;

        points.push({
          x: 4 + depth * 7,
          y: rect.top - listRect.top + rect.height / 2,
        });
      });

      if (points.length === 0) {
        svg.style.display = "none";
        return;
      }

      const height = Math.max(
        list.scrollHeight,
        points[points.length - 1].y + 10,
      );

      let pathData = `M ${points[0].x} 0 L ${points[0].x} ${points[0].y}`;
      for (let i = 1; i < points.length; i++) {
        const prev = points[i - 1];
        const curr = points[i];

        if (prev.x === curr.x) {
          pathData += ` L ${curr.x} ${curr.y}`;
        } else {
          const midY = (prev.y + curr.y) / 2;
          pathData +=
            ` L ${prev.x} ${midY - 5}` +
            ` C ${prev.x} ${midY - 2}, ${curr.x} ${midY + 2}, ${curr.x} ${midY + 5}` +
            ` L ${curr.x} ${curr.y}`;
        }
      }
      pathData += ` L ${points[points.length - 1].x} ${height}`;

      svg.setAttribute("height", String(height));
      svg.setAttribute("viewBox", `0 0 18 ${height}`);
      track.setAttribute("d", pathData);
      progress.setAttribute("d", pathData);

      const activeIndex = Math.max(
        0,
        currentHeadings.findIndex((h) => h.id === currentActiveId),
      );
      const targetPoint = points[activeIndex] || points[0];

      const totalLength = progress.getTotalLength();
      let low = 0;
      let high = totalLength;

      for (let i = 0; i < 16; i++) {
        const mid = (low + high) / 2;
        const pt = progress.getPointAtLength(mid);
        if (pt.y < targetPoint.y) {
          low = mid;
        } else {
          high = mid;
        }
      }
      const activeLength = (low + high) / 2;

      if (!animate) {
        progress.style.transition = "none";
        dot.style.transition = "none";
        if (glow) glow.style.transition = "none";
      } else {
        progress.style.transition = "stroke-dashoffset 0.22s ease-out";
        dot.style.transition = "all 0.22s ease-out";
        if (glow) glow.style.transition = "all 0.22s ease-out";
      }

      progress.style.strokeDasharray = String(totalLength);
      progress.style.strokeDashoffset = String(totalLength - activeLength);

      dot.setAttribute("cx", String(targetPoint.x));
      dot.setAttribute("cy", String(targetPoint.y));
      if (glow) {
        glow.setAttribute("cx", String(targetPoint.x));
        glow.setAttribute("cy", String(targetPoint.y));
      }

      svg.style.display = "block";

      if (!animate) {
        requestAnimationFrame(() => {
          progress.style.transition = "stroke-dashoffset 0.22s ease-out";
          dot.style.transition = "all 0.22s ease-out";
          if (glow) glow.style.transition = "all 0.22s ease-out";
        });
      }
    },
    [],
  );

  // Track heading đang hiển thị qua IntersectionObserver
  useEffect(() => {
    if (headings.length === 0) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setUserActiveId(entry.target.id);
          }
        });
      },
      { rootMargin: "-80px 0% -60% 0%" },
    );

    headings.forEach((item) => {
      const el = document.getElementById(item.id);
      if (el) observer.observe(el);
    });

    return () => observer.disconnect();
  }, [headings]);

  useEffect(() => {
    headingsRef.current = headings;
    activeIdRef.current = activeId;

    const isPageSwitch = headings !== lastDrawnHeadingsRef.current;
    lastDrawnHeadingsRef.current = headings;

    updateIndicator(headings, activeId, !isPageSwitch);
  }, [activeId, headings, updateIndicator]);

  useEffect(() => {
    if (!listRef.current) return;
    const ro = new ResizeObserver(() => {
      updateIndicator(headingsRef.current, activeIdRef.current, false);
    });
    ro.observe(listRef.current);
    return () => ro.disconnect();
  }, [updateIndicator]);

  const handleCopyMarkdown = () => {
    navigator.clipboard.writeText(rawMarkdown).then(() => {
      setCopied(true);
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current);
      copyTimerRef.current = setTimeout(() => setCopied(false), 2000);
    });
  };

  const handleOpenAI = (platform: "chatgpt" | "claude") => {
    if (typeof window === "undefined") return;

    const cleanUrl = `${window.location.origin}${window.location.pathname}${window.location.search}`;
    const prompt = `Please review and provide a concise, technical breakdown of this documentation: ${cleanUrl}`;

    setAiFeedback(
      `Opening ${platform === "chatgpt" ? "ChatGPT" : "Claude"}...`,
    );
    if (feedbackTimerRef.current) clearTimeout(feedbackTimerRef.current);
    feedbackTimerRef.current = setTimeout(() => setAiFeedback(null), 2500);

    const targetUrl =
      platform === "chatgpt"
        ? `https://chatgpt.com/?q=${encodeURIComponent(prompt)}`
        : `https://claude.ai/new?q=${encodeURIComponent(prompt)}`;

    window.open(targetUrl, "_blank", "noopener,noreferrer");
  };

  const handleViewRaw = () => {
    const blob = new Blob([rawMarkdown], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    window.open(url, "_blank", "noopener,noreferrer");

    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };

  return (
    <aside
      aria-label="Table of contents"
      className="space-y-7 py-4 pr-4 text-[14px] font-sans select-none text-slate-600 dark:text-slate-400"
    >
      {headings.length > 0 && (
        <div>
          <div className="font-bold text-slate-900 dark:text-slate-100 text-[12px] uppercase tracking-wider mb-3.5">
            On this page
          </div>

          <div className="relative pl-6">
            <svg
              ref={svgRef}
              aria-hidden="true"
              width="18"
              className="absolute left-0 top-0 pointer-events-none overflow-visible"
              style={{ display: "none" }}
            >
              <path
                ref={trackRef}
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
                className="text-slate-200 dark:text-slate-800"
              />
              <path
                ref={progressPathRef}
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                className="text-blue-600 dark:text-blue-500"
              />
              <circle
                ref={activeDotRef}
                r="3"
                className="fill-blue-600 dark:fill-blue-400"
              />
              <circle
                ref={activeDotGlowRef}
                r="6"
                className="fill-blue-500/20 dark:fill-blue-400/20"
              />
            </svg>

            <ul ref={listRef} className="space-y-3">
              {headings.map((item) => {
                const isActive = activeId === item.id;
                return (
                  <li
                    key={item.id}
                    ref={(el) => {
                      itemRefs.current[item.id] = el;
                    }}
                    className={`transition-all duration-200 ${
                      item.level === 3 ? "pl-2 text-[13px]" : "text-[14px]"
                    }`}
                  >
                    <a
                      href={`#${item.id}`}
                      className={`block leading-relaxed transition-colors ${
                        isActive
                          ? "text-slate-950 dark:text-white font-semibold"
                          : "text-slate-600 dark:text-slate-400 hover:text-slate-950 dark:hover:text-slate-100"
                      }`}
                    >
                      {item.text}
                    </a>
                  </li>
                );
              })}
            </ul>
          </div>
        </div>
      )}

      <div className="border-t border-slate-200/80 dark:border-slate-800" />

      <div>
        <div className="font-bold text-slate-900 dark:text-slate-100 text-[12px] uppercase tracking-wider mb-3.5 flex items-center justify-between">
          <span>Ask AI about this page</span>
        </div>

        {aiFeedback && (
          <div className="text-[12px] text-emerald-600 dark:text-emerald-400 mb-2 font-medium">
            {aiFeedback}
          </div>
        )}

        <ul className="space-y-1.5 text-slate-600 dark:text-slate-400">
          <li>
            <button
              type="button"
              onClick={handleCopyMarkdown}
              className="w-full flex items-center justify-between px-2.5 py-2 rounded-lg hover:bg-slate-100/80 dark:hover:bg-slate-800/60 hover:text-slate-900 dark:hover:text-slate-100 transition-colors cursor-pointer text-left"
            >
              <span className="flex items-center gap-2.5">
                <svg
                  className="w-4 h-4 text-slate-500 dark:text-slate-400 shrink-0"
                  viewBox="0 0 16 16"
                  fill="currentColor"
                >
                  <path d="M14 3a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h12zM2 2a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H2z" />
                  <path d="M3.5 11V5h1.5l1.5 2 1.5-2H9.5v6H8V7.8L6.5 9.8 5 7.8V11H3.5zm7.5-2h1V6h1.5v3h1l-1.75 2.5L11 9z" />
                </svg>
                <span className="text-[13.5px] font-medium">
                  {copied ? "Copied! ✓" : "Copy Markdown"}
                </span>
              </span>
            </button>
          </li>

          <li>
            <button
              type="button"
              onClick={() => handleOpenAI("chatgpt")}
              className="w-full flex items-center justify-between px-2.5 py-2 rounded-lg hover:bg-slate-100/80 dark:hover:bg-slate-800/60 hover:text-slate-900 dark:hover:text-slate-100 transition-colors cursor-pointer text-left"
            >
              <span className="flex items-center gap-2.5">
                <svg
                  className="w-4 h-4 text-slate-500 dark:text-slate-400 shrink-0"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                >
                  <path d="M22.28 9.82a5.98 5.98 0 0 0-.51-4.91 6.05 6.05 0 0 0-6.51-2.9 6.07 6.07 0 0 0-10.28 2.17 5.98 5.98 0 0 0-4 2.9 6.05 6.05 0 0 0 .74 7.1 5.98 5.98 0 0 0 .51 4.91 6.05 6.05 0 0 0 6.52 2.9 5.98 5.98 0 0 0 4.5 2.01 6.06 6.06 0 0 0 5.77-4.2 5.99 5.99 0 0 0 4-2.9 6.06 6.06 0 0 0-.74-7.08zm-9.02 12.61a4.48 4.48 0 0 1-2.88-1.04l.14-.08 4.78-2.76a.8.8 0 0 0 .4-.68v-6.74l2.02 1.17a.07.07 0 0 1 .04.05v5.59a4.5 4.5 0 0 1-4.5 4.49zm-9.66-4.13a4.47 4.47 0 0 1-.53-3.01l.14.08 4.78 2.76a.77.77 0 0 0 .78 0l5.85-3.37v2.33a.08.08 0 0 1-.03.06l-4.84 2.8a4.5 4.5 0 0 1-6.15-1.65z" />
                </svg>
                <span className="text-[13.5px] font-medium">
                  Open in ChatGPT
                </span>
              </span>
              <span className="text-xs text-slate-400 dark:text-slate-500">
                ↗
              </span>
            </button>
          </li>

          <li>
            <button
              type="button"
              onClick={() => handleOpenAI("claude")}
              className="w-full flex items-center justify-between px-2.5 py-2 rounded-lg hover:bg-slate-100/80 dark:hover:bg-slate-800/60 hover:text-slate-900 dark:hover:text-slate-100 transition-colors cursor-pointer text-left"
            >
              <span className="flex items-center gap-2.5">
                <svg
                  className="w-4 h-4 text-slate-500 dark:text-slate-400 shrink-0"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                >
                  <path d="M13.5 2h-3l.8 6.2-5.4-3.2-1.5 2.6 5.4 3.2-6.3.8v3l6.3.8-5.4 3.2 1.5 2.6 5.4-3.2-.8 6.2h3l-.8-6.2 5.4 3.2 1.5-2.6-5.4-3.2 6.3-.8v-3l-6.3-.8 5.4-3.2-1.5-2.6-5.4 3.2z" />
                </svg>
                <span className="text-[13.5px] font-medium">
                  Open in Claude
                </span>
              </span>
              <span className="text-xs text-slate-400 dark:text-slate-500">
                ↗
              </span>
            </button>
          </li>

          <li>
            <button
              type="button"
              onClick={handleViewRaw}
              className="w-full flex items-center justify-between px-2.5 py-2 rounded-lg hover:bg-slate-100/80 dark:hover:bg-slate-800/60 hover:text-slate-900 dark:hover:text-slate-100 transition-colors cursor-pointer text-left"
            >
              <span className="flex items-center gap-2.5">
                <svg
                  className="w-4 h-4 text-slate-500 dark:text-slate-400 shrink-0"
                  viewBox="0 0 16 16"
                  fill="currentColor"
                >
                  <path d="M14 3a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h12zM2 2a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H2z" />
                  <path d="M3.5 11V5h1.5l1.5 2 1.5-2H9.5v6H8V7.8L6.5 9.8 5 7.8V11H3.5zm7.5-2h1V6h1.5v3h1l-1.75 2.5L11 9z" />
                </svg>
                <span className="text-[13.5px] font-medium">
                  View in Markdown
                </span>
              </span>
              <span className="text-xs text-slate-400 dark:text-slate-500">
                ↗
              </span>
            </button>
          </li>
        </ul>
      </div>
    </aside>
  );
}
