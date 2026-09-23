"use client";

import { useTheme } from "next-themes";
import { useSyncExternalStore } from "react";
import { Sun, Moon, Laptop } from "lucide-react";

export default function ThemeToggle() {
  const mounted = useSyncExternalStore(
    () => () => {},
    () => true,
    () => false
  );
  const { theme, setTheme } = useTheme();

  if (!mounted) return <div className="w-9 h-9" aria-hidden="true" />;

  const cycleTheme = () => {
    const next = theme === "dark" ? "system" : theme === "system" ? "light" : "dark";
    setTheme(next);
    window.dispatchEvent(new Event("theme-change"));
  };

  return (
    <button
      type="button"
      onClick={cycleTheme}
      title={`Current: ${theme}`}
      aria-label="Toggle theme"
      className="p-2 rounded-md text-slate-600 dark:text-slate-400 hover:text-slate-950 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors cursor-pointer"
    >
      {theme === "light" && <Sun className="w-4 h-4" />}
      {theme === "dark" && <Moon className="w-4 h-4" />}
      {theme === "system" && <Laptop className="w-4 h-4" />}
    </button>
  );
}