"use client";

import { useRef } from "react";
import { useTranslations } from "next-intl";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Upload, Download, Trash2, FileText } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  listDocuments,
  uploadDocument,
  downloadDocument,
  deleteDocument,
} from "@/features/documents/api";
import type { Document } from "@/features/documents/api";

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

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface DocumentsListProps {
  projectId: string;
}

export function DocumentsList({ projectId }: DocumentsListProps) {
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

  const uploadMutation = useMutation({
    mutationFn: (file: File) => uploadDocument(projectId, file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["documents", projectId] });
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
          <Button
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
          {documents.map((doc) => {
            const ext = getFileExtension(doc.title);
            const colorClass =
              FILE_TYPE_COLORS[ext] ?? FILE_TYPE_COLORS.txt;

            return (
              <div
                key={doc.id}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <FileText className="h-5 w-5 shrink-0 text-muted-foreground" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{doc.title}</p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(doc.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  {ext && (
                    <Badge
                      variant="secondary"
                      className={colorClass}
                    >
                      {ext.toUpperCase()}
                    </Badge>
                  )}
                </div>
                <div className="flex items-center gap-1 shrink-0">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-muted-foreground hover:text-foreground"
                    onClick={() => handleDownload(doc)}
                  >
                    <Download className="h-4 w-4" />
                    <span className="sr-only">{t("download")}</span>
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-muted-foreground hover:text-destructive"
                    onClick={() => handleDelete(doc)}
                    disabled={deleteMutation.isPending}
                  >
                    <Trash2 className="h-4 w-4" />
                    <span className="sr-only">{t("delete")}</span>
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {deleteMutation.isError && (
        <p className="text-xs text-destructive mt-2">{t("deleteError")}</p>
      )}
    </div>
  );
}
