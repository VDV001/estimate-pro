"use client";

import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api-client";
import { toast } from "sonner";

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

          // Show toast
          const config = EVENT_CONFIG[type];
          if (config) {
            toast(config.title, {
              description: config.description,
            });
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
