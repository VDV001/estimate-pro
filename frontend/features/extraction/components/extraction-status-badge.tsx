// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useTranslations } from "next-intl";
import {
  Ban,
  CheckCircle2,
  CircleX,
  Clock,
  LoaderCircle,
} from "lucide-react";
import type { ComponentType, SVGProps } from "react";
import type { ExtractionStatus } from "../api";

interface ExtractionStatusBadgeProps {
  status: ExtractionStatus;
}

type IconComponent = ComponentType<SVGProps<SVGSVGElement>>;

const ICON_BY_STATUS: Record<ExtractionStatus, IconComponent> = {
  pending: Clock,
  processing: LoaderCircle,
  completed: CheckCircle2,
  failed: CircleX,
  cancelled: Ban,
};

const COLOR_BY_STATUS: Record<ExtractionStatus, string> = {
  pending: "bg-muted text-muted-foreground",
  processing: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  completed: "bg-green-500/10 text-green-600 dark:text-green-400",
  failed: "bg-red-500/10 text-red-600 dark:text-red-400",
  cancelled: "bg-gray-500/10 text-gray-500",
};

export function ExtractionStatusBadge({ status }: ExtractionStatusBadgeProps) {
  const t = useTranslations("extraction.status");
  const Icon = ICON_BY_STATUS[status];
  const color = COLOR_BY_STATUS[status];
  const iconClass =
    status === "processing" ? "h-3.5 w-3.5 animate-spin" : "h-3.5 w-3.5";

  return (
    <span
      role="status"
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${color}`}
    >
      <Icon className={iconClass} />
      {t(status)}
    </span>
  );
}
