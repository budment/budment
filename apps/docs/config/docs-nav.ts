export interface DocItem {
  title: string;
  href: string;
  keywords?: string;
}

export interface DocSection {
  title: string;
  items: DocItem[];
}

export const DOCS_NAV: DocSection[] = [
  {
    title: "Getting Started",
    items: [
      {
        title: "Introduction",
        href: "/docs/getting-started/01-introduction",
        keywords: "overview intro concept engine distributed typescript go",
      },
      {
        title: "Installation",
        href: "/docs/getting-started/02-installation",
        keywords: "install curl brew binary setup npm download",
      },
      {
        title: "Quickstart Guide",
        href: "/docs/getting-started/03-quickstart",
        keywords: "quickstart run script first scenario test execute",
      },
    ],
  },
  {
    title: "Core Guide",
    items: [
      {
        title: "01. Lifecycle & Flow",
        href: "/docs/guide/01-lifecycle",
        keywords: "lifecycle dag phase setup execution teardown state machine",
      },
      {
        title: "02. HTTP Pipelines",
        href: "/docs/guide/02-http-requests",
        keywords: "http request pipeline mutate assert client status code",
      },
      {
        title: "03. Control Flow DAG",
        href: "/docs/guide/03-control-flow",
        keywords: "control flow dag branch match loop poll condition",
      },
      {
        title: "04. Test Data & Memory",
        href: "/docs/guide/04-test-data",
        keywords: "memory scope global local worker context pool test data",
      },
      {
        title: "05. Metrics & Thresholds",
        href: "/docs/guide/05-metrics-thresholds",
        keywords: "metrics thresholds latency rps duration benchmark p99",
      },
      {
        title: "06. FAQ & Best Practices",
        href: "/docs/guide/06-faq-best-practices",
        keywords: "faq best practices tips troubleshooting common errors",
      },
    ],
  },
  {
    title: "Reference & Spec",
    items: [
      {
        title: "CLI Reference",
        href: "/docs/reference/cli",
        keywords: "cli budment plan run vus duration json dry-run flags terminal",
      },
      {
        title: "System Architecture",
        href: "/docs/ARCHITECTURE",
        keywords: "architecture compilation esbuild goja protobuf ir ast runtime fsm",
      },
    ],
  },
];

// Tự động làm phẳng danh sách để dùng cho thanh tìm kiếm và Footer
export const SEARCH_INDEX = DOCS_NAV.flatMap((section) =>
  section.items.map((item) => ({
    ...item,
    category: section.title,
    keywords: item.keywords ?? "",
  }))
);