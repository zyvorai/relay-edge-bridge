import type { Metadata } from "next";
import { JetBrains_Mono, IBM_Plex_Sans, Outfit } from "next/font/google";
import "./globals.css";
import Sidebar from "@/components/Sidebar";
import TopBar from "@/components/TopBar";

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains-mono",
  subsets: ["latin"],
  weight: ["400", "500", "700"],
});

const ibmPlexSans = IBM_Plex_Sans({
  variable: "--font-ibm-plex-sans",
  subsets: ["latin"],
  weight: ["400", "500", "600"],
});

const outfit = Outfit({
  variable: "--font-outfit",
  subsets: ["latin"],
  weight: ["500", "600", "700"],
});

export const metadata: Metadata = {
  title: "Relay Edge Console",
  description: "Fleet, device health, live events, and actions for relay-edge gateways.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      className={`${jetbrainsMono.variable} ${ibmPlexSans.variable} ${outfit.variable} h-full antialiased`}
    >
      <body className="min-h-full">
        <div className="flex min-h-screen">
          <Sidebar />
          <div className="flex-1 flex flex-col min-w-0">
            <TopBar />
            <main className="flex-1 p-6 max-w-[1400px] w-full mx-auto">{children}</main>
          </div>
        </div>
      </body>
    </html>
  );
}
