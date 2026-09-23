import fs from "fs";
import path from "path";
import matter from "gray-matter";
import { marked, type Tokens } from "marked";
import { notFound, redirect } from "next/navigation";
import type { Metadata } from "next";
import Link from "next/link";
import DocContent from "../../../components/DocContent";
import RightSidebar, { HeadingItem } from "../../../components/TableOfContents";
import hljs from "highlight.js";
import DocFooter from "@/components/DocFooter";
import { DOCS_NAV } from "@/config/docs-nav";

interface PageProps {
  params: Promise<{ slug?: string[] }>;
}

/**
 * Resolves the root documentation directory across workspace structures.
 */
function resolveDocsDir(): string {
  const fromApp = path.resolve(process.cwd(), "../../docs");
  if (fs.existsSync(fromApp)) return fromApp;
  return path.resolve(process.cwd(), "docs");
}

/**
 * Locates the markdown file corresponding to the route slug.
 */
function getFilePath(slug: string[]): string | null {
  const docsDir = resolveDocsDir();

  // Route alias: reference architecture mapping
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

/**
 * Generates dynamic static routing params for SSG builds.
 */
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

/**
 * Extracts SEO frontmatter metadata per document page.
 */
export async function generateMetadata({
  params,
}: PageProps): Promise<Metadata> {
  const { slug } = await params;
  if (!slug || slug.length === 0) {
    return {
      title: "Documentation",
      alternates: { canonical: "/docs" },
    };
  }

  const targetFilePath = getFilePath(slug);
  const currentPath = `/docs/${slug.join("/")}`;

  if (!targetFilePath) {
    return {
      title: "Documentation",
      alternates: { canonical: currentPath },
    };
  }

  const rawContent = fs.readFileSync(targetFilePath, "utf8");
  const { data: frontmatter } = matter(rawContent);

  const title = frontmatter.title || "Documentation";
  const description =
    frontmatter.description || "Budment declarative engine documentation.";

  return {
    title,
    description,
    alternates: {
      canonical: currentPath,
    },
    openGraph: {
      title: `${title} | Budment Docs`,
      description,
      url: currentPath,
      type: "article",
    },
    twitter: {
      card: "summary_large_image",
      title: `${title} | Budment Docs`,
      description,
    },
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

/**
 * Applies syntax tokenization and color highlights to AST trees.
 */
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
    /\[(LOOP\vert{}POLL)\]/g,
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
    /\[(BEFORE\vert{}AFTER)\]/gi,
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

const EXPAND_ICON_SVG = `<svg class="w-4 h-4 text-slate-300 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" /></svg>`;

export default async function DocPage({ params }: PageProps) {
  const { slug } = await params;

  if (!slug || slug.length === 0) {
    redirect("/docs/getting-started/01-introduction");
  }

  const targetFilePath = getFilePath(slug);

  if (!targetFilePath) {
    const currentPrefix = `/docs/${slug.join("/")}`;
    const matchedItem = DOCS_NAV.flatMap((s) => s.items).find((item) =>
      item.href.startsWith(currentPrefix),
    );
    if (matchedItem) {
      redirect(matchedItem.href);
    }
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
    if (count > 0) finalId = `${baseId}-${count}`;
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
      const encodedCode = encodeURIComponent(text);
      return `
        <div class="mermaid-block group relative my-8 rounded-2xl bg-white/80 dark:bg-slate-900/60 border border-slate-200/80 dark:border-slate-800 shadow-2xs overflow-hidden" translate="no" data-code="${encodedCode}">
          <div class="mermaid-container relative p-6 flex justify-center items-center min-h-105 overflow-x-auto">
            <div class="flex flex-col items-center justify-center gap-3 text-slate-400 dark:text-slate-500">
              <span class="text-xs font-mono tracking-wide text-slate-400 dark:text-slate-500">
                Rendering architecture diagram...
              </span>
            </div>
          </div>
        </div>
      `;
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

    const lines = text.split("\n").length;
    const isExpandable = lines > 15;

    return `
      <div class="terminal-box ${isExpandable ? "is-expandable relative" : ""} my-6 rounded-lg border border-slate-800/80 bg-[#16181d] overflow-hidden text-left" translate="no">
        <div class="flex items-center justify-between px-4 bg-[#1e222b] border-b border-slate-800">
          <div class="flex items-center">
            <div class="relative py-2 px-1">
              <span class="font-mono text-xs font-semibold uppercase tracking-wider ${theme.text}">${language}</span>
              <span class="absolute bottom-0 left-0 right-0 h-0.5 ${theme.bar}"></span>
            </div>
          </div>
          <button type="button" class="copy-btn my-1.5 flex items-center gap-1.5 px-2.5 py-1 bg-[#282d37] hover:bg-[#323846] text-slate-200 border border-slate-700/60 transition-all cursor-pointer font-sans shadow-xs" data-code="${encoded}">
            <svg class="w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            <span class="copy-text text-[13px] font-medium">Copy</span>
          </button>
        </div>
        <pre class="p-4 overflow-x-auto text-[13.5px] font-mono leading-relaxed text-slate-200 ${isExpandable ? "max-h-80 overflow-y-hidden" : ""}"><code class="language-${language}">${highlightedCode}</code></pre>
        ${
          isExpandable
            ? `
          <div class="expand-fade absolute bottom-0 left-0 right-0 h-24 bg-linear-to-t from-[#16181d] via-[#16181d]/85 to-transparent pointer-events-none transition-opacity duration-300 z-1"></div>
          <button type="button" class="expand-toggle-btn absolute bottom-3 right-3 z-10 flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-[#212631]/95 hover:bg-[#2c3342] text-slate-200 border border-slate-700/80 hover:border-slate-500 shadow-md hover:shadow-lg backdrop-blur-sm transition-all duration-200 cursor-pointer text-xs font-medium">
            ${EXPAND_ICON_SVG}
            <span class="btn-text">Expand</span>
          </button>
        `
            : ""
        }
      </div>
    `;
  };

  const htmlContent = await marked.parse(cleanContent, { renderer });

  // Resolve breadcrumb navigation URLs
  const currentPath = `/docs/${slug.join("/")}`;
  const docsHomeHref = DOCS_NAV[0]?.items[0]?.href || "/docs";

  // Match the first valid link of the parent section to avoid 404 on category landing
  const parentSection = DOCS_NAV.find((section) =>
    section.items.some((item) => item.href.includes(`/${slug[0]}`)),
  );
  const sectionHref = parentSection?.items[0]?.href || `/docs/${slug[0]}`;

  // Clean raw segment titles (e.g., "01-getting-started" -> "getting started")
  const breadcrumb = slug.map((s) => s.replace(/^\d+-/, "").replace(/-/g, " "));

  return (
    <div className="w-full flex justify-between gap-10">
      <article className="flex-1 min-w-0 max-w-3xl">
        {/* Interactive Breadcrumbs Navigation */}
        <div className="flex items-center justify-between text-xs text-slate-400 dark:text-slate-500 mb-6 font-medium">
          <nav
            aria-label="Breadcrumb"
            className="flex items-center gap-1.5 capitalize flex-wrap"
          >
            <Link
              href={docsHomeHref}
              className="hover:text-slate-900 dark:hover:text-white transition-colors"
            >
              Docs
            </Link>

            <span className="text-slate-300 dark:text-slate-600">/</span>

            {slug.length > 1 ? (
              <>
                <Link
                  href={sectionHref}
                  className="hover:text-slate-900 dark:hover:text-white transition-colors"
                >
                  {breadcrumb[0]}
                </Link>

                <span className="text-slate-300 dark:text-slate-600">/</span>

                <Link
                  href={currentPath}
                  aria-current="page"
                  className="text-slate-700 dark:text-slate-300 font-semibold hover:text-slate-950 dark:hover:text-white transition-colors"
                >
                  {breadcrumb.slice(1).join(" / ")}
                </Link>
              </>
            ) : (
              <Link
                href={currentPath}
                aria-current="page"
                className="text-slate-700 dark:text-slate-300 font-semibold hover:text-slate-950 dark:hover:text-white transition-colors"
              >
                {breadcrumb[0]}
              </Link>
            )}
          </nav>

          <span className="bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400 px-2 py-0.5 rounded-full text-[11px] shrink-0">
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
