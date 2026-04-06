// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  createEstimation,
  pertHours,
  type CreateEstimationItemInput,
} from "@/features/estimation/api";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface TaskRow {
  key: string;
  task_name: string;
  min_hours: string;
  likely_hours: string;
  max_hours: string;
  note: string;
}

function emptyRow(): TaskRow {
  return {
    key: crypto.randomUUID(),
    task_name: "",
    min_hours: "",
    likely_hours: "",
    max_hours: "",
    note: "",
  };
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface EstimationFormProps {
  projectId: string;
  onCreated?: () => void;
}

export function EstimationForm({ projectId, onCreated }: EstimationFormProps) {
  const t = useTranslations("estimation");
  const queryClient = useQueryClient();

  const [rows, setRows] = useState<TaskRow[]>([emptyRow()]);

  const mutation = useMutation({
    mutationFn: (items: CreateEstimationItemInput[]) =>
      createEstimation(projectId, { items }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["estimations", projectId] });
      queryClient.invalidateQueries({ queryKey: ["estimations-aggregated", projectId] });
      setRows([emptyRow()]);
      onCreated?.();
    },
  });

  const updateRow = (index: number, field: keyof TaskRow, value: string) => {
    setRows((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], [field]: value };
      return next;
    });
  };

  const addRow = () => {
    setRows((prev) => [...prev, emptyRow()]);
  };

  const removeRow = (index: number) => {
    setRows((prev) => prev.filter((_, i) => i !== index));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const validRows = rows.filter((r) => r.task_name.trim() !== "");
    if (validRows.length === 0) return;

    const items: CreateEstimationItemInput[] = validRows.map((r, i) => ({
      task_name: r.task_name.trim(),
      min_hours: parseFloat(r.min_hours) || 0,
      likely_hours: parseFloat(r.likely_hours) || 0,
      max_hours: parseFloat(r.max_hours) || 0,
      sort_order: i,
      note: r.note.trim() || undefined,
    }));

    mutation.mutate(items);
  };

  const totalPert = rows.reduce((sum, r) => {
    const min = parseFloat(r.min_hours) || 0;
    const likely = parseFloat(r.likely_hours) || 0;
    const max = parseFloat(r.max_hours) || 0;
    return sum + pertHours(min, likely, max);
  }, 0);

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="rounded-lg border">
        {/* Header */}
        <div className="grid grid-cols-[1fr_80px_80px_80px_80px_40px] gap-2 px-3 py-2 border-b bg-muted/50 text-xs font-medium text-muted-foreground">
          <span>{t("task")}</span>
          <span className="text-center">{t("minHours")}</span>
          <span className="text-center">{t("likelyHours")}</span>
          <span className="text-center">{t("maxHours")}</span>
          <span className="text-center">{t("pert")}</span>
          <span />
        </div>

        {/* Rows */}
        {rows.map((row, index) => {
          const min = parseFloat(row.min_hours) || 0;
          const likely = parseFloat(row.likely_hours) || 0;
          const max = parseFloat(row.max_hours) || 0;
          const pert = pertHours(min, likely, max);

          return (
            <div
              key={row.key}
              className="grid grid-cols-[1fr_80px_80px_80px_80px_40px] gap-2 px-3 py-1.5 border-b last:border-b-0 items-center"
            >
              <Input
                value={row.task_name}
                onChange={(e) => updateRow(index, "task_name", e.target.value)}
                placeholder={t("taskPlaceholder")}
                className="h-8 text-sm"
              />
              <Input
                type="number"
                min="0"
                step="0.5"
                value={row.min_hours}
                onChange={(e) => updateRow(index, "min_hours", e.target.value)}
                className="h-8 text-sm text-center"
              />
              <Input
                type="number"
                min="0"
                step="0.5"
                value={row.likely_hours}
                onChange={(e) => updateRow(index, "likely_hours", e.target.value)}
                className="h-8 text-sm text-center"
              />
              <Input
                type="number"
                min="0"
                step="0.5"
                value={row.max_hours}
                onChange={(e) => updateRow(index, "max_hours", e.target.value)}
                className="h-8 text-sm text-center"
              />
              <span className="text-sm text-center font-medium tabular-nums">
                {pert > 0 ? pert.toFixed(1) : "—"}
              </span>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-8 w-8 text-muted-foreground hover:text-destructive"
                onClick={() => removeRow(index)}
                disabled={rows.length <= 1}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          );
        })}

        {/* Total row */}
        <div className="grid grid-cols-[1fr_80px_80px_80px_80px_40px] gap-2 px-3 py-2 bg-muted/30 text-sm font-semibold">
          <span>{t("total")}</span>
          <span />
          <span />
          <span />
          <span className="text-center tabular-nums">
            {totalPert > 0 ? totalPert.toFixed(1) : "—"}
          </span>
          <span />
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between">
        <Button
          type="button"
          size="sm"
          className="gap-2"
          onClick={addRow}
        >
          <Plus className="h-4 w-4" />
          {t("addTask")}
        </Button>

        <div className="flex items-center gap-2">
          {mutation.isError && (
            <p className="text-xs text-destructive">{t("createError")}</p>
          )}
          <Button
            type="submit"
            size="sm"
            disabled={mutation.isPending || rows.every((r) => !r.task_name.trim())}
          >
            {mutation.isPending ? t("creating") : t("create")}
          </Button>
        </div>
      </div>
    </form>
  );
}
