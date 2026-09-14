import type { Metadata } from "next";
import {
  Plus_Jakarta_Sans,
  JetBrains_Mono,
  Instrument_Sans,
} from "next/font/google";
import { ThemeProvider } from "next-themes";
import "./globals.css";

const jakarta = Plus_Jakarta_Sans({
  variable: "--font-sans",
  subsets: ["latin"],
  display: "swap",
  weight: ["400", "500", "600"],
});

const instrument = Instrument_Sans({
  variable: "--font-display",
  subsets: ["latin"],
  display: "swap",
  weight: ["600", "700"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-mono",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  metadataBase: new URL("https://budment.com"),
  title: {
    default: "Budment — Scenarios as AST, Logic as Code",
    template: "%s | Budment",
  },
  description:
    "Open-source API execution engine. Define workflows in TypeScript and compile them into deterministic execution plans.",
  applicationName: "Budment",
  authors: [{ name: "Budment Team", url: "https://budment.com" }],
  generator: "Next.js",
  keywords: [
    "load testing",
    "API testing",
    "AST",
    "TypeScript",
    "Go",
    "developer tools",
    "performance testing",
    "declarative testing",
  ],
  referrer: "origin-when-cross-origin",
  creator: "Budment",
  publisher: "Budment",
  formatDetection: {
    email: false,
    address: false,
    telephone: false,
  },
  openGraph: {
    type: "website",
    locale: "en_US",
    url: "https://budment.com",
    siteName: "Budment",
    title: "Budment — Scenarios as AST, Logic as Code",
    description:
      "Open-source API execution engine. Define workflows in TypeScript and compile them into deterministic execution plans.",
    images: [
      {
        url: "/og-image.png",
        width: 1200,
        height: 630,
        alt: "Budment Engine Overview",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Budment — Scenarios as AST, Logic as Code",
    description:
      "Open-source API execution engine. Define workflows in TypeScript and compile them into deterministic execution plans.",
    images: ["/og-image.png"],
    creator: "@budment",
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-video-preview": -1,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },
  icons: {
    icon: "/favicon.ico",
    shortcut: "/favicon.ico",
    apple: "/apple-touch-icon.png",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${jakarta.variable} ${instrument.variable} ${jetbrainsMono.variable}`}
    >
      <body className="font-sans antialiased text-slate-800 dark:text-slate-200 bg-[#fbfbfa] dark:bg-[#0c0d12] transition-colors duration-200">
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          {children}
        </ThemeProvider>
      </body>
    </html>
  );
}
