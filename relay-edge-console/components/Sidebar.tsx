"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { getGateways } from "@/lib/gateways";

const NAV = [
  { href: "/", label: "Fleet" },
  { href: "/actions", label: "Actions" },
];

export default function Sidebar() {
  const pathname = usePathname();
  const gateways = getGateways();

  return (
    <aside className="w-60 shrink-0 border-r border-border glass flex flex-col">
      <div className="px-5 py-5">
        <div className="font-display font-semibold tracking-wide text-lg text-gradient">
          RELAY EDGE
        </div>
        <div className="text-xs text-text-dim font-mono mt-0.5">operator console</div>
      </div>
      <nav className="px-3 flex flex-col gap-1">
        {NAV.map((item) => {
          const active = pathname === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                active
                  ? "bg-cyan/10 text-cyan border border-cyan/30"
                  : "text-text-dim hover:text-text hover:bg-white/5"
              }`}
            >
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="mt-6 px-5">
        <div className="text-xs uppercase tracking-wider text-text-dim mb-2">Sites</div>
        <div className="flex flex-col gap-1">
          {gateways.map((gw) => (
            <Link
              key={gw.name}
              href={`/sites/${encodeURIComponent(gw.name)}`}
              className={`px-3 py-1.5 rounded-md text-sm font-mono transition-colors ${
                pathname === `/sites/${gw.name}`
                  ? "bg-cyan/10 text-cyan border border-cyan/30"
                  : "text-text-dim hover:text-text hover:bg-white/5"
              }`}
            >
              {gw.name}
            </Link>
          ))}
        </div>
      </div>
    </aside>
  );
}
