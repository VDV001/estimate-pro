"use client";

import { useEffect, useState } from "react";
import { cn } from "@/lib/utils";
import { getAccessToken } from "@/lib/api-client";

interface UserAvatarProps {
  name?: string | null;
  avatarUrl?: string | null;
  size?: "sm" | "md" | "lg";
  className?: string;
}

const sizeClasses = {
  sm: "h-8 w-8 text-xs",
  md: "h-10 w-10 text-sm",
  lg: "h-24 w-24 text-3xl",
};

function getInitials(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

function getColor(name: string): string {
  const colors = [
    "#8b5cf6", "#3b82f6", "#10b981", "#f59e0b",
    "#ef4444", "#06b6d4", "#ec4899", "#6366f1",
    "#14b8a6", "#f97316",
  ];
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return colors[Math.abs(hash) % colors.length];
}

// Global cache — survives React remounts (locale switch)
const avatarCache = new Map<string, string>();

export function UserAvatar({ name, avatarUrl, size = "md", className }: UserAvatarProps) {
  const displayName = name || "?";
  const initials = getInitials(displayName);
  const bgColor = getColor(displayName);

  // Check cache first — instant render on remount
  const cached = avatarUrl ? avatarCache.get(avatarUrl) : null;
  const [blobUrl, setBlobUrl] = useState<string | null>(cached ?? null);

  useEffect(() => {
    if (!avatarUrl) {
      setBlobUrl(null);
      return;
    }

    // Already cached — use it
    if (avatarCache.has(avatarUrl)) {
      setBlobUrl(avatarCache.get(avatarUrl)!);
      return;
    }

    const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
    const token = getAccessToken();
    const headers: Record<string, string> = {};
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    let cancelled = false;
    fetch(`${apiBase}${avatarUrl}`, { headers })
      .then((res) => {
        if (!res.ok) throw new Error("Failed to load avatar");
        return res.blob();
      })
      .then((blob) => {
        if (!cancelled) {
          const url = URL.createObjectURL(blob);
          avatarCache.set(avatarUrl, url);
          setBlobUrl(url);
        }
      })
      .catch(() => {
        if (!cancelled) setBlobUrl(null);
      });

    return () => { cancelled = true; };
  }, [avatarUrl]);

  return (
    <div
      className={cn(
        "rounded-full overflow-hidden flex items-center justify-center font-semibold text-white flex-shrink-0 aspect-square",
        sizeClasses[size],
        className,
      )}
      style={{ backgroundColor: blobUrl ? undefined : bgColor }}
    >
      {blobUrl ? (
        <img
          src={blobUrl}
          alt={displayName}
          className="w-full h-full object-cover"
        />
      ) : (
        initials
      )}
    </div>
  );
}
