"use client";

import React, { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api-client";
import { toast } from "sonner";
import { Heart } from "lucide-react";

function ToastWithHeart({ id, title, description }: { id: string | number; title: string; description?: string }) {
  const [liked, setLiked] = useState(false);

  return React.createElement("div", {
    className: "flex items-center gap-3 w-full max-w-sm rounded-lg border border-border bg-background p-4 shadow-lg",
  },
    React.createElement("div", { className: "flex-1 min-w-0" },
      React.createElement("p", { className: "text-sm font-medium text-foreground" }, title),
      description && React.createElement("p", { className: "text-xs text-muted-foreground mt-0.5" }, description),
    ),
    React.createElement("button", {
      onClick: (e: React.MouseEvent) => {
        e.stopPropagation();
        e.preventDefault();
        setLiked(true);
        setTimeout(() => toast.dismiss(id), 1500);
      },
      className: "flex-shrink-0 p-2 rounded-full hover:bg-muted transition-colors",
    },
      React.createElement(Heart, {
        style: liked
          ? { color: "#ef4444", fill: "#ef4444", transform: "scale(1.3)", transition: "all 0.3s ease" }
          : { color: "#888", fill: "none", transition: "all 0.3s ease" },
        className: "h-5 w-5",
      }),
    ),
  );
}

const EVENT_CONFIG: Record<string, { title: string; description?: string }> = {
  "estimation.created": { title: "Создана новая оценка" },
  "estimation.submitted": { title: "Оценка отправлена", description: "Сводная таблица обновлена" },
  "document.uploaded": { title: "Загружен новый документ" },
  "member.added": { title: "Добавлен новый участник" },
  "project.updated": { title: "Проект обновлён" },
};

export function useWebSocket() {
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<NodeJS.Timeout | undefined>(undefined);

  useEffect(() => {
    function connect() {
      const token = getAccessToken();
      if (!token) return;

      const wsBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080")
        .replace("http://", "ws://")
        .replace("https://", "wss://");

      const ws = new WebSocket(`${wsBase}/api/v1/ws?token=${token}`);
      wsRef.current = ws;

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          const { type, project_id } = data;

          // Show toast with dismiss
          const config = EVENT_CONFIG[type];
          if (config) {
            toast.custom((id) =>
              React.createElement(ToastWithHeart, { id, title: config.title, description: config.description }),
              { duration: 15000 },
            );
          }

          // Invalidate caches
          if (type?.startsWith("estimation.") && project_id) {
            queryClient.invalidateQueries({ queryKey: ["estimations", project_id] });
            queryClient.invalidateQueries({ queryKey: ["aggregated", project_id] });
          }
          if (type === "document.uploaded" && project_id) {
            queryClient.invalidateQueries({ queryKey: ["documents", project_id] });
          }
          if (type === "member.added" && project_id) {
            queryClient.invalidateQueries({ queryKey: ["members", project_id] });
          }
          if (type === "project.updated") {
            queryClient.invalidateQueries({ queryKey: ["projects"] });
          }
        } catch {
          // ignore
        }
      };

      ws.onclose = () => {
        reconnectTimer.current = setTimeout(connect, 3000);
      };

      ws.onerror = () => {
        ws.close();
      };
    }

    connect();

    return () => {
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
    };
  }, [queryClient]);
}
