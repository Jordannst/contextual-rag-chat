import type { Metadata } from "next";
import { Geist_Mono } from "next/font/google";
import { JetBrains_Mono } from "next/font/google";
import Script from "next/script";
import "@fontsource-variable/inter";
import "./globals.css";
import AmbientBackground from "@/components/ui/AmbientBackground";

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains-mono",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: "AI RAG Chatbot",
  description: "Modern AI-powered document chat interface",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark" suppressHydrationWarning>
      <head>
        {/* Always set dark mode */}
        <Script
          id="theme-script"
          strategy="beforeInteractive"
          dangerouslySetInnerHTML={{
            __html: `
              (function() {
                document.documentElement.classList.add('dark');
              })();
            `,
          }}
        />
      </head>
      <body
        className={`${geistMono.variable} ${jetbrainsMono.variable} font-sans antialiased transition-colors duration-300`}
      >
        <AmbientBackground />
        {children}
      </body>
    </html>
  );
}
