// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

"use client";

import { useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Upload, Download, Trash2, FileText, CheckCircle2, Star, X, Plus, Hash } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  listDocuments,
  uploadDocument,
  downloadDocument,
  deleteDocument,
  getDocument,
  updateVersionFlags,
  setVersionTags,
  PREDEFINED_TAGS,
} from "@/features/documents/api";
import type { Document } from "@/features/documents/api";
import {
  listExtractions,
  requestExtraction,
} from "@/features/extraction/api";
import type { Extraction } from "@/features/extraction/api";
import { ExtractionPanel } from "@/features/extraction/components/extraction-panel";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getFileExtension(title: string): string {
  const dot = title.lastIndexOf(".");
  if (dot === -1) return "";
  return title.slice(dot + 1).toLowerCase();
}

const FILE_TYPE_COLORS: Record<string, string> = {
  pdf: "bg-red-500/10 text-red-600 dark:text-red-400",
  docx: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  doc: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  xlsx: "bg-green-500/10 text-green-600 dark:text-green-400",
  xls: "bg-green-500/10 text-green-600 dark:text-green-400",
  md: "bg-gray-500/10 text-gray-600 dark:text-gray-400",
  txt: "bg-gray-500/10 text-gray-600 dark:text-gray-400",
  csv: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
};

const ACCEPTED_TYPES = ".pdf,.docx,.doc,.xlsx,.xls,.md,.txt,.csv";

// Formats whose contents the extractor pipeline can read. Mirrors the
// readers shipped in backend/internal/shared/reader. Anything outside
// this set (images, archives, …) skips the auto-extraction step so we
// don't kick off a job that's guaranteed to fail.
const EXTRACTION_SUPPORTED_EXTENSIONS = new Set([
  "pdf",
  "docx",
  "txt",
  "md",
  "csv",
  "xlsx",
]);

function isExtractionSupported(filename: string): boolean {
  return EXTRACTION_SUPPORTED_EXTENSIONS.has(getFileExtension(filename));
}

const TAG_COLORS: Record<string, string> = {
  "на_подпись": "bg-blue-500/10 text-blue-500",
  "подписана": "bg-green-500/10 text-green-500",
  "на_правках": "bg-yellow-500/10 text-yellow-600",
  "отклонена": "bg-red-500/10 text-red-500",
  "черновик": "bg-gray-500/10 text-gray-500",
  "от_заказчика": "bg-purple-500/10 text-purple-500",
  "спорная": "bg-orange-500/10 text-orange-500",
  "архив": "bg-gray-500/10 text-gray-400",
  "срочно": "bg-red-600/20 text-red-600",
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface DocumentsListProps {
  projectId: string;
  onCreateEstimation?: (taskNames: string[]) => void;
}

export function DocumentsList({
  projectId,
  onCreateEstimation,
}: DocumentsListProps) {
  const t = useTranslations("documents");
  const tCommon = useTranslations("common");
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const {
    data: documents,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["documents", projectId],
    queryFn: () => listDocuments(projectId),
  });

  const { data: extractions } = useQuery({
    queryKey: ["extractions", projectId],
    queryFn: () => listExtractions(projectId),
  });

  const latestExtractionByDocument = (extractions ?? []).reduce<
    Record<string, Extraction>
  >((acc, ext) => {
    const existing = acc[ext.document_id];
    if (!existing || ext.created_at > existing.created_at) {
      acc[ext.document_id] = ext;
    }
    return acc;
  }, {});

  const [extractionKickoffFailed, setExtractionKickoffFailed] = useState(false);

  const uploadMutation = useMutation({
    mutationFn: async (file: File) => {
      const uploaded = await uploadDocument(projectId, file);
      return { uploaded, file };
    },
    onSuccess: async ({ uploaded, file }) => {
      queryClient.invalidateQueries({ queryKey: ["documents", projectId] });
      if (isExtractionSupported(file.name)) {
        try {
          await requestExtraction(
            projectId,
            uploaded.document.id,
            uploaded.version.id,
            uploaded.version.file_size,
          );
          queryClient.invalidateQueries({
            queryKey: ["extractions", projectId],
          });
          setExtractionKickoffFailed(false);
        } catch {
          // Upload itself succeeded; only the extraction kickoff failed.
          // Surface a non-blocking inline message so the user knows tasks
          // weren't extracted automatically and can re-upload if needed.
          setExtractionKickoffFailed(true);
        }
      }
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (docId: string) => deleteDocument(projectId, docId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", projectId] });
    },
  });

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      uploadMutation.mutate(file);
    }
    // Reset input so the same file can be re-selected
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const handleDownload = (doc: Document) => {
    downloadDocument(projectId, doc.id);
  };

  const handleDelete = (doc: Document) => {
    const message = t("confirmDelete", { name: doc.title });
    if (window.confirm(message)) {
      deleteMutation.mutate(doc.id);
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
    <div>
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-lg font-semibold">{t("title")}</h3>
        <div className="flex items-center gap-2">
          {uploadMutation.isError && (
            <p className="text-xs text-destructive">{t("uploadError")}</p>
          )}
          {extractionKickoffFailed && !uploadMutation.isError && (
            <p className="text-xs text-amber-600 dark:text-amber-400">
              {t("extractionKickoffFailed")}
            </p>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleUploadClick}
            disabled={uploadMutation.isPending}
            className="gap-2"
          >
            <Upload className="h-4 w-4" />
            {uploadMutation.isPending ? t("uploading") : t("upload")}
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept={ACCEPTED_TYPES}
            onChange={handleFileChange}
            className="hidden"
          />
        </div>
      </div>

      {!documents || documents.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <FileText className="h-10 w-10 text-muted-foreground/30 mb-3" />
          <p className="text-sm font-medium text-muted-foreground">
            {t("empty")}
          </p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            {t("emptyDesc")}
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {documents.map((doc) => (
            <DocumentCard
              key={doc.id}
              doc={doc}
              projectId={projectId}
              extractionId={latestExtractionByDocument[doc.id]?.id}
              onDownload={() => handleDownload(doc)}
              onDelete={() => handleDelete(doc)}
              onCreateEstimation={onCreateEstimation}
              isDeleting={deleteMutation.isPending}
            />
          ))}
        </div>
      )}

      {deleteMutation.isError && (
        <p className="text-xs text-destructive mt-2">{t("deleteError")}</p>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Document Card with flags + tags
// ---------------------------------------------------------------------------

function DocumentCard({
  doc,
  projectId,
  extractionId,
  onDownload,
  onDelete,
  onCreateEstimation,
  isDeleting,
}: {
  doc: Document;
  projectId: string;
  extractionId: string | undefined;
  onDownload: () => void;
  onDelete: () => void;
  onCreateEstimation?: (taskNames: string[]) => void;
  isDeleting: boolean;
}) {
  const t = useTranslations("documents");
  const queryClient = useQueryClient();
  const [showTagPicker, setShowTagPicker] = useState(false);

  const ext = getFileExtension(doc.title);
  const colorClass = FILE_TYPE_COLORS[ext] ?? FILE_TYPE_COLORS.txt;

  // Fetch document with version details
  const { data: detail } = useQuery({
    queryKey: ["document-detail", doc.id],
    queryFn: () => getDocument(projectId, doc.id),
  });

  const version = detail?.latest_version;

  const flagsMutation = useMutation({
    mutationFn: (flags: { is_signed: boolean; is_final: boolean }) =>
      updateVersionFlags(projectId, doc.id, version!.id, flags),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["document-detail"] });
      queryClient.invalidateQueries({ queryKey: ["documents", projectId] });
    },
  });

  const tagsMutation = useMutation({
    mutationFn: (tags: string[]) =>
      setVersionTags(projectId, doc.id, version!.id, tags),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["document-detail", doc.id] });
    },
  });

  const currentTags = version?.tags ?? [];

  const toggleTag = (tag: string) => {
    const newTags = currentTags.includes(tag)
      ? currentTags.filter((t) => t !== tag)
      : currentTags.length < 3 ? [...currentTags, tag] : currentTags;
    tagsMutation.mutate(newTags);
  };

  const removeTag = (tag: string) => {
    tagsMutation.mutate(currentTags.filter((t) => t !== tag));
  };

  return (
    <div className="rounded-lg border p-4 space-y-3">
      {/* Top row: file info + actions */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 min-w-0">
          <FileText className="h-5 w-5 shrink-0 text-muted-foreground" />
          <div className="min-w-0">
            <p className="text-sm font-medium truncate">{doc.title}</p>
            <p className="text-xs text-muted-foreground">
              {new Date(doc.created_at).toLocaleDateString()}
            </p>
          </div>
          {ext && (
            <Badge variant="secondary" className={colorClass}>
              {ext.toUpperCase()}
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-foreground" onClick={onDownload}>
            <Download className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-destructive" onClick={onDelete} disabled={isDeleting}>
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Flags row */}
      {version && (
        <div className="flex items-center gap-3 flex-wrap">
          {/* Signed checkbox */}
          <button
            onClick={() => flagsMutation.mutate({ is_signed: !version.is_signed, is_final: version.is_final })}
            className={`inline-flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border transition-colors ${
              version.is_signed
                ? "border-green-500/30 text-green-500"
                : "border-border text-muted-foreground hover:text-foreground"
            }`}
          >
            <CheckCircle2 className="h-3.5 w-3.5" />
            {version.is_signed ? t("signed") : t("sign")}
          </button>

          {/* Final checkbox */}
          <button
            onClick={() => flagsMutation.mutate({ is_signed: version.is_signed, is_final: !version.is_final })}
            className={`inline-flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border transition-colors ${
              version.is_final
                ? "border-amber-500/30 text-amber-500"
                : "border-border text-muted-foreground hover:text-foreground"
            }`}
          >
            <Star className="h-3.5 w-3.5" />
            {version.is_final ? t("finalVersion") : t("setFinal")}
          </button>

          {/* Tags */}
          {currentTags.map((tag) => (
            <span
              key={tag}
              className={`inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full ${TAG_COLORS[tag] ?? "bg-muted text-muted-foreground"}`}
            >
              #{tag}
              <button onClick={() => removeTag(tag)} className="hover:text-foreground">
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}

          {/* Add tag button */}
          {currentTags.length < 3 && (
            <div className="relative">
              <button
                onClick={() => setShowTagPicker(!showTagPicker)}
                className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full border border-dashed border-border text-muted-foreground hover:text-foreground transition-colors"
              >
                <Hash className="h-3 w-3" />
                <Plus className="h-3 w-3" />
              </button>

              {showTagPicker && (
                <div className="absolute top-8 left-0 z-50 w-52 rounded-lg border bg-popover p-2 shadow-lg space-y-1">
                  {PREDEFINED_TAGS.filter((tag) => !currentTags.includes(tag)).map((tag) => (
                    <button
                      key={tag}
                      onClick={() => { toggleTag(tag); setShowTagPicker(false); }}
                      className={`w-full text-left text-xs px-2 py-1.5 rounded-md hover:bg-muted transition-colors ${TAG_COLORS[tag] ?? ""}`}
                    >
                      #{tag}
                    </button>
                  ))}
                  <form
                    onSubmit={(e) => {
                      e.preventDefault();
                      const input = e.currentTarget.elements.namedItem("customTag") as HTMLInputElement;
                      const val = input.value.trim().replace(/^#/, "").replace(/\s/g, "_");
                      if (val && !currentTags.includes(val)) {
                        toggleTag(val);
                        setShowTagPicker(false);
                      }
                    }}
                    className="flex gap-1 pt-1 border-t border-border mt-1"
                  >
                    <input
                      name="customTag"
                      placeholder="свой тег..."
                      className="flex-1 text-xs px-2 py-1 rounded-md bg-background border border-border focus:outline-none"
                      autoFocus
                    />
                    <button type="submit" className="text-xs px-2 py-1 rounded-md hover:bg-muted">
                      <Plus className="h-3 w-3" />
                    </button>
                  </form>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {extractionId && (
        <ExtractionPanel
          extractionId={extractionId}
          onCreateEstimation={onCreateEstimation}
        />
      )}
    </div>
  );
}
