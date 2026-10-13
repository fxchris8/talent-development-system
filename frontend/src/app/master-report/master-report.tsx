"use client";

import * as React from "react";
import { useState, useRef, useEffect, useCallback } from "react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import Image from "next/image";
import { Separator } from "@radix-ui/react-separator";
import { api } from "@/app/lib/api";

import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
  CommandInput,
  CommandEmpty,
} from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import { TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import {
  CheckIcon,
  ChevronsUpDownIcon,
  PlusIcon,
  EditIcon,
  TrashIcon,
  XIcon,
  AlertTriangle,
  Loader2,
  RefreshCw,
} from "lucide-react";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationPrevious,
  PaginationNext,
  PaginationLink,
} from "@/components/ui/pagination";
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { useMasterReports } from "./_hooks/master-report";
import { useMasterReportUI } from "./_hooks/useMasterReportUI";
import { useGetAllAssessmentTypes } from "./_hooks/useAssessmentType";
import type { IReport, SeamanLookup } from "@/types/global-types";
import { TrainingStatusBadge } from "@/components/training-status-badge";
import { useAvailableSeamen } from "./_hooks/useAvailableSeamen";
import { useBatches, type Batch } from "./_hooks/useBatch";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";

type ReportPickerState = {
  items: IReport[];
  query: string;
  anchorId: number | null;
  hasMore: boolean;
  isLoading: boolean;
  isLoadingMore: boolean;
};

const EMPTY_REPORT_PICKER: ReportPickerState = {
  items: [],
  query: "",
  anchorId: 0,
  hasMore: true,
  isLoading: false,
  isLoadingMore: false,
};

const REPORT_PICKER_PAGE_SIZE = 20;

/* Dynamic sticky offset calculator */
function useDynamicStickyOffsets(ref: React.RefObject<HTMLDivElement | null>, pinnedCount = 2) {
  const [offsets, setOffsets] = useState<number[]>([]);
  useEffect(() => {
    const container = ref.current;
    if (!container) return;
    const updateOffsets = () => {
      const heads = container.querySelectorAll<HTMLTableCellElement>("thead th");
      const newOffsets: number[] = [];
      let runningLeft = 0;
      for (let i = 0; i < pinnedCount; i++) {
        newOffsets.push(runningLeft);
        runningLeft += heads[i]?.offsetWidth ?? 0;
      }
      setOffsets(newOffsets);
    };
    const observer = new ResizeObserver(updateOffsets);
    observer.observe(container);
    updateOffsets();
    return () => observer.disconnect();
  }, [ref, pinnedCount]);
  return offsets;
}

export default function MasterPage({
  onBatchChange,
}: {
  onBatchChange?: (batchId: number | null) => void;
}) {
  // Data hooks
  const {
    onCallApi,
    paginationData,
    paginationRequest,
    setPaginationRequest,
    setPageSize,
    searchName,
    setSearchName,
    createReport,
    deleteReport,
    updateReport,
    bulkAssignBatch,
    refreshAllReadiness,
    refreshPersonalData,
  } = useMasterReports(10);

  // Bulk assign to batch dialog state
  const [bulkAssignBatchOpen, setBulkAssignBatchOpen] = useState(false);
  const [selectedBatchForAssign, setSelectedBatchForAssign] = useState<string>("");
  const [selectAllGlobal, setSelectAllGlobal] = useState(false);
  const [selectedReportIdsForBatch, setSelectedReportIdsForBatch] = useState<number[]>([]);
  const [reportPicker, setReportPicker] = useState<ReportPickerState>(EMPTY_REPORT_PICKER);

  const [addForm, setAddForm] = useState<{
    search: string;
    selected?: SeamanLookup;
    selectedBulk?: SeamanLookup[];
    bulkAddMode?: boolean;
  }>({
    search: "",
    selectedBulk: [],
    bulkAddMode: false,
  });

  const { data: seamen, loading } = useAvailableSeamen(addForm.search);

  const [seamanResults, setSeamanResults] = useState<SeamanLookup[]>([]);

  // Sync seamen data from hook to state whenever it changes
  useEffect(() => {
    if (seamen && seamen.length > 0) {
      // Only update if it's different to avoid unnecessary re-renders
      setSeamanResults((prevResults) => {
        const isEqual = JSON.stringify(prevResults) === JSON.stringify(seamen);
        return isEqual ? prevResults : seamen;
      });
    } else {
      setSeamanResults([]);
    }
  }, [seamen]);
  const { data: assessmentTypes = [] } = useGetAllAssessmentTypes();

  // Batch hooks
  const { batches, loading: loadingBatches } = useBatches();

  // Track whether we've already set the initial default batch

  // Set default batch ID to the latest batch when first loaded — only once
  // Notify parent whenever the active batch changes (for shared batch filter)
  useEffect(() => {
    onBatchChange?.(paginationRequest.batchId ?? null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paginationRequest.batchId]);

  // UI hooks
  const {
    openDialog,
    setOpenDialog,
    confirmDeleteDialog,
    setConfirmDeleteDialog,
    openEditDialog,
    setOpenEditDialog,
    editingRow,
    setEditingRow,
    isEditMode,
    setIsEditMode,
    selectedIds,
    setSelectedIds,
    currentPage,
    setCurrentPage,
    competencyTypes,
    selectedCompetencies,
    setSelectedCompetencies,
    competencySearchOpen,
    setCompetencySearchOpen,
    loadingCompetencies,
    linkedMentoringReports,
    loadingLinkedMentoring,
    fetchLinkedMentoringReports,
    mentoringDetailsDialogOpen,
    setMentoringDetailsDialogOpen,
    selectedPersonForMentoring,
    setSelectedPersonForMentoring,
    isEditFormValid,
    toggleCompetencySelection,
    removeCompetency,
  } = useMasterReportUI();

  // Reset currentPage to 1 when API returns empty results (navigated past last page)
  useEffect(() => {
    if (
      paginationData &&
      paginationData.results?.length === 0 &&
      !paginationData.first_page &&
      currentPage > 1
    ) {
      setCurrentPage(1);
    }
  }, [paginationData, currentPage, setCurrentPage]);

  const tableRef = useRef<HTMLDivElement>(null);
  const offsets = useDynamicStickyOffsets(tableRef, 2);
  const skipNextReportPickerSearchRef = useRef(false);

  // Derived: true when the selected batch is completed (data comes from snapshot)
  const isArchived = batches.some(
    (b: Batch) => b.id === paginationRequest.batchId && b.status === "completed"
  );

  const PAGE_SIZES = [10, 20, 30, 50, 100];

  // Build dynamic columns with assessment types
  const STATIC_COLUMNS = [
    "No",
    "Name",
    "Seaman Code",
    "Seafarer Code",
    "Vessel Name",
    "Position",
    "IDP Program",
    "Training Status",
    "Age",
    "Certificate",
    "Kondite Review",
    "KPI Vessel",
    "Performance Score",
    "Competency Gap Analysis",
    "Total Gap",
    "Strength Analysis",
    "Hav Quadran",
    "Talent Classified",
    "Readiness",
    "Certificate Eligible",
    "Actions",
  ];

  // Add assessment type columns dynamically
  const assessmentTypeColumns = assessmentTypes.map((type) => type.assessmentTypeName);

  const TABLE_COLUMNS = [...STATIC_COLUMNS, ...assessmentTypeColumns];

  const navigatePage = (page: "prev" | "next") => {
    if (!paginationData) return;
    setCurrentPage((prev) => {
      const newPage = page === "prev" ? Math.max(1, prev - 1) : prev + 1;
      return newPage;
    });

    setPaginationRequest({
      ...paginationRequest,
      page,
      anchorId: page === "next" ? paginationData.last_id : paginationData.first_id,
    });
  };

  const handleAdd = async () => {
    const toAdd = addForm.bulkAddMode
      ? addForm.selectedBulk || []
      : addForm.selected
        ? [addForm.selected]
        : [];

    if (toAdd.length === 0) {
      toast.error("Pilih seaman terlebih dahulu");
      return;
    }

    try {
      // Add all selected seamen
      for (const seaman of toAdd) {
        await createReport({
          nama: seaman.name,
          seamanCode: seaman.seamanCode,
        });
      }

      toast.success(`${toAdd.length} report(s) added successfully!`);
      setAddForm({
        search: "",
        selectedBulk: [],
        bulkAddMode: false,
      });
      setOpenDialog(false);

      setPaginationRequest({
        ...paginationRequest,
        anchorId: 0,
        page: "next",
      });
    } catch (err: unknown) {
      const error = err as {
        response?: { data?: { error?: string } };
      };
      toast.error(error.response?.data?.error || "Failed to add report(s)");
    }
  };

  const toggleBulkSelection = (seaman: SeamanLookup) => {
    const currentSelected = addForm.selectedBulk || [];
    const isSelected = currentSelected.some((s) => s.seamanCode === seaman.seamanCode);

    if (isSelected) {
      setAddForm({
        ...addForm,
        selectedBulk: currentSelected.filter((s) => s.seamanCode !== seaman.seamanCode),
      });
    } else {
      setAddForm({
        ...addForm,
        selectedBulk: [...currentSelected, seaman],
      });
    }
  };

  const toggleBulkMode = () => {
    setAddForm({
      ...addForm,
      bulkAddMode: !addForm.bulkAddMode,
      selected: undefined,
      selectedBulk: [],
    });
  };

  const handleEdit = async () => {
    if (!isEditFormValid()) return toast.error("Please fill in all fields!");
    if (!editingRow) return;
    try {
      const updatePayload: Partial<IReport> = {
        nama: editingRow.nama,
        seamanCode: editingRow.seamanCode,
        seafarerCode: editingRow.seafarerCode,
      };

      // Add optional fields if they have values
      if (editingRow.vesselName) {
        updatePayload.vesselName = editingRow.vesselName;
      }
      if (editingRow.jabatan) {
        updatePayload.jabatan = editingRow.jabatan;
      }
      if (editingRow.age) {
        updatePayload.age = editingRow.age;
      }
      if (editingRow.certificate) {
        updatePayload.certificate = editingRow.certificate;
      }
      if (editingRow.idpProgram) {
        updatePayload.idpProgram = editingRow.idpProgram;
      }
      if (editingRow.performanceScore) {
        updatePayload.performanceScore = editingRow.performanceScore;
      }
      if (
        editingRow.totalReadinessUpdateMonths !== undefined &&
        editingRow.totalReadinessUpdateMonths !== null
      ) {
        updatePayload.totalReadinessUpdateMonths = editingRow.totalReadinessUpdateMonths;
      }
      if (editingRow.talentClassified) {
        updatePayload.talentClassified = editingRow.talentClassified;
      }

      // Add competencies if they were modified
      if (selectedCompetencies.length > 0) {
        updatePayload.competencies = selectedCompetencies.map((typeId) => ({
          competencyTypeId: typeId,
        }));
      } else {
        // Send empty array to clear competencies
        updatePayload.competencies = [];
      }

      // Add assessment type scores if they exist
      if (editingRow.reportScores && editingRow.reportScores.length > 0) {
        updatePayload.reportScores = editingRow.reportScores;
      }

      await updateReport(editingRow.id, updatePayload);

      setOpenEditDialog(false);
      setEditingRow(null);
      setPaginationRequest({ ...paginationRequest });
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      toast.error(error.response?.data?.error || "Failed to update report");
    }
  };

  const toggleEditMode = () => {
    if (isEditMode) {
      setSelectedIds(new Set());
      setSelectAllGlobal(false);
    }
    setIsEditMode((prev) => !prev);
  };

  const toggleRowSelection = (id: number) => {
    const newSet = new Set(selectedIds);
    if (newSet.has(id)) {
      newSet.delete(id);
    } else {
      newSet.add(id);
    }
    setSelectedIds(newSet);
  };

  const toggleSelectAll = () => {
    if (!paginationData?.results) return;
    const ids = paginationData.results.map((r) => r.id);
    const allSelected = ids.every((id) => selectedIds.has(id));
    const newSet = new Set(selectedIds);
    if (allSelected) {
      ids.forEach((id) => newSet.delete(id));
      setSelectAllGlobal(false); // Reset global select-all
    } else {
      ids.forEach((id) => newSet.add(id));
    }
    setSelectedIds(newSet);
  };

  const confirmDelete = () => {
    if (selectedIds.size === 0) {
      toast.error("No rows selected!");
      return;
    }
    setConfirmDeleteDialog(true);
  };

  const handleDeleteConfirmed = async () => {
    setConfirmDeleteDialog(false);
    try {
      for (const id of selectedIds) await deleteReport(id);
      setSelectedIds(new Set());
      setPaginationRequest({ ...paginationRequest, anchorId: 0, page: "next" });
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      toast.error(error.response?.data?.error || "Delete failed");
    }
  };

  const handleRowClick = (row: IReport) => {
    if (!isEditMode) return;
    setEditingRow({ ...row });

    // Set selected competencies from row data
    const competencyIds =
      row.competencies
        ?.map((c) => c.competencyTypeId)
        .filter((id): id is number => id !== undefined) || [];
    setSelectedCompetencies(competencyIds);

    // Fetch mentoring reports linked to this person
    if (row.nama) {
      fetchLinkedMentoringReports(row.nama);
    }

    setOpenEditDialog(true);
  };

  const getRowNumber = (i: number) => (currentPage - 1) * paginationRequest.pageSize + i + 1;

  const handleViewMentoringPrograms = async (row: IReport, e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent row click when clicking the button
    setSelectedPersonForMentoring(row);
    await fetchLinkedMentoringReports(row.nama);
    setMentoringDetailsDialogOpen(true);
  };

  function colorFromString(str: string | undefined | null) {
    if (!str) return "hsl(200, 70%, 70%)"; // Default color if undefined
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      hash = str.charCodeAt(i) + ((hash << 5) - hash);
    }
    const hue = Math.abs(hash) % 360;
    return `hsl(${hue}, 70%, 70%)`;
  }

  // Helper function to get score for assessment type
  const getScoreForAssessmentType = (
    row: IReport & {
      reportScores?: Array<{ score?: number; assessmentType?: { assessmentTypeName?: string } }>;
    },
    assessmentTypeName: string
  ): number => {
    if (!row.reportScores || !Array.isArray(row.reportScores)) {
      return 0;
    }
    const scoreEntry = row.reportScores.find(
      (rs) => rs.assessmentType?.assessmentTypeName === assessmentTypeName
    );
    return scoreEntry?.score ?? 0;
  };

  const isAllCurrentPageSelected = () => {
    const results = paginationData?.results;
    if (!results || results.length === 0) return false;
    return results.every((r) => selectedIds.has(r.id));
  };

  const fetchReportPicker = useCallback(
    async (options?: { reset?: boolean; query?: string; anchorId?: number | null }) => {
      const reset = options?.reset ?? false;
      const query = options?.query ?? "";
      const anchorId = reset ? 0 : (options?.anchorId ?? 0);

      setReportPicker((prev) => ({
        ...prev,
        ...(reset ? { items: [], anchorId: 0, hasMore: true } : {}),
        ...(reset ? { isLoading: true } : { isLoadingMore: true }),
      }));

      try {
        const params = new URLSearchParams({
          page: "next",
          page_size: REPORT_PICKER_PAGE_SIZE.toString(),
          anchor_id: String(anchorId ?? 0),
        });

        if (query.trim()) params.set("query", query.trim());
        if (paginationRequest.batchId !== null && paginationRequest.batchId !== undefined) {
          params.set("batch_id", paginationRequest.batchId.toString());
        }

        const response = await api.get(`/api/master-reports?${params.toString()}`);
        const payload = response.data?.data;
        const reports = Array.isArray(payload?.data)
          ? (payload.data as IReport[])
          : Array.isArray(response.data?.data)
            ? (response.data.data as IReport[])
            : Array.isArray(response.data)
              ? (response.data as IReport[])
              : [];

        setReportPicker((prev) => ({
          ...prev,
          items: reset ? reports : [...prev.items, ...reports],
          anchorId: payload?.lastId ?? reports.at(-1)?.id ?? null,
          hasMore: payload?.hasMore ?? reports.length >= REPORT_PICKER_PAGE_SIZE,
          isLoading: false,
          isLoadingMore: false,
        }));
      } catch {
        setReportPicker((prev) => ({
          ...prev,
          isLoading: false,
          isLoadingMore: false,
        }));
        toast.error("Gagal memuat report");
      }
    },
    [paginationRequest.batchId]
  );

  useEffect(() => {
    if (!bulkAssignBatchOpen) {
      skipNextReportPickerSearchRef.current = false;
      return;
    }
    skipNextReportPickerSearchRef.current = true;
    fetchReportPicker({ reset: true, query: reportPicker.query });
  }, [bulkAssignBatchOpen, paginationRequest.batchId, fetchReportPicker, reportPicker.query]);

  useEffect(() => {
    if (!bulkAssignBatchOpen) return;
    if (skipNextReportPickerSearchRef.current) {
      skipNextReportPickerSearchRef.current = false;
      return;
    }
    const timeout = setTimeout(() => {
      fetchReportPicker({ reset: true, query: reportPicker.query });
    }, 300);
    return () => clearTimeout(timeout);
  }, [bulkAssignBatchOpen, reportPicker.query, fetchReportPicker]);

  const toggleReportSelectionForBatch = (reportId: number) => {
    setSelectedReportIdsForBatch((prev) =>
      prev.includes(reportId) ? prev.filter((id) => id !== reportId) : [...prev, reportId]
    );
  };

  const toggleSelectAllReportsForBatch = () => {
    const visibleIds = reportPicker.items.map((report) => report.id);
    const allSelected =
      visibleIds.length > 0 &&
      visibleIds.every((reportId) => selectedReportIdsForBatch.includes(reportId));

    if (allSelected) {
      const visibleSet = new Set(visibleIds);
      setSelectedReportIdsForBatch((prev) => prev.filter((id) => !visibleSet.has(id)));
      return;
    }

    setSelectedReportIdsForBatch((prev) => Array.from(new Set([...prev, ...visibleIds])));
  };

  const handleReportPickerScroll = (event: React.UIEvent<HTMLDivElement>) => {
    if (!reportPicker.hasMore || reportPicker.isLoading || reportPicker.isLoadingMore) return;

    const target = event.currentTarget;
    const nearBottom = target.scrollTop + target.clientHeight >= target.scrollHeight - 48;
    if (nearBottom) {
      fetchReportPicker({ query: reportPicker.query, anchorId: reportPicker.anchorId });
    }
  };

  return (
    <div className="mt-6 space-y-6 px-6 pb-8">
      {/* Header Section */}
      <section className="space-y-5 rounded-xl border bg-background px-5 py-5 shadow-sm">
        <div>
          <div className="flex items-center gap-4 mb-5">
            <Image
              width={56}
              height={56}
              src="/images/logo1.png"
              alt="Company Logo"
              className="h-11 w-auto"
            />

            <div>
              <h1 className="text-2xl font-bold tracking-tight text-gray-900">Master Table</h1>
              <p className="text-sm text-gray-500 mt-0.5">
                Kelola data induk pelaut, performa, dan rencana pengembangan individu.
              </p>
            </div>

            <Image
              width={56}
              height={56}
              src="/images/logo2.png"
              alt="Partner Logo"
              className="h-11 w-auto ml-auto"
            />
          </div>

          <Separator />
        </div>

        {/* Toolbar */}
        <div className="flex flex-wrap items-center gap-3 rounded-xl border bg-slate-50/60 p-3">
          <Input
            placeholder="🔍 Search by Name or Seafarer Code..."
            value={searchName}
            onChange={(e) => {
              setSearchName(e.target.value);
              setCurrentPage(1);
              setSelectAllGlobal(false);
            }}
            className="w-[280px] bg-white shadow-sm"
          />

          <div className="flex items-center gap-2 ml-auto">
            <Button
              size="sm"
              variant="default"
              onClick={refreshAllReadiness}
              disabled={onCallApi}
              className="flex items-center gap-2"
            >
              {onCallApi ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" /> Refreshing...
                </>
              ) : (
                <>
                  <RefreshCw className="w-4 h-4" /> Refresh Readiness
                </>
              )}
            </Button>

            <Button
              size="sm"
              variant="outline"
              onClick={refreshPersonalData}
              disabled={onCallApi}
              className="flex items-center gap-2"
            >
              {onCallApi ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" /> Refreshing...
                </>
              ) : (
                <>
                  <RefreshCw className="w-4 h-4" /> Refresh Personal Data
                </>
              )}
            </Button>

            <div className="flex items-center gap-2 border-l pl-4 ml-2">
              <Select
                value={
                  paginationRequest.batchId === null
                    ? "all"
                    : paginationRequest.batchId === -1
                      ? "no-batch"
                      : paginationRequest.batchId?.toString() || "all"
                }
                onValueChange={(val) => {
                  let batchId: number | null = null;
                  if (val === "all") {
                    batchId = null;
                  } else if (val === "no-batch") {
                    batchId = -1;
                  } else {
                    batchId = parseInt(val);
                  }
                  setSelectAllGlobal(false);
                  setSelectedIds(new Set());
                  setIsEditMode(false);
                  setCurrentPage(1);
                  setPaginationRequest((prev) => ({
                    ...prev,
                    batchId,
                    anchorId: 0,
                    page: "next",
                  }));
                }}
                disabled={loadingBatches}
              >
                <SelectTrigger className="w-[200px]">
                  <SelectValue placeholder="Pilih Batch" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Semua Batch</SelectItem>
                  <SelectItem value="no-batch">Tanpa Batch</SelectItem>
                  {batches.map((batch: Batch) => (
                    <SelectItem key={batch.id} value={batch.id.toString()}>
                      {batch.batchName}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              {/* Show a notice when selected batch is completed (snapshot data) */}
              {(() => {
                const selectedBatch = batches.find(
                  (b: Batch) => b.id === paginationRequest.batchId
                );
                if (selectedBatch?.status === "completed") {
                  return (
                    <span className="text-xs bg-amber-100 text-amber-700 border border-amber-300 rounded px-2 py-1">
                      📦 Data arsip batch {selectedBatch.batchName}
                    </span>
                  );
                }
                return null;
              })()}
            </div>

            <div className="w-px h-6 bg-gray-300 mx-1" />
            {!isArchived && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  setSelectedBatchForAssign("");
                  setSelectedReportIdsForBatch([]);
                  setReportPicker(EMPTY_REPORT_PICKER);
                  setBulkAssignBatchOpen(true);
                }}
                className="flex items-center gap-2 border-blue-300 text-blue-700 hover:bg-blue-50"
              >
                Assign ke Batch
              </Button>
            )}

            {!isArchived && (
              <Button
                size="sm"
                variant={isEditMode ? "destructive" : "outline"}
                onClick={toggleEditMode}
                className="flex items-center gap-2"
              >
                {isEditMode ? (
                  <>
                    <XIcon className="w-4 h-4" /> Batal Edit
                  </>
                ) : (
                  <>
                    <EditIcon className="w-4 h-4" /> Edit Mode
                  </>
                )}
              </Button>
            )}

            {isEditMode && (
              <>
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={confirmDelete}
                  disabled={selectedIds.size === 0}
                  className="flex items-center gap-2"
                >
                  <TrashIcon className="w-4 h-4" /> Delete
                  {selectedIds.size > 0 && (
                    <span className="ml-1 bg-red-700 text-white text-xs font-semibold px-1.5 py-0.5 rounded-full">
                      {selectedIds.size}
                    </span>
                  )}
                </Button>
              </>
            )}

            <div className="w-px h-6 bg-gray-300 mx-1" />
            {!isArchived && (
              <Dialog open={openDialog} onOpenChange={setOpenDialog}>
                <DialogTrigger asChild>
                  <Button size="sm" className="flex items-center gap-2">
                    <PlusIcon className="w-4 h-4" /> Add Report
                  </Button>
                </DialogTrigger>

                <DialogContent className="max-w-2xl">
                  <DialogHeader>
                    <DialogTitle>Add New Report</DialogTitle>
                    <DialogDescription>
                      Cari data dari sistem seaman dan pilih untuk ditambahkan.
                    </DialogDescription>
                  </DialogHeader>

                  <div className="space-y-4 py-2">
                    {/* Search Input */}
                    <div>
                      <Label>Search Seaman *</Label>
                      <Input
                        placeholder="Cari nama atau seaman code..."
                        value={addForm.search}
                        onChange={(e) => setAddForm({ ...addForm, search: e.target.value })}
                      />
                      <p className="text-xs text-gray-500 mt-1">
                        Ketik minimal 2 karakter untuk mencari
                      </p>
                    </div>

                    {/* Bulk Mode Toggle */}
                    {seamanResults.length > 0 && (
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          id="bulkMode"
                          checked={addForm.bulkAddMode || false}
                          onChange={toggleBulkMode}
                          className="w-4 h-4 rounded border-gray-300"
                        />
                        <Label htmlFor="bulkMode" className="cursor-pointer text-sm font-medium">
                          Bulk Add Mode (Pilih multiple seaman)
                        </Label>
                      </div>
                    )}

                    {/* Search Results */}
                    <div className="max-h-80 overflow-auto border rounded-md bg-white">
                      {loading && (
                        <p className="text-center text-gray-500 py-8">
                          <span className="inline-block">Mencari data...</span>
                        </p>
                      )}

                      {!loading && seamanResults.length > 0 && (
                        <div className="divide-y">
                          {seamanResults.map((seaman) => {
                            const isSelected = addForm.bulkAddMode
                              ? (addForm.selectedBulk || []).some(
                                  (s) => s.seamanCode === seaman.seamanCode
                                )
                              : addForm.selected?.seamanCode === seaman.seamanCode;

                            const alreadyAdded = !!seaman.alreadyAdded;

                            return (
                              <div
                                key={seaman.seamanCode}
                                onClick={() => {
                                  if (alreadyAdded) return;
                                  if (addForm.bulkAddMode) {
                                    toggleBulkSelection(seaman);
                                  } else {
                                    setAddForm({
                                      ...addForm,
                                      selected: seaman,
                                    });
                                  }
                                }}
                                className={`px-4 py-3 transition-colors ${
                                  alreadyAdded
                                    ? "cursor-not-allowed bg-gray-50 opacity-60"
                                    : isSelected
                                      ? "cursor-pointer bg-blue-50 border-l-4 border-blue-500"
                                      : "cursor-pointer hover:bg-gray-50"
                                }`}
                              >
                                <div className="flex items-center gap-3">
                                  {addForm.bulkAddMode && (
                                    <input
                                      type="checkbox"
                                      checked={isSelected}
                                      disabled={alreadyAdded}
                                      onChange={() => toggleBulkSelection(seaman)}
                                      onClick={(e) => e.stopPropagation()}
                                      className="w-4 h-4 rounded border-gray-300"
                                    />
                                  )}
                                  <div className="flex-1">
                                    <div className="font-semibold text-gray-800">{seaman.name}</div>
                                    <div className="text-sm text-gray-500">
                                      Seaman Code: {seaman.seamanCode}
                                    </div>
                                  </div>
                                  {alreadyAdded && (
                                    <span className="rounded bg-gray-200 px-2 py-0.5 text-xs font-medium text-gray-700">
                                      Sudah ada di report
                                    </span>
                                  )}
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      )}

                      {!loading && seamanResults.length === 0 && addForm.search && (
                        <p className="text-center text-gray-500 py-8">
                          Tidak ada data seaman yang cocok dengan pencarian &quot;{addForm.search}
                          &quot;
                        </p>
                      )}

                      {!loading && seamanResults.length === 0 && !addForm.search && (
                        <p className="text-center text-gray-400 py-8">
                          Mulai ketik untuk mencari seaman
                        </p>
                      )}
                    </div>

                    {/* Selected Info */}
                    {addForm.bulkAddMode && (addForm.selectedBulk || []).length > 0 && (
                      <div className="bg-blue-50 border border-blue-200 rounded-md p-3">
                        <p className="text-sm font-medium text-blue-900">
                          {(addForm.selectedBulk || []).length} seaman dipilih
                        </p>
                        <div className="text-xs text-blue-700 mt-2 space-y-1">
                          {(addForm.selectedBulk || []).map((s) => (
                            <div key={s.seamanCode} className="flex justify-between items-center">
                              <span>{s.name}</span>
                              <button
                                type="button"
                                onClick={() => toggleBulkSelection(s)}
                                className="text-blue-600 hover:text-blue-800 font-medium"
                              >
                                Remove
                              </button>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>

                  <DialogFooter>
                    <Button variant="outline" onClick={() => setOpenDialog(false)}>
                      Cancel
                    </Button>

                    <Button
                      disabled={
                        addForm.bulkAddMode
                          ? (addForm.selectedBulk || []).length === 0 || onCallApi
                          : !addForm.selected || onCallApi
                      }
                      onClick={handleAdd}
                    >
                      {onCallApi
                        ? "Adding..."
                        : `Add ${
                            addForm.bulkAddMode ? `(${(addForm.selectedBulk || []).length})` : ""
                          }`}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            )}
          </div>
        </div>
      </section>

      {/* Global Select-All Banner */}
      {isEditMode && (
        <>
          {isAllCurrentPageSelected() &&
            !selectAllGlobal &&
            (paginationData?.total ?? 0) > selectedIds.size && (
              <div className="flex items-center gap-3 px-4 py-2.5 bg-blue-50 border border-blue-200 rounded-xl mb-3 text-sm">
                <span className="text-blue-800">
                  <strong>{selectedIds.size}</strong> baris di halaman ini dipilih.
                </span>
                <button
                  onClick={() => setSelectAllGlobal(true)}
                  className="text-blue-600 font-semibold underline hover:text-blue-800 transition-colors"
                >
                  Pilih semua {paginationData?.total} baris?
                </button>
              </div>
            )}
          {selectAllGlobal && (
            <div className="flex items-center gap-3 px-4 py-2.5 bg-blue-100 border border-blue-400 rounded-xl mb-3 text-sm">
              <span className="text-blue-900 font-medium">
                ✓ Semua <strong>{paginationData?.total}</strong> baris dipilih.
              </span>
              <button
                onClick={() => {
                  setSelectAllGlobal(false);
                  setSelectedIds(new Set());
                }}
                className="text-blue-600 font-semibold underline hover:text-blue-800 transition-colors"
              >
                Batalkan pilihan
              </button>
            </div>
          )}
        </>
      )}

      {/* Bulk Assign to Batch Dialog */}
      <Dialog open={bulkAssignBatchOpen} onOpenChange={setBulkAssignBatchOpen}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>Assign ke Batch</DialogTitle>
            <DialogDescription>
              Pilih report crew dan batch tujuan. Pencarian memakai query backend dan daftar akan
              memuat bertahap saat discroll.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <Label>Pilih Report</Label>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={toggleSelectAllReportsForBatch}
                >
                  Pilih Semua
                </Button>
              </div>
              <Input
                value={reportPicker.query}
                onChange={(e) =>
                  setReportPicker((prev) => ({ ...prev, query: e.target.value, anchorId: 0 }))
                }
                placeholder="Cari nama atau seafarer code"
              />
              <div
                className="max-h-64 space-y-2 overflow-auto rounded-lg border p-3"
                onScroll={handleReportPickerScroll}
              >
                {reportPicker.items.map((report) => (
                  <label
                    key={report.id}
                    className="flex cursor-pointer items-start gap-3 rounded-md border border-transparent px-2 py-2 hover:bg-slate-50"
                  >
                    <Checkbox
                      checked={selectedReportIdsForBatch.includes(report.id)}
                      onCheckedChange={() => toggleReportSelectionForBatch(report.id)}
                    />
                    <div className="space-y-0.5 text-sm">
                      <div className="font-medium text-slate-900">{report.nama}</div>
                      <div className="text-slate-500">
                        {report.seafarerCode || report.seamanCode || "-"} •{" "}
                        {batches.find((batch: Batch) => batch.id === report.batchId)?.batchName ||
                          "Tanpa batch"}
                      </div>
                    </div>
                  </label>
                ))}
                {reportPicker.isLoading && (
                  <div className="py-3 text-center text-sm text-muted-foreground">
                    Memuat report...
                  </div>
                )}
                {reportPicker.isLoadingMore && (
                  <div className="py-3 text-center text-sm text-muted-foreground">
                    Memuat lebih banyak...
                  </div>
                )}
                {!reportPicker.isLoading && reportPicker.items.length === 0 && (
                  <div className="py-3 text-center text-sm text-muted-foreground">
                    Tidak ada report yang cocok
                  </div>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label>Batch Tujuan</Label>
              <Select value={selectedBatchForAssign} onValueChange={setSelectedBatchForAssign}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Pilih batch..." />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="null">Tanpa Batch</SelectItem>
                  {batches.map((batch: Batch) => (
                    <SelectItem key={batch.id} value={batch.id.toString()}>
                      {batch.batchName}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="text-sm text-muted-foreground">
              {selectedReportIdsForBatch.length} report dipilih
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setBulkAssignBatchOpen(false);
                setSelectedBatchForAssign("");
                setSelectedReportIdsForBatch([]);
                setReportPicker(EMPTY_REPORT_PICKER);
              }}
            >
              Batal
            </Button>
            <Button
              disabled={
                !selectedBatchForAssign || selectedReportIdsForBatch.length === 0 || onCallApi
              }
              onClick={async () => {
                try {
                  const batchId =
                    selectedBatchForAssign === "null" ? null : parseInt(selectedBatchForAssign);
                  await bulkAssignBatch(selectedReportIdsForBatch, batchId);
                  setBulkAssignBatchOpen(false);
                  setSelectedBatchForAssign("");
                  setSelectedReportIdsForBatch([]);
                  setReportPicker(EMPTY_REPORT_PICKER);
                } catch {
                  // error already shown in hook
                }
              }}
            >
              {onCallApi ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" /> Menyimpan...
                </>
              ) : (
                "Assign"
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={confirmDeleteDialog} onOpenChange={setConfirmDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-red-600">
              <AlertTriangle className="w-5 h-5" /> Confirm Deletion
            </DialogTitle>
            <DialogDescription>
              This action cannot be undone. The selected reports will be permanently deleted.
            </DialogDescription>
          </DialogHeader>
          <p className="text-gray-700">
            Are you sure you want to delete {selectedIds.size} selected{" "}
            {selectedIds.size > 1 ? "reports" : "report"}?
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmDeleteDialog(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDeleteConfirmed} disabled={onCallApi}>
              {onCallApi ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Deleting...
                </>
              ) : (
                "Yes, Delete"
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Mentoring Programs Details Dialog */}
      <Dialog open={mentoringDetailsDialogOpen} onOpenChange={setMentoringDetailsDialogOpen}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              Mentoring Programs for {selectedPersonForMentoring?.nama || editingRow?.nama}
            </DialogTitle>
            <DialogDescription>All mentoring programs linked to this person</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {linkedMentoringReports.length === 0 ? (
              <div className="p-4 bg-gray-50 rounded-md text-center">
                <p className="text-gray-600">No mentoring programs found</p>
              </div>
            ) : (
              linkedMentoringReports.map((report, index) => (
                <div key={report.id} className="p-4 border rounded-lg bg-white space-y-3">
                  <div className="flex items-center justify-between border-b pb-2">
                    <h3 className="text-lg font-semibold text-gray-800">
                      Program {index + 1}: {report.programTitle || "N/A"}
                    </h3>
                    <Badge variant="outline">{report.program || "N/A"}</Badge>
                  </div>

                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="font-medium text-gray-600">Mentor Name:</span>
                      <p className="text-gray-800">{report.mentorName || "-"}</p>
                    </div>
                    <div>
                      <span className="font-medium text-gray-600">Period:</span>
                      <p className="text-gray-800">{report.period || "-"}</p>
                    </div>
                    <div>
                      <span className="font-medium text-gray-600">Department:</span>
                      <p className="text-gray-800">{report.department || "-"}</p>
                    </div>
                    <div>
                      <span className="font-medium text-gray-600">Session Number:</span>
                      <p className="text-gray-800">{report.sessionNumber || "-"}</p>
                    </div>
                    <div>
                      <span className="font-medium text-gray-600">Date:</span>
                      <p className="text-gray-800">{report.date || "-"}</p>
                    </div>
                    <div>
                      <span className="font-medium text-gray-600">Duration:</span>
                      <p className="text-gray-800">{report.duration || "-"} minutes</p>
                    </div>
                  </div>

                  <Separator />

                  <div className="space-y-2">
                    <div>
                      <span className="font-medium text-gray-600">Purpose:</span>
                      <p className="text-gray-800 mt-1">{report.purpose || "-"}</p>
                    </div>
                    <div>
                      <span className="font-medium text-gray-600">Observation:</span>
                      <p className="text-gray-800 mt-1">{report.observation || "-"}</p>
                    </div>
                    <div>
                      <span className="font-medium text-gray-600">Reflection:</span>
                      <p className="text-gray-800 mt-1">{report.reflection || "-"}</p>
                    </div>
                    <div>
                      <span className="font-medium text-gray-600">Action Plan:</span>
                      <p className="text-gray-800 mt-1">{report.actionPlan || "-"}</p>
                    </div>
                    {report.additionalNotes && (
                      <div>
                        <span className="font-medium text-gray-600">Additional Notes:</span>
                        <p className="text-gray-800 mt-1">{report.additionalNotes}</p>
                      </div>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setMentoringDetailsDialogOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog with Competency Selection */}
      <Dialog open={openEditDialog} onOpenChange={setOpenEditDialog}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Report</DialogTitle>
            <DialogDescription>
              Update report information and manage competency gap analysis.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-2">
            <div>
              <Label htmlFor="edit-nama">Name *</Label>
              <Input
                id="edit-nama"
                placeholder="Name"
                value={editingRow?.nama || ""}
                onChange={(e) =>
                  editingRow && setEditingRow({ ...editingRow, nama: e.target.value.toUpperCase() })
                }
              />
            </div>

            <div>
              <Label htmlFor="edit-seamanCode">Seaman Code *</Label>
              <Input
                id="edit-seamanCode"
                placeholder="Seaman Code"
                value={editingRow?.seamanCode || ""}
                onChange={(e) =>
                  editingRow &&
                  setEditingRow({ ...editingRow, seamanCode: e.target.value.toUpperCase() })
                }
              />
            </div>

            <div>
              <Label htmlFor="edit-seafarerCode">Seafarer Code</Label>
              <Input
                id="edit-seafarerCode"
                placeholder="Seafarer Code (Optional)"
                value={editingRow?.seafarerCode || ""}
                onChange={(e) =>
                  editingRow &&
                  setEditingRow({ ...editingRow, seafarerCode: e.target.value.toUpperCase() })
                }
              />
            </div>

            {/* Additional Master Report Fields */}
            <div>
              <Label htmlFor="edit-vesselName">Vessel Name</Label>
              <Input
                id="edit-vesselName"
                placeholder="Vessel Name"
                value={editingRow?.vesselName || ""}
                onChange={(e) =>
                  editingRow &&
                  setEditingRow({ ...editingRow, vesselName: e.target.value.toUpperCase() })
                }
              />
            </div>

            <div>
              <Label htmlFor="edit-jabatan">Position</Label>
              <Input
                id="edit-jabatan"
                placeholder="Position"
                value={editingRow?.jabatan || ""}
                onChange={(e) =>
                  editingRow &&
                  setEditingRow({ ...editingRow, jabatan: e.target.value.toUpperCase() })
                }
              />
            </div>

            <div>
              <Label htmlFor="edit-age">Age</Label>
              <Input
                id="edit-age"
                placeholder="Age"
                value={editingRow?.age || ""}
                onChange={(e) =>
                  editingRow && setEditingRow({ ...editingRow, age: e.target.value })
                }
              />
            </div>

            <div>
              <Label htmlFor="edit-certificate">Certificate</Label>
              <Input
                id="edit-certificate"
                placeholder="Certificate"
                value={editingRow?.certificate || ""}
                onChange={(e) =>
                  editingRow &&
                  setEditingRow({ ...editingRow, certificate: e.target.value.toUpperCase() })
                }
              />
            </div>

            <div>
              <Label htmlFor="edit-idpProgram">IDP Program</Label>
              <Input
                id="edit-idpProgram"
                placeholder="IDP Program"
                value={editingRow?.idpProgram || ""}
                onChange={(e) =>
                  editingRow && setEditingRow({ ...editingRow, idpProgram: e.target.value })
                }
              />
            </div>

            <div>
              <Label htmlFor="edit-performanceScore">Performance Score</Label>
              <Input
                id="edit-performanceScore"
                placeholder="Performance Score"
                type="number"
                value={editingRow?.performanceScore || ""}
                onChange={(e) =>
                  editingRow &&
                  setEditingRow({ ...editingRow, performanceScore: parseInt(e.target.value) || 0 })
                }
              />
            </div>

            <div>
              <Label htmlFor="edit-readiness">Readiness</Label>
              <Input
                id="edit-readiness"
                placeholder="Readiness"
                type="number"
                value={editingRow?.totalReadinessUpdateMonths || ""}
                onChange={(e) =>
                  editingRow &&
                  setEditingRow({
                    ...editingRow,
                    totalReadinessUpdateMonths: parseInt(e.target.value) || 0,
                  })
                }
              />
            </div>

            <div>
              <Label htmlFor="edit-talentClassified">Talent Classified</Label>
              <Input
                id="edit-talentClassified"
                placeholder="Talent Classified"
                value={editingRow?.talentClassified || ""}
                onChange={(e) =>
                  editingRow && setEditingRow({ ...editingRow, talentClassified: e.target.value })
                }
              />
            </div>

            <div className="space-y-2">
              <Label>Mentoring Programs for {editingRow?.nama}</Label>
              {loadingLinkedMentoring ? (
                <div className="p-3 bg-blue-50 rounded-md flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <p className="text-sm text-blue-700">Loading mentoring programs...</p>
                </div>
              ) : linkedMentoringReports.length === 0 ? (
                <div className="p-3 bg-gray-50 rounded-md">
                  <p className="text-sm text-gray-600">
                    No mentoring programs found for <strong>{editingRow?.nama}</strong>
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  <div className="p-3 bg-green-50 border border-green-200 rounded-md">
                    <p className="text-sm text-green-800 font-medium">
                      Found {linkedMentoringReports.length} mentoring program
                      {linkedMentoringReports.length > 1 ? "s" : ""}
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    className="w-full"
                    type="button"
                    onClick={() => setMentoringDetailsDialogOpen(true)}
                  >
                    View All Programs ({linkedMentoringReports.length})
                  </Button>
                </div>
              )}
            </div>

            <div className="space-y-2">
              <Label>Assessment Type Scores</Label>
              <div className="grid grid-cols-2 gap-3">
                {assessmentTypes.map((assessmentType) => {
                  const existingScore =
                    editingRow?.reportScores?.find(
                      (rs) =>
                        rs.assessmentType?.assessmentTypeName === assessmentType.assessmentTypeName
                    )?.score ?? "";

                  return (
                    <div key={assessmentType.id}>
                      <Label htmlFor={`score-${assessmentType.id}`} className="text-xs">
                        {assessmentType.assessmentTypeName}
                      </Label>
                      <Input
                        id={`score-${assessmentType.id}`}
                        type="number"
                        placeholder="Score"
                        value={existingScore}
                        onChange={(e) => {
                          if (!editingRow) return;
                          const newScore =
                            e.target.value === "" ? 0 : parseInt(e.target.value) || 0;

                          // Get existing scores or initialize empty array
                          const existingScores = editingRow.reportScores || [];

                          // Check if this assessment type already exists
                          const scoreExists = existingScores.some(
                            (rs) =>
                              rs.assessmentType?.assessmentTypeName ===
                              assessmentType.assessmentTypeName
                          );

                          let updatedScores;
                          if (scoreExists) {
                            // Update existing score
                            updatedScores = existingScores.map((rs) =>
                              rs.assessmentType?.assessmentTypeName ===
                              assessmentType.assessmentTypeName
                                ? { ...rs, score: newScore }
                                : rs
                            );
                          } else {
                            // Add new score entry
                            updatedScores = [
                              ...existingScores,
                              {
                                score: newScore,
                                assessmentType: {
                                  assessmentTypeName: assessmentType.assessmentTypeName,
                                },
                              },
                            ];
                          }

                          setEditingRow({ ...editingRow, reportScores: updatedScores });
                        }}
                      />
                    </div>
                  );
                })}
              </div>
            </div>

            <div className="space-y-2">
              <Label>Competency Gap Analysis</Label>

              {/* Selected Competencies */}
              <div className="flex flex-wrap gap-2 min-h-[60px] p-3 border rounded-md bg-gray-50">
                {selectedCompetencies.length === 0 ? (
                  <span className="text-sm text-gray-400">No competencies selected</span>
                ) : (
                  selectedCompetencies.map((typeId) => {
                    const comp = competencyTypes.find((ct) => ct.id === typeId);
                    if (!comp) return null;
                    return (
                      <Badge
                        key={typeId}
                        style={{ backgroundColor: colorFromString(comp.code) }}
                        className="text-white flex items-center gap-1"
                      >
                        <span className="font-semibold">{comp.code}</span>
                        <span className="text-xs opacity-90">- {comp.name}</span>
                        <button
                          onClick={() => removeCompetency(typeId)}
                          className="ml-1 hover:text-red-200 transition-colors"
                          type="button"
                        >
                          <XIcon className="w-3 h-3" />
                        </button>
                      </Badge>
                    );
                  })
                )}
              </div>

              {/* Competency Selector */}
              <Popover open={competencySearchOpen} onOpenChange={setCompetencySearchOpen}>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    role="combobox"
                    className="w-full justify-between"
                    type="button"
                    disabled={loadingCompetencies}
                  >
                    {loadingCompetencies ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Loading competencies...
                      </>
                    ) : (
                      <>
                        Add Competency Type
                        <ChevronsUpDownIcon className="ml-2 h-4 w-4 opacity-50" />
                      </>
                    )}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-[500px] p-0" align="start">
                  <Command>
                    <CommandInput placeholder="Search competency types..." />
                    <CommandEmpty>No competency type found.</CommandEmpty>
                    <CommandList className="max-h-[300px]">
                      <CommandGroup>
                        {competencyTypes.map((type) => (
                          <CommandItem
                            key={type.id}
                            value={`${type.code} ${type.name}`}
                            onSelect={() => {
                              toggleCompetencySelection(type.id);
                            }}
                          >
                            <CheckIcon
                              className={cn(
                                "mr-2 h-4 w-4",
                                selectedCompetencies.includes(type.id) ? "opacity-100" : "opacity-0"
                              )}
                            />
                            <div className="flex flex-col">
                              <span className="font-medium">{type.code}</span>
                              <span className="text-xs text-gray-500">{type.name}</span>
                            </div>
                          </CommandItem>
                        ))}
                      </CommandGroup>
                    </CommandList>
                  </Command>
                </PopoverContent>
              </Popover>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setOpenEditDialog(false);
                setEditingRow(null);
                setSelectedCompetencies([]);
              }}
            >
              Cancel
            </Button>
            <Button onClick={handleEdit} disabled={!isEditFormValid()}>
              Update
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Table */}
      <section className="overflow-hidden rounded-xl border bg-background shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b bg-slate-50/60 px-5 py-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">Master Data Crew</h2>
            <p className="text-sm text-muted-foreground">
              Data induk crew, performa, assessment, dan readiness development.
            </p>
          </div>
          <Badge variant="outline" className="bg-white">
            {paginationData?.total ?? paginationData?.results?.length ?? 0} data
          </Badge>
        </div>
        <div
          ref={tableRef}
          className={`overflow-auto max-h-[70vh] transition-opacity ${
            onCallApi ? "opacity-60" : "opacity-100"
          }`}
        >
          <table className="w-full min-w-[2000px] border-collapse caption-bottom text-sm">
            <TableHeader className="sticky top-0 z-50">
              <TableRow className="bg-slate-900 hover:bg-slate-900">
                {isEditMode && (
                  <TableHead className="text-center sticky top-0 left-0 z-50 bg-slate-900 w-[50px] text-white">
                    <button
                      onClick={toggleSelectAll}
                      className="aspect-square h-4 w-4 rounded border border-white/60 inline-flex items-center justify-center hover:border-white transition-colors"
                    >
                      {isAllCurrentPageSelected() && (
                        <div className="h-2.5 w-2.5 rounded bg-white" />
                      )}
                    </button>
                  </TableHead>
                )}
                {TABLE_COLUMNS.map((col, i) => (
                  <TableHead
                    key={col}
                    className={cn(
                      "text-center sticky top-0 text-white font-semibold text-xs uppercase tracking-wide whitespace-nowrap px-3 py-3 border-r border-white/10",
                      i < 2 ? "z-50 bg-slate-900" : "z-40 bg-slate-900"
                    )}
                    style={i < 2 ? { left: `${offsets[i] || 0}px` } : {}}
                  >
                    {col}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>

            <TableBody>
              {onCallApi ? (
                <TableRow>
                  <TableCell
                    colSpan={TABLE_COLUMNS.length + (isEditMode ? 1 : 0)}
                    className="text-center text-gray-400 h-32"
                  >
                    <Loader2 className="mx-auto h-6 w-6 animate-spin" />
                    <p className="mt-2">Loading...</p>
                  </TableCell>
                </TableRow>
              ) : paginationData?.results?.length ? (
                paginationData.results.map((row, i) => (
                  <TableRow
                    key={row.id}
                    className={`transition-colors ${
                      selectedIds.has(row.id)
                        ? "bg-blue-50 border-l-2 border-l-blue-500"
                        : i % 2 === 0
                          ? "bg-white"
                          : "bg-gray-50/60"
                    } ${isEditMode ? "cursor-pointer hover:bg-blue-50/40" : "hover:bg-gray-100/50"}`}
                    onClick={() => handleRowClick(row)}
                  >
                    {isEditMode && (
                      <TableCell
                        className="text-center sticky left-0 z-40 bg-inherit border-r"
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleRowSelection(row.id);
                        }}
                      >
                        <button className="aspect-square h-4 w-4 rounded border border-primary inline-flex items-center justify-center hover:bg-primary/10 transition-colors">
                          {selectedIds.has(row.id) && (
                            <div className="h-2.5 w-2.5 rounded bg-primary" />
                          )}
                        </button>
                      </TableCell>
                    )}
                    <TableCell
                      className="text-center bg-inherit border-r sticky z-30 text-gray-500 text-xs font-medium"
                      style={{ left: `${offsets[0] || 0}px`, width: "60px", pointerEvents: "none" }}
                    >
                      <span className="pointer-events-auto">{getRowNumber(i)}</span>
                    </TableCell>
                    <TableCell
                      className="bg-inherit border-r sticky z-30 font-medium text-gray-900"
                      style={{
                        left: `${offsets[1] || 0}px`,
                        width: "200px",
                        pointerEvents: "none",
                      }}
                    >
                      <span className="pointer-events-auto">{row.nama}</span>
                    </TableCell>
                    <TableCell className="text-center text-xs font-mono text-gray-600">
                      {row.seamanCode || "-"}
                    </TableCell>
                    <TableCell className="text-center text-xs font-mono text-gray-600">
                      {row.seafarerCode || "-"}
                    </TableCell>
                    <TableCell className="text-center text-xs text-gray-700 whitespace-nowrap">
                      {row.vesselName || "-"}
                    </TableCell>
                    <TableCell className="text-center text-xs text-gray-700 whitespace-nowrap">
                      {row.jabatan || "-"}
                    </TableCell>
                    <TableCell className="text-center text-xs">{row.idpProgram || "-"}</TableCell>
                    <TableCell className="text-center text-xs">
                      <TrainingStatusBadge
                        status={row.trainingStatus}
                        exitedProgram={row.exitedProgram}
                      />
                    </TableCell>
                    <TableCell className="text-center">
                      {row.age ? (
                        <span className="inline-block bg-gray-100 text-gray-700 text-xs font-medium px-2 py-0.5 rounded-full">
                          {row.age}
                        </span>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell className="text-center text-xs">{row.certificate || "-"}</TableCell>
                    <TableCell className="text-center">
                      {row.konditeReview ? (
                        <span className="inline-block bg-purple-50 text-purple-700 text-xs font-medium px-2 py-0.5 rounded-full border border-purple-100">
                          {row.konditeReview}
                        </span>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      {row.kpiVessel ? (
                        <span className="inline-block bg-orange-50 text-orange-700 text-xs font-medium px-2 py-0.5 rounded-full border border-orange-100">
                          {row.kpiVessel}
                        </span>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      {row.performanceScore ? (
                        <span
                          className={`inline-block text-xs font-bold px-2 py-0.5 rounded-full ${
                            Number(row.performanceScore) >= 80
                              ? "bg-green-100 text-green-700"
                              : Number(row.performanceScore) >= 60
                                ? "bg-yellow-100 text-yellow-700"
                                : "bg-red-100 text-red-700"
                          }`}
                        >
                          {row.performanceScore}
                        </span>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      {Array.isArray(row.competencies) && row.competencies.length > 0 ? (
                        <div className="flex flex-wrap gap-1 justify-center">
                          {row.competencies.map((c, index) => {
                            const code = c?.competencyType?.code;
                            if (!code) return null;

                            return (
                              <span
                                key={c.id || c.competencyTypeId || `${code}-${index}`}
                                className="px-2 py-1 rounded-xl text-xs font-semibold text-white"
                                style={{ backgroundColor: colorFromString(code) }}
                              >
                                {code}
                              </span>
                            );
                          })}
                        </div>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      {row.totalGap != null ? (
                        <span
                          className={`inline-block text-xs font-semibold px-2 py-0.5 rounded-full ${
                            Number(row.totalGap) === 0
                              ? "bg-green-100 text-green-700"
                              : Number(row.totalGap) <= 3
                                ? "bg-yellow-100 text-yellow-700"
                                : "bg-red-100 text-red-700"
                          }`}
                        >
                          {row.totalGap}
                        </span>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      {row.strength ? (
                        <span className="inline-block bg-emerald-50 text-emerald-700 text-xs font-medium px-2 py-0.5 rounded-full border border-emerald-100">
                          {row.strength}
                        </span>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      {row.havQuadran2 ? (
                        <span className="inline-block bg-sky-50 text-sky-700 text-xs font-medium px-2 py-0.5 rounded-full border border-sky-100">
                          {row.havQuadran2}
                        </span>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      {row.talentClassified ? (
                        <span
                          className={`inline-block text-xs font-semibold px-2.5 py-1 rounded-full ${
                            row.talentClassified.toLowerCase().includes("high")
                              ? "bg-green-100 text-green-800"
                              : row.talentClassified.toLowerCase().includes("medium")
                                ? "bg-yellow-100 text-yellow-800"
                                : row.talentClassified.toLowerCase().includes("low")
                                  ? "bg-red-100 text-red-800"
                                  : "bg-gray-100 text-gray-700"
                          }`}
                        >
                          {row.talentClassified}
                        </span>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      {row.totalReadinessUpdateMonths != null ? (
                        <span
                          className={`inline-block text-xs font-semibold px-2 py-0.5 rounded-full ${
                            Number(row.totalReadinessUpdateMonths) >= 12
                              ? "bg-green-100 text-green-700"
                              : Number(row.totalReadinessUpdateMonths) >= 6
                                ? "bg-yellow-100 text-yellow-700"
                                : "bg-red-100 text-red-700"
                          }`}
                        >
                          {row.totalReadinessUpdateMonths} mo
                        </span>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      {row.certificateEligible ? (
                        <span className="inline-block bg-blue-50 text-blue-700 text-xs font-medium px-2 py-0.5 rounded-full border border-blue-100">
                          {row.certificateEligible}
                        </span>
                      ) : (
                        "-"
                      )}
                    </TableCell>

                    {/* Actions Column */}
                    <TableCell className="text-center" onClick={(e) => e.stopPropagation()}>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={(e) => handleViewMentoringPrograms(row, e)}
                        className="text-xs h-7 px-2.5 border-slate-300 hover:bg-slate-100 hover:border-slate-400"
                      >
                        📋 Programs
                      </Button>
                    </TableCell>

                    {/* Dynamic assessment type score columns */}
                    {assessmentTypeColumns.map((assessmentTypeName) => {
                      const score = getScoreForAssessmentType(row, assessmentTypeName);
                      return (
                        <TableCell key={assessmentTypeName} className="text-center">
                          <span
                            className={`inline-block text-xs font-bold px-2 py-0.5 rounded-full`}
                          >
                            {score > 0 ? score : "-"}
                          </span>
                        </TableCell>
                      );
                    })}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell
                    colSpan={TABLE_COLUMNS.length + (isEditMode ? 1 : 0)}
                    className="text-center text-gray-400 h-40"
                  >
                    <div className="flex flex-col items-center gap-2">
                      <span className="text-3xl">📭</span>
                      <p className="font-medium text-gray-500">Tidak ada data</p>
                      <p className="text-xs text-gray-400">
                        Coba ubah filter batch atau kata kunci pencarian
                      </p>
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between border-t px-5 py-4">
          {/* Page size selector */}
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-500">Tampilkan</span>
            <Popover>
              <PopoverTrigger asChild>
                <Button
                  variant="outline"
                  role="combobox"
                  className="w-20 h-8 text-sm justify-between"
                >
                  {paginationRequest.pageSize}
                  <ChevronsUpDownIcon className="ml-1 h-3 w-3 opacity-50" />
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-[100px] p-0">
                <Command>
                  <CommandList>
                    <CommandGroup>
                      {PAGE_SIZES.map((size) => (
                        <CommandItem
                          key={size}
                          value={size.toString()}
                          onSelect={() => {
                            setCurrentPage(1);
                            setPaginationRequest({
                              ...paginationRequest,
                              pageSize: size,
                              anchorId: 0,
                              page: "next",
                            });
                            setPageSize(size);
                          }}
                        >
                          <CheckIcon
                            className={cn(
                              "mr-2 h-4 w-4",
                              paginationRequest.pageSize === size ? "opacity-100" : "opacity-0"
                            )}
                          />
                          {size}
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  </CommandList>
                </Command>
              </PopoverContent>
            </Popover>
            <span className="text-xs text-gray-500">baris</span>
          </div>

          {/* Page navigation */}
          <Pagination className="mx-0 w-auto">
            <PaginationContent className="flex items-center gap-1">
              <PaginationItem>
                <PaginationPrevious
                  onClick={(e) => {
                    e.preventDefault();
                    if (
                      !(
                        !paginationData ||
                        paginationData.first_page ||
                        currentPage <= 1 ||
                        onCallApi
                      )
                    ) {
                      navigatePage("prev");
                    }
                  }}
                  className={cn(
                    "cursor-pointer",
                    (!paginationData ||
                      paginationData.first_page ||
                      currentPage <= 1 ||
                      onCallApi) &&
                      "pointer-events-none opacity-50"
                  )}
                />
              </PaginationItem>

              <PaginationItem>
                <PaginationLink
                  isActive
                  className="cursor-default hover:bg-background"
                  onClick={(e) => e.preventDefault()}
                >
                  {currentPage}
                </PaginationLink>
              </PaginationItem>

              <PaginationItem>
                <span className="text-sm text-gray-500 mx-1">
                  {paginationData?.total
                    ? `dari ${Math.ceil(paginationData.total / paginationRequest.pageSize)} halaman`
                    : ""}
                </span>
              </PaginationItem>

              <PaginationItem>
                <PaginationNext
                  onClick={(e) => {
                    e.preventDefault();
                    if (!(!paginationData?.has_more || onCallApi)) {
                      navigatePage("next");
                    }
                  }}
                  className={cn(
                    "cursor-pointer",
                    (!paginationData?.has_more || onCallApi) && "pointer-events-none opacity-50"
                  )}
                />
              </PaginationItem>
            </PaginationContent>
          </Pagination>

          {/* Row count info */}
          <div className="flex flex-col items-end gap-0.5">
            <span className="text-xs text-gray-500 bg-gray-100 px-3 py-1.5 rounded-lg">
              Menampilkan <strong>{paginationData?.results?.length || 0}</strong> baris di halaman
              ini
            </span>
            {paginationData?.total ? (
              <span className="text-xs text-gray-400 px-1">
                Total <strong>{paginationData.total}</strong> data
              </span>
            ) : null}
          </div>
        </div>
      </section>
    </div>
  );
}
