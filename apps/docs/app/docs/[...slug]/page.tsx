import fs from "fs";
import path from "path";
import matter from "gray-matter";
import { marked, type Tokens } from "marked";
import { notFound } from "next/navigation";
import type { Metadata } from "next";
import DocContent from "../../../components/DocContent";
import RightSidebar, { HeadingItem } from "../../../components/TableOfContents";
import hljs from "highlight.js";
import DocFooter from "@/components/DocFooter";

interface PageProps {
  params: Promise<{ slug: string[] }>;
}

function resolveDocsDir(): string {
  const fromApp = path.resolve(process.cwd(), "../../docs");
  if (fs.existsSync(fromApp)) return fromApp;
  return path.resolve(process.cwd(), "docs");
}

function getFilePath(slug: string[]): string | null {
  const docsDir = resolveDocsDir();

  if (
    slug.length === 2 &&
    slug[0] === "reference" &&
    slug[1] === "architecture"
  ) {
    const archAtRoot = path.resolve(docsDir, "../ARCHITECTURE.md");
    if (fs.existsSync(archAtRoot)) return archAtRoot;
  }

  const candidatePath = path.join(docsDir, `${slug.join("/")}.md`);
  if (fs.existsSync(candidatePath)) return candidatePath;

  return null;
}

export async function generateStaticParams() {
  const docsDir = resolveDocsDir();
  if (!fs.existsSync(docsDir)) return [];

  const getFiles = (dir: string): string[] => {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    return entries.flatMap((entry) => {
      const res = path.resolve(dir, entry.name);
      return entry.isDirectory() ? getFiles(res) : res;
    });
  };

  const mdFiles = getFiles(docsDir).filter((file) => file.endsWith(".md"));

  const paramsList = mdFiles.map((file) => {
    const relative = path.relative(docsDir, file);
    const slug = relative.replace(/\.md$/, "").split(path.sep);
    return { slug };
  });

  paramsList.push({ slug: ["reference", "architecture"] });

  return paramsList;
}

export async function generateMetadata({
  params,
}: PageProps): Promise<Metadata> {
  const { slug } = await params;
  const targetFilePath = getFilePath(slug);
  if (!targetFilePath) return { title: "Documentation | Budment" };

  const rawContent = fs.readFileSync(targetFilePath, "utf8");
  const { data: frontmatter } = matter(rawContent);

  return {
    title: frontmatter.title
      ? `${frontmatter.title} — Budment Docs`
      : "Documentation | Budment",
    description:
      frontmatter.description || "Budment declarative engine documentation.",
  };
}

function escapeHtml(text: string) {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/<[^>]*>/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

function highlightAST(raw: string): string {
  let text = escapeHtml(raw);

  text = text.replace(
    /^([◆▶]\s*.*)$/gm,
    '<span class="text-pink-500 font-bold">$1</span>',
  );
  text = text.replace(
    /\[HTTP\]/g,
    '<span class="text-sky-400 font-semibold">[HTTP]</span>',
  );
  text = text.replace(
    /\bGET\b/g,
    '<span class="text-emerald-400 font-bold">GET</span>',
  );
  text = text.replace(
    /\b(POST|PUT|PATCH)\b/g,
    '<span class="text-amber-400 font-bold">$1</span>',
  );
  text = text.replace(
    /\bDELETE\b/g,
    '<span class="text-rose-500 font-bold">DELETE</span>',
  );
  text = text.replace(
    /\[BRANCH\]/g,
    '<span class="text-yellow-400 font-bold">[BRANCH]</span>',
  );
  text = text.replace(
    /\[MATCH\]/g,
    '<span class="text-fuchsia-400 font-bold">[MATCH]</span>',
  );
  text = text.replace(
    /\[(LOOP|POLL)\]/g,
    '<span class="text-purple-400 font-bold">[$1]</span>',
  );
  text = text.replace(
    /\[TRUE\]/g,
    '<span class="text-cyan-400 font-semibold">[TRUE]</span>',
  );
  text = text.replace(
    /\[FALSE\]/g,
    '<span class="text-rose-400 font-semibold">[FALSE]</span>',
  );
  text = text.replace(
    /(Case:\s*)([a-zA-Z0-9_-]+)/g,
    '<span class="text-slate-400 font-medium">$1</span><span class="text-amber-300 font-semibold">$2</span>',
  );
  text = text.replace(
    /\[(BEFORE|AFTER)\]/gi,
    '<span class="text-fuchsia-400 font-semibold">[$1]</span>',
  );
  text = text.replace(/→\s*([a-zA-Z0-9_,\s]+)/g, (_, actions) => {
    const highlighted = actions.replace(
      /\b(res_assert|script|req_mutate|barrier|log)\b/g,
      '<span class="text-slate-300 font-medium">$1</span>',
    );
    return `<span class="text-slate-500">→</span> ${highlighted}`;
  });
  text = text.replace(
    /(\[id:\s*[^\]]+\])/g,
    '<span class="text-slate-500">$1</span>',
  );
  text = text.replace(
    /(├──|└──|│)/g,
    '<span class="text-slate-600 font-normal">$1</span>',
  );
  text = text.replace(/(-{10,})/g, '<span class="text-slate-700">$1</span>');

  return text;
}

const LANG_THEMES: Record<string, { text: string; bar: string }> = {
  ast: { text: "text-pink-400", bar: "bg-pink-500" },
  tree: { text: "text-pink-400", bar: "bg-pink-500" },
  lifetree: { text: "text-pink-400", bar: "bg-pink-500" },
  bash: { text: "text-amber-400", bar: "bg-amber-400" },
  sh: { text: "text-amber-400", bar: "bg-amber-400" },
  shell: { text: "text-amber-400", bar: "bg-amber-400" },
  zsh: { text: "text-amber-400", bar: "bg-amber-400" },
  typescript: { text: "text-blue-400", bar: "bg-blue-500" },
  ts: { text: "text-blue-400", bar: "bg-blue-500" },
  javascript: { text: "text-yellow-400", bar: "bg-yellow-400" },
  js: { text: "text-yellow-400", bar: "bg-yellow-400" },
  go: { text: "text-cyan-400", bar: "bg-cyan-500" },
  golang: { text: "text-cyan-400", bar: "bg-cyan-500" },
  json: { text: "text-orange-400", bar: "bg-orange-400" },
  yaml: { text: "text-rose-400", bar: "bg-rose-500" },
  text: { text: "text-slate-400", bar: "bg-slate-500" },
};

export default async function DocPage({ params }: PageProps) {
  const { slug } = await params;
  const targetFilePath = getFilePath(slug);

  if (!targetFilePath) {
    notFound();
  }

  const rawContent = fs.readFileSync(targetFilePath, "utf8");
  const { data: frontmatter, content } = matter(rawContent);

  let cleanContent = content.trim();
  if (frontmatter.title) {
    cleanContent = cleanContent.replace(/^#\s+[^\r\n]+(\r?\n)*/, "");
  }

  const words = cleanContent.split(/\s+/).length;
  const readTime = Math.max(1, Math.ceil(words / 200));

  const headings: HeadingItem[] = [];
  const headingCounts = new Map<string, number>();

  const renderer = new marked.Renderer();

  renderer.heading = function ({ text, depth }: Tokens.Heading): string {
    const rawText = text.replace(/<[^>]*>/g, "");
    const baseId = slugify(rawText);
    let finalId = baseId;
    const count = headingCounts.get(baseId) || 0;
    if (count > 0) {
      finalId = `${baseId}-${count}`;
    }
    headingCounts.set(baseId, count + 1);

    if (depth === 2 || depth === 3) {
      headings.push({ id: finalId, text: rawText, level: depth });
      return `<h${depth} id="${finalId}" class="scroll-mt-20">${text}</h${depth}>`;
    }

    return `<h${depth}>${text}</h${depth}>`;
  };

  renderer.code = function ({ text, lang }: Tokens.Code): string {
    const language = (lang || "text").toLowerCase();

    if (language === "mermaid") {
      return `<div class="mermaid my-8 flex justify-center not-prose overflow-x-auto p-4 bg-white/80 dark:bg-slate-900/60 border border-slate-200/80 dark:border-slate-800 rounded-2xl shadow-2xs" translate="no">${text}</div>`;
    }

    const theme = LANG_THEMES[language] || {
      text: "text-blue-400",
      bar: "bg-blue-500",
    };
    const encoded = encodeURIComponent(text);
    let highlightedCode = escapeHtml(text);

    if (["ast", "tree", "lifetree"].includes(language)) {
      highlightedCode = highlightAST(text);
    } else if (lang && hljs.getLanguage(lang)) {
      try {
        highlightedCode = hljs.highlight(text, {
          language: lang,
          ignoreIllegals: true,
        }).value;
      } catch {}
    }

    if (["bash", "sh", "shell", "zsh"].includes(language)) {
      highlightedCode = highlightedCode.replace(
        /(^|\s)(budment)(\s|$)/g,
        '$1<span class="text-blue-400 font-semibold">$2</span>$3',
      );
    }

    return `
      <div class="terminal-box my-6 rounded-lg border border-slate-800/80 bg-[#16181d] overflow-hidden text-left" translate="no">
        <div class="flex items-center justify-between px-4 bg-[#1e222b] border-b border-slate-800">
          <div class="flex items-center">
            <div class="relative py-2 px-1">
              <span class="font-mono text-xs font-semibold uppercase tracking-wider ${theme.text}">
                ${language}
              </span>
              <span class="absolute bottom-0 left-0 right-0 h-0.5 ${theme.bar}"></span>
            </div>
          </div>
          <button type="button" class="copy-btn my-1.5 flex items-center gap-1.5 px-2.5 py-1 bg-[#282d37] hover:bg-[#323846] text-slate-200 border border-slate-700/60 transition-all cursor-pointer font-sans shadow-xs" data-code="${encoded}">
            <svg class="w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            <span class="copy-text text-[13px] font-medium">Copy</span>
          </button>
        </div>
        <pre class="p-4 overflow-x-auto text-[13.5px] font-mono leading-relaxed text-slate-200"><code class="language-${language}">${highlightedCode}</code></pre>
      </div>
    `;
  };

  const htmlContent = await marked.parse(cleanContent, { renderer });
  const breadcrumb = slug.map((s) => s.replace(/^\d+-/, "").replace(/-/g, " "));

  return (
    <div className="w-full flex justify-between gap-10">
      <article className="flex-1 min-w-0 max-w-3xl">
        <div className="flex items-center justify-between text-xs text-slate-400 dark:text-slate-500 mb-6 font-medium">
          <div className="flex items-center gap-1.5 capitalize">
            <span>Docs</span>
            <span>/</span>
            <span>{breadcrumb[0]}</span>
            {breadcrumb[1] && (
              <>
                <span>/</span>
                <span className="text-slate-600 dark:text-slate-300">
                  {breadcrumb[1]}
                </span>
              </>
            )}
          </div>
          <span className="bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400 px-2 py-0.5 rounded-full text-[11px]">
            {readTime} min read
          </span>
        </div>

        {frontmatter.title && (
          <div className="border-b border-slate-200/80 dark:border-slate-800 pb-6 mb-8">
            <h1 className="text-4xl sm:text-[45px] font-light tracking-[-0.035em] text-slate-950 dark:text-white mb-3 leading-[1.12]">
              {frontmatter.title}
            </h1>
            {frontmatter.description && (
              <p className="text-slate-500 dark:text-slate-400 text-[15px] leading-relaxed font-normal">
                {frontmatter.description}
              </p>
            )}
          </div>
        )}

        <DocContent html={htmlContent} />
        <DocFooter />
      </article>

      <aside className="w-64 shrink-0 hidden xl:block sticky top-14 h-[calc(100vh-3.5rem)] overflow-y-auto pl-2">
        <RightSidebar headings={headings} rawMarkdown={rawContent} />
      </aside>
    </div>
  );
}
