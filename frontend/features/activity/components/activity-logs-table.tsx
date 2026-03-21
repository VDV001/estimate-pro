// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { AnimatePresence, motion } from "framer-motion";
import { ChevronDown, Filter, Search, Check } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type ActivityLevel = "info" | "warning" | "success";

export interface ActivityLog {
  id: string;
  timestamp: string;
  level: ActivityLevel;
  service: string; // project name
  eventType?: string; // event type label (translated)
  message: string;
  status: string;
  tags: string[];
}

type Filters = {
  level: string[];
  service: string[];
  eventType: string[];
  status: string[];
};

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const levelStyles: Record<ActivityLevel, string> = {
  info: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  warning: "bg-yellow-500/10 text-yellow-600 dark:text-yellow-400",
  success: "bg-green-500/10 text-green-600 dark:text-green-400",
};

const levelLabels: Record<ActivityLevel, string> = {
  info: "Info",
  success: "OK",
  warning: "Draft",
};

const statusStyles: Record<string, string> = {
  uploaded: "text-green-600 dark:text-green-400",
  submitted: "text-green-600 dark:text-green-400",
  draft: "text-yellow-600 dark:text-yellow-400",
  created: "text-blue-600 dark:text-blue-400",
};

// ---------------------------------------------------------------------------
// LogRow
// ---------------------------------------------------------------------------

function LogRow({
  log,
  expanded,
  onToggle,
}: {
  log: ActivityLog;
  expanded: boolean;
  onToggle: () => void;
}) {
  const t = useTranslations("dashboard");
  const formattedTime = new Date(log.timestamp).toLocaleTimeString("ru-RU", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });

  const formattedDate = new Date(log.timestamp).toLocaleDateString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
  });

  return (
    <>
      <motion.div
        role="button"
        tabIndex={0}
        onClick={onToggle}
        onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") onToggle(); }}
        className="w-full p-4 text-left transition-colors hover:bg-muted/50 cursor-pointer"
      >
        <div className="flex items-center gap-4">
          <motion.div
            animate={{ rotate: expanded ? 180 : 0 }}
            transition={{ duration: 0.2 }}
            className="flex-shrink-0"
          >
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          </motion.div>

          <Badge
            variant="secondary"
            className={`flex-shrink-0 capitalize ${levelStyles[log.level]}`}
          >
            {levelLabels[log.level]}
          </Badge>

          <time className="w-24 flex-shrink-0 font-mono text-xs text-muted-foreground" suppressHydrationWarning>
            {formattedDate} {formattedTime}
          </time>

          <span className="flex-shrink-0 min-w-max text-sm font-medium text-foreground">
            {log.service}
          </span>

          <p className="flex-1 truncate text-sm text-muted-foreground">
            {log.message}
          </p>

          <span
            className={`flex-shrink-0 text-sm font-medium ${
              statusStyles[log.status] ?? "text-muted-foreground"
            }`}
          >
            {log.status}
          </span>
        </div>
      </motion.div>

      <AnimatePresence initial={false}>
        {expanded && (
          <motion.div
            key="details"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="overflow-hidden border-t border-border bg-muted/50"
          >
            <div className="space-y-4 p-4">
              <div>
                <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  {t("logsDetails")}
                </p>
                <p className="rounded bg-background p-3 text-sm text-foreground">
                  {log.message}
                </p>
              </div>

              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                    {t("logsProject")}
                  </p>
                  <p className="text-foreground">{log.service}</p>
                </div>
                <div>
                  <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                    {t("logsTimestamp")}
                  </p>
                  <p className="font-mono text-xs text-foreground" suppressHydrationWarning>
                    {new Date(log.timestamp).toLocaleString()}
                  </p>
                </div>
              </div>

              {log.tags.length > 0 && (
                <div>
                  <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                    {t("logsTags")}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {log.tags.map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </>
  );
}

// ---------------------------------------------------------------------------
// FilterPanel
// ---------------------------------------------------------------------------

function FilterPanel({
  filters,
  onChange,
  logs,
}: {
  filters: Filters;
  onChange: (f: Filters) => void;
  logs: ActivityLog[];
}) {
  const t = useTranslations("dashboard");
  const services = useMemo(() => [...new Set(logs.map((l) => l.service))], [logs]);
  const levels = useMemo(() => [...new Set(logs.map((l) => l.level))], [logs]);
  const eventTypes = useMemo(() => [...new Set(logs.map((l) => l.eventType).filter(Boolean))] as string[], [logs]);
  const statuses = useMemo(() => [...new Set(logs.map((l) => l.status))], [logs]);

  const toggleFilter = (key: keyof Filters, value: string) => {
    const current = filters[key];
    onChange({
      ...filters,
      [key]: current.includes(value) ? current.filter((v) => v !== value) : [...current, value],
    });
  };

  return (
    <motion.div className="h-full overflow-y-auto p-4 space-y-6">
      <div className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{t("logsLevel")}</p>
        <div className="space-y-2">
          {levels.map((level) => {
            const selected = filters.level.includes(level);
            return (
              <motion.button
                key={level}
                type="button"
                whileHover={{ x: 2 }}
                onClick={() => toggleFilter("level", level)}
                className={`flex w-full items-center justify-between gap-2 border rounded-md px-3 py-2 text-sm transition-colors ${
                  selected ? "border-primary bg-primary/10 text-primary" : "border-border text-muted-foreground hover:border-primary/40"
                }`}
              >
                <span className="capitalize">{level}</span>
                {selected && <Check className="h-3.5 w-3.5" />}
              </motion.button>
            );
          })}
        </div>
      </div>

      <div className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{t("logsProject")}</p>
        <div className="space-y-2">
          {services.map((s) => {
            const selected = filters.service.includes(s);
            return (
              <motion.button
                key={s}
                type="button"
                whileHover={{ x: 2 }}
                onClick={() => toggleFilter("service", s)}
                className={`flex w-full items-center justify-between gap-2 border rounded-md px-3 py-2 text-sm transition-colors ${
                  selected ? "border-primary bg-primary/10 text-primary" : "border-border text-muted-foreground hover:border-primary/40"
                }`}
              >
                <span>{s}</span>
                {selected && <Check className="h-3.5 w-3.5" />}
              </motion.button>
            );
          })}
        </div>
      </div>

      {eventTypes.length > 0 && (
        <div className="space-y-3">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{t("logsEventType")}</p>
          <div className="space-y-2">
            {eventTypes.map((et) => {
              const selected = filters.eventType.includes(et);
              return (
                <motion.button
                  key={et}
                  type="button"
                  whileHover={{ x: 2 }}
                  onClick={() => toggleFilter("eventType", et)}
                  className={`flex w-full items-center justify-between gap-2 border rounded-md px-3 py-2 text-sm transition-colors ${
                    selected ? "border-primary bg-primary/10 text-primary" : "border-border text-muted-foreground hover:border-primary/40"
                  }`}
                >
                  <span>{et}</span>
                  {selected && <Check className="h-3.5 w-3.5" />}
                </motion.button>
              );
            })}
          </div>
        </div>
      )}

      <div className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{t("logsStatus")}</p>
        <div className="space-y-2">
          {statuses.map((status) => {
            const selected = filters.status.includes(status);
            return (
              <motion.button
                key={status}
                type="button"
                whileHover={{ x: 2 }}
                onClick={() => toggleFilter("status", status)}
                className={`flex w-full items-center justify-between gap-2 border rounded-md px-3 py-2 text-sm transition-colors ${
                  selected ? "border-primary bg-primary/10 text-primary" : "border-border text-muted-foreground hover:border-primary/40"
                }`}
              >
                <span className="capitalize">{status}</span>
                {selected && <Check className="h-3.5 w-3.5" />}
              </motion.button>
            );
          })}
        </div>
      </div>
    </motion.div>
  );
}

// ---------------------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------------------

interface ActivityLogsTableProps {
  logs: ActivityLog[];
  title?: string;
  subtitle?: string;
}

export function ActivityLogsTable({ logs, title, subtitle }: ActivityLogsTableProps) {
  const t = useTranslations("dashboard");
  const [searchQuery, setSearchQuery] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [showFilters, setShowFilters] = useState(false);
  const [filters, setFilters] = useState<Filters>({
    level: [],
    service: [],
    eventType: [],
    status: [],
  });

  const filteredLogs = useMemo(() => {
    return logs.filter((log) => {
      const lowerQuery = searchQuery.toLowerCase();
      const matchSearch =
        log.message.toLowerCase().includes(lowerQuery) ||
        log.service.toLowerCase().includes(lowerQuery);
      const matchLevel = filters.level.length === 0 || filters.level.includes(log.level);
      const matchService = filters.service.length === 0 || filters.service.includes(log.service);
      const matchEventType = filters.eventType.length === 0 || (log.eventType && filters.eventType.includes(log.eventType));
      const matchStatus = filters.status.length === 0 || filters.status.includes(log.status);
      return matchSearch && matchLevel && matchService && matchEventType && matchStatus;
    });
  }, [logs, filters, searchQuery]);

  const activeFilters = filters.level.length + filters.service.length + filters.eventType.length + filters.status.length;

  if (logs.length === 0) return null;

  return (
    <div className="overflow-hidden">
      {/* Header */}
      <div className="border-b border-border p-6">
        <div className="space-y-3">
          <div>
            <h2 className="text-lg font-semibold text-foreground">{title ?? t("logsTitle")}</h2>
            {subtitle && <p className="text-sm text-muted-foreground">{subtitle}</p>}
            <p className="text-xs text-muted-foreground mt-1">
              {filteredLogs.length} / {logs.length}
            </p>
          </div>
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder={t("logsSearch")}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="h-9 pl-9 text-sm"
              />
            </div>
            <Button
              variant={showFilters ? "default" : "outline"}
              size="sm"
              onClick={() => setShowFilters((c) => !c)}
              className="relative"
            >
              <Filter className="h-4 w-4" />
              {activeFilters > 0 && (
                <Badge className="absolute -right-2 -top-2 flex h-5 w-5 items-center justify-center p-0 text-xs" style={{ backgroundColor: "#ef4444" }}>
                  {activeFilters}
                </Badge>
              )}
            </Button>
          </div>
        </div>
      </div>

      {/* Body */}
      <div className="flex max-h-[500px] overflow-hidden">
        <AnimatePresence initial={false}>
          {showFilters && (
            <motion.div
              key="filters"
              initial={{ width: 0, opacity: 0 }}
              animate={{ width: 240, opacity: 1 }}
              exit={{ width: 0, opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="overflow-hidden border-r border-border flex-shrink-0"
            >
              <FilterPanel filters={filters} onChange={setFilters} logs={logs} />
            </motion.div>
          )}
        </AnimatePresence>

        <div className="flex-1 overflow-y-auto">
          <div className="divide-y divide-border">
            <AnimatePresence mode="popLayout">
              {filteredLogs.length > 0 ? (
                filteredLogs.map((log, index) => (
                  <motion.div
                    key={log.id}
                    initial={{ opacity: 0, y: -10 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -10 }}
                    transition={{ duration: 0.2, delay: index * 0.02 }}
                  >
                    <LogRow
                      log={log}
                      expanded={expandedId === log.id}
                      onToggle={() => setExpandedId((c) => (c === log.id ? null : log.id))}
                    />
                  </motion.div>
                ))
              ) : (
                <motion.div key="empty" initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="p-12 text-center">
                  <p className="text-muted-foreground">{t("noActivity")}</p>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </div>
      </div>
    </div>
  );
}
