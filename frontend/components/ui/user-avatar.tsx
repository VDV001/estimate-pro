"use client";

import { cn } from "@/lib/utils";

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

// Deterministic color from name
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

export function UserAvatar({ name, avatarUrl, size = "md", className }: UserAvatarProps) {
  const displayName = name || "?";
  const initials = getInitials(displayName);
  const bgColor = getColor(displayName);

  return (
    <div
      className={cn(
        "rounded-full overflow-hidden flex items-center justify-center font-semibold text-white flex-shrink-0 aspect-square",
        sizeClasses[size],
        className,
      )}
      style={{ backgroundColor: avatarUrl ? undefined : bgColor }}
    >
      {avatarUrl ? (
        <img
          src={avatarUrl}
          alt={displayName}
          className="w-full h-full object-cover"
        />
      ) : (
        initials
      )}
    </div>
  );
}
