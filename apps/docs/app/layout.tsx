import { Plus_Jakarta_Sans, JetBrains_Mono, Instrument_Sans } from "next/font/google";
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

export default function RootLayout({ children }: { children: React.ReactNode }) {
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
