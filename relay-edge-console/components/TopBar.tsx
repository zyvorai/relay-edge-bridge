"use client";

import { useEffect, useState } from "react";

export default function TopBar() {
  const [now, setNow] = useState<Date | null>(null);

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);

  return (
    <header className="h-14 border-b border-border glass flex items-center justify-between px-6">
      <div className="text-sm text-text-dim font-mono">
        {now ? now.toISOString().replace("T", " ").slice(0, 19) + " UTC" : " "}
      </div>
      <div className="text-xs text-text-dim font-mono">Phase 1 &middot; single-token auth</div>
    </header>
  );
}
