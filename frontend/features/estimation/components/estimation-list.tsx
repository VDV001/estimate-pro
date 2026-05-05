// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Send, Trash2, Clock, ChevronDown, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  listEstimations,
  getEstimation,
  submitEstimation,
  deleteEstimation,
  pertHours,
  type Estimation,
  type EstimationWithItems,
} from "@/features/estimation/api";
import { EstimationForm } from "./estimation-form";

// ---------------------------------------------------------------------------
// Status badge
// ---------------------------------------------------------------------------

const STATUS_STYLES: Record<string, string> = {
  draft: "bg-yellow-500/10 text-yellow-600 dark:text-yellow-400",
  submitted: "bg-green-500/10 text-green-600 dark:text-green-400",
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface EstimationListProps {
  projectId: string;
  initialTasks?: string[];
  onTasksConsumed?: () => void;
}

export function EstimationList({
  projectId,
  initialTasks,
  onTasksConsumed,
}: EstimationListProps) {
  const t = useTranslations("estimation");
  const tCommon = useTranslations("common");
  const queryClient = useQueryClient();

  // Open the form automatically when initialTasks arrived from
  // the parent. The page-level callsite remounts EstimationList
  // via a `key` prop tied to initialTasks identity, so the lazy
  // initializer runs on every fresh hand-off — no useEffect, no
  // cascading render.
  const [showForm, setShowForm] = useState(() =>
    Boolean(initialTasks && initialTasks.length > 0),
  );
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const {
    data: estimations,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["estimations", projectId, "mine"],
    queryFn: () => listEstimations(projectId, true),
  });

  const submitMutation = useMutation({
    mutationFn: (estId: string) => submitEstimation(projectId, estId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["estimations", projectId] });
      queryClient.invalidateQueries({ queryKey: ["estimations-aggregated", projectId] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (estId: string) => deleteEstimation(projectId, estId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["estimations", projectId] });
      queryClient.invalidateQueries({ queryKey: ["estimations-aggregated", projectId] });
      setExpandedId(null);
    },
  });

  const handleSubmit = (estId: string) => {
    submitMutation.mutate(estId);
  };

  const handleDelete = (estId: string) => {
    if (window.confirm(t("confirmDelete"))) {
      deleteMutation.mutate(estId);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-muted-foreground">
        <p className="text-sm">{tCommon("loading")}</p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex items-center justify-center py-12 text-destructive">
        <p className="text-sm">{tCommon("error")}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">{t("myEstimations")}</h3>
        <Button
          variant="outline"
          size="sm"
          className="gap-2"
          onClick={() => setShowForm((v) => !v)}
        >
          <Plus className="h-4 w-4" />
          {t("create")}
        </Button>
      </div>

      {/* Create form */}
      {showForm && (
        <div className="rounded-lg border p-4 space-y-3">
          <div>
            <h4 className="text-sm font-semibold">{t("createTitle")}</h4>
            <p className="text-xs text-muted-foreground">{t("createDesc")}</p>
          </div>
          <EstimationForm
            projectId={projectId}
            initialTasks={initialTasks}
            onCreated={() => {
              setShowForm(false);
              onTasksConsumed?.();
            }}
          />
        </div>
      )}

      {/* List */}
      {!estimations || estimations.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <Clock className="h-10 w-10 text-muted-foreground/30 mb-3" />
          <p className="text-sm font-medium text-muted-foreground">
            {t("noEstimations")}
          </p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            {t("noEstimationsDesc")}
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {estimations.map((est) => (
            <EstimationRow
              key={est.id}
              estimation={est}
              projectId={projectId}
              isExpanded={expandedId === est.id}
              onToggle={() => setExpandedId(expandedId === est.id ? null : est.id)}
              onSubmit={() => handleSubmit(est.id)}
              onDelete={() => handleDelete(est.id)}
              isSubmitting={submitMutation.isPending}
              isDeleting={deleteMutation.isPending}
            />
          ))}
        </div>
      )}

      {(submitMutation.isError || deleteMutation.isError) && (
        <p className="text-xs text-destructive">
          {submitMutation.isError ? t("submitError") : t("deleteError")}
        </p>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Row sub-component
// ---------------------------------------------------------------------------

interface EstimationRowProps {
  estimation: Estimation;
  projectId: string;
  isExpanded: boolean;
  onToggle: () => void;
  onSubmit: () => void;
  onDelete: () => void;
  isSubmitting: boolean;
  isDeleting: boolean;
}

function EstimationRow({
  estimation,
  projectId,
  isExpanded,
  onToggle,
  onSubmit,
  onDelete,
  isSubmitting,
  isDeleting,
}: EstimationRowProps) {
  const t = useTranslations("estimation");

  const { data: details } = useQuery({
    queryKey: ["estimation", estimation.id],
    queryFn: () => getEstimation(projectId, estimation.id),
    enabled: isExpanded,
  });

  const statusLabel = estimation.status === "draft" ? t("draft") : t("submitted");
  const isDraft = estimation.status === "draft";

  return (
    <div className="rounded-lg border">
      {/* Summary row */}
      <div
        role="button"
        tabIndex={0}
        onClick={onToggle}
        onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") onToggle(); }}
        className="flex items-center justify-between w-full px-4 py-3 text-left cursor-pointer"
      >
        <div className="flex items-center gap-3">
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
          <div>
            <p className="text-sm font-medium">
              {new Date(estimation.created_at).toLocaleDateString()}
            </p>
          </div>
          <Badge variant="secondary" className={STATUS_STYLES[estimation.status]}>
            {statusLabel}
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          {isDraft && (
            <>
              <Button
                size="sm"
                variant="outline"
                className="gap-1.5 h-7 text-xs"
                onClick={(e) => {
                  e.stopPropagation();
                  onSubmit();
                }}
                disabled={isSubmitting}
              >
                <Send className="h-3 w-3" />
                {isSubmitting ? t("submitting") : t("submit")}
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="gap-1.5 h-7 text-xs text-muted-foreground hover:text-destructive hover:border-destructive"
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete();
                }}
                disabled={isDeleting}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Expanded details */}
      {isExpanded && details && (
        <div className="border-t px-4 py-3">
          <ItemsTable items={details.items} />
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Items table (reused in row expand)
// ---------------------------------------------------------------------------

function ItemsTable({ items }: { items: EstimationWithItems["items"] }) {
  const t = useTranslations("estimation");

  if (!items || items.length === 0) return null;

  const total = items.reduce(
    (sum, item) => sum + pertHours(item.min_hours, item.likely_hours, item.max_hours),
    0
  );

  return (
    <div className="text-sm">
      <div className="grid grid-cols-[1fr_64px_64px_64px_72px] gap-2 text-xs font-medium text-muted-foreground pb-1 border-b">
        <span>{t("task")}</span>
        <span className="text-right">{t("minHours")}</span>
        <span className="text-right">{t("likelyHours")}</span>
        <span className="text-right">{t("maxHours")}</span>
        <span className="text-right">{t("pert")}</span>
      </div>
      {items.map((item) => {
        const pert = pertHours(item.min_hours, item.likely_hours, item.max_hours);
        return (
          <div
            key={item.id}
            className="grid grid-cols-[1fr_64px_64px_64px_72px] gap-2 py-1.5 border-b last:border-b-0"
          >
            <span className="truncate">{item.task_name}</span>
            <span className="text-right tabular-nums text-muted-foreground">
              {item.min_hours}
            </span>
            <span className="text-right tabular-nums text-muted-foreground">
              {item.likely_hours}
            </span>
            <span className="text-right tabular-nums text-muted-foreground">
              {item.max_hours}
            </span>
            <span className="text-right tabular-nums font-medium">
              {pert.toFixed(1)}
            </span>
          </div>
        );
      })}
      <div className="grid grid-cols-[1fr_64px_64px_64px_72px] gap-2 pt-2 font-semibold">
        <span>{t("total")}</span>
        <span />
        <span />
        <span />
        <span className="text-right tabular-nums">{total.toFixed(1)}</span>
      </div>
    </div>
  );
}
