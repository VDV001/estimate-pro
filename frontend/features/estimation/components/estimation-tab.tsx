// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { EstimationList } from "./estimation-list";
import { AggregatedView } from "./aggregated-view";

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

type SubTab = "my" | "aggregated";

interface EstimationTabProps {
  projectId: string;
  initialTasks?: string[];
  onTasksConsumed?: () => void;
}

export function EstimationTab({
  projectId,
  initialTasks,
  onTasksConsumed,
}: EstimationTabProps) {
  const t = useTranslations("estimation");
  // Pre-filled tasks always belong to the "my" sub-tab. The parent
  // remounts EstimationTab via a `key` prop tied to initialTasks
  // identity, so the lazy initializer runs on every fresh hand-off
  // and we don't need a useEffect to flip the sub-tab.
  const [subTab, setSubTab] = useState<SubTab>(() =>
    initialTasks && initialTasks.length > 0 ? "my" : "my",
  );

  const subTabs: { key: SubTab; label: string }[] = [
    { key: "my", label: t("myEstimations") },
    { key: "aggregated", label: t("aggregate") },
  ];

  return (
    <div className="space-y-4">
      {/* Sub-tab navigation */}
      <div className="flex gap-1">
        {subTabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setSubTab(tab.key)}
            className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
              subTab === tab.key
                ? "border border-border text-foreground"
                : "text-muted-foreground hover:text-foreground hover:bg-muted"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      {subTab === "my" && (
        <EstimationList
          projectId={projectId}
          initialTasks={initialTasks}
          onTasksConsumed={onTasksConsumed}
        />
      )}
      {subTab === "aggregated" && <AggregatedView projectId={projectId} />}
    </div>
  );
}
