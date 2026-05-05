// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useMutation } from "@tanstack/react-query";
import { Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { downloadReport, REPORT_FORMATS, type ReportFormat } from "../api";

interface DownloadReportButtonProps {
  projectId: string;
}

export function DownloadReportButton({ projectId }: DownloadReportButtonProps) {
  const t = useTranslations("report");
  const [open, setOpen] = useState(false);

  const mutation = useMutation({
    mutationFn: (format: ReportFormat) => downloadReport(projectId, format),
  });

  const handlePick = (format: ReportFormat) => {
    setOpen(false);
    mutation.mutate(format);
  };

  return (
    <div className="relative inline-block">
      <Button
        variant="outline"
        size="sm"
        className="gap-2"
        onClick={() => setOpen((v) => !v)}
        disabled={mutation.isPending}
      >
        <Download className="h-4 w-4" />
        {mutation.isPending ? t("downloading") : t("download")}
      </Button>

      {open && (
        <div className="absolute right-0 top-9 z-50 w-32 rounded-lg border bg-popover p-1 shadow-lg">
          {REPORT_FORMATS.map((format) => (
            <button
              key={format}
              type="button"
              onClick={() => handlePick(format)}
              className="w-full text-left text-sm px-2 py-1.5 rounded-md hover:bg-muted transition-colors"
            >
              {format}
            </button>
          ))}
        </div>
      )}

      {mutation.isError && (
        <p className="text-xs text-destructive mt-1">{t("error")}</p>
      )}
    </div>
  );
}
