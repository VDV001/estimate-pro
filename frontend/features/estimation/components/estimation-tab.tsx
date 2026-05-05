// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useEffect, useState } from "react";
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
  const [subTab, setSubTab] = useState<SubTab>("my");

  // Pre-filled tasks always belong to the "my" sub-tab where the
  // EstimationForm lives — pull the user there if they were routed
  // here from the extraction panel.
  useEffect(() => {
    if (initialTasks && initialTasks.length > 0) {
      setSubTab("my");
    }
  }, [initialTasks]);

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
