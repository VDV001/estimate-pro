// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useTranslations } from "next-intl";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { cancelExtraction, retryExtraction } from "../api";
import {
  isTerminalStatus,
  useExtractionStatus,
} from "../hooks/use-extraction-status";
import { mapExtractionReason } from "../lib/reason-mapping";
import { ExtractionStatusBadge } from "./extraction-status-badge";

interface ExtractionPanelProps {
  extractionId: string;
}

export function ExtractionPanel({ extractionId }: ExtractionPanelProps) {
  const t = useTranslations("extraction");
  const tCommon = useTranslations("common");
  const queryClient = useQueryClient();
  const { data, isLoading, isError } = useExtractionStatus(extractionId);

  const cancelMutation = useMutation({
    mutationFn: () => cancelExtraction(extractionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["extraction", extractionId] });
    },
  });

  const retryMutation = useMutation({
    mutationFn: () => retryExtraction(extractionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["extraction", extractionId] });
    },
  });

  if (isLoading && !data) {
    return (
      <div className="rounded-lg border p-3 text-sm text-muted-foreground">
        {tCommon("loading")}
      </div>
    );
  }

  if (isError && !data) return null;
  if (!data) return null;

  const { status, tasks, failure_reason } = data.extraction;
  const inFlight = !isTerminalStatus(status);
  const reasonKey = mapExtractionReason(
    status === "cancelled" ? "cancelled" : failure_reason,
  );

  const handleCancel = () => {
    if (window.confirm(t("confirmCancel"))) {
      cancelMutation.mutate();
    }
  };

  const handleRetry = () => {
    retryMutation.mutate();
  };

  return (
    <div className="rounded-lg border p-3 space-y-3">
      <div className="flex items-center justify-between gap-3">
        <ExtractionStatusBadge status={status} />
        <div className="flex items-center gap-2">
          {inFlight && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleCancel}
              disabled={cancelMutation.isPending}
            >
              {t("actions.cancel")}
            </Button>
          )}
          {(status === "failed" || status === "cancelled") && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleRetry}
              disabled={retryMutation.isPending}
            >
              {t("actions.retry")}
            </Button>
          )}
        </div>
      </div>

      {(status === "failed" || status === "cancelled") && (
        <p className="text-sm text-destructive">{t(`reason.${reasonKey}`)}</p>
      )}

      {status === "completed" && (
        <div>
          {tasks.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("noTasks")}</p>
          ) : (
            <>
              <p className="text-xs text-muted-foreground mb-2">
                {t("tasksFound", { count: tasks.length })}
              </p>
              <ul className="space-y-1.5">
                {tasks.map((task, idx) => (
                  <li
                    key={`${task.name}-${idx}`}
                    className="flex items-baseline gap-2 text-sm"
                  >
                    <span className="font-medium">{task.name}</span>
                    {task.estimate_hint && (
                      <span className="text-xs text-muted-foreground">
                        {task.estimate_hint}
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>
      )}
    </div>
  );
}
