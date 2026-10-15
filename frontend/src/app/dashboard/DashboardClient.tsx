"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  HistoryIcon,
  RefreshCwIcon,
  SearchIcon,
  UsersIcon,
  XIcon,
} from "lucide-react";
import { IReport, IPaginationRequest, PageType, FilterType } from "@/types/global-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { useGetIdpCount, useGetReports } from "./_hooks/useReports";
import { useCitrixSync } from "./_hooks/useCitrixSync";
import { useProgramExits } from "./_hooks/useProgramExits";
import ProgramExitHistoryDialog from "./ProgramExitHistoryDialog";
import { TrainingStatusBadge } from "@/components/training-status-badge";
import { useBatches } from "../master-report/_hooks/useBatch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Batch } from "../master-report/_hooks/useBatch";
import { useDebounce } from "use-debounce";
import { cn } from "@/lib/utils";

const PAGE_SIZES = [10, 20, 50, 100];

// Program tiles double as the program filter; "" means everyone currently in a program.
const PROGRAMS: { value: FilterType; label: string; hint: string; accent: string }[] = [
  { value: "", label: "Semua", hint: "Seluruh peserta program", accent: "bg-slate-500" },
  { value: "FDP", label: "FDP", hint: "Mualim III · Masinis IV", accent: "bg-sky-500" },
  { value: "MDP", label: "MDP", hint: "Mualim II · Masinis III", accent: "bg-violet-500" },
  { value: "SDP", label: "SDP", hint: "Mualim I · Masinis II", accent: "bg-emerald-500" },
];

const PROGRAM_BADGE: Record<string, string> = {
  FDP: "bg-sky-100 text-sky-800",
  MDP: "bg-violet-100 text-violet-800",
  SDP: "bg-emerald-100 text-emerald-800",
};

export default function DashboardClient() {
  const router = useRouter();
  const citrixSync = useCitrixSync();

  const handleCitrixSync = () => {
    toast.promise(citrixSync.mutateAsync(), {
      loading: "Mengambil data terbaru dari CITRIX...",
      success: "Data CITRIX berhasil disinkronkan",
      error: (err: Error) => err.message,
    });
  };

  // Search state
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedQuery] = useDebounce(searchQuery, 500);

  // Pagination request state
  const [paginationRequest, setPaginationRequest] = useState<IPaginationRequest>({
    anchorId: 0,
    page: "next",
    pageSize: 10,
    filter: "",
    batchId: undefined,
  });

  // Any filter change starts again from the first page
  const applyFilter = (patch: Partial<IPaginationRequest>) => {
    setPaginationRequest((prev) => ({ ...prev, ...patch, anchorId: 0, page: "next" }));
  };

  // Reset pagination when search query changes
  useEffect(() => {
    setPaginationRequest((prev) => ({
      ...prev,
      anchorId: 0,
      page: "next",
      query: debouncedQuery || undefined,
    }));
  }, [debouncedQuery]);

  // React Query hooks for fetching data
  const { data: idpCountData, error: idpCountError } = useGetIdpCount(
    paginationRequest.batchId,
    debouncedQuery
  );
  const { batches: batchData } = useBatches();

  // People promoted out of their program live in their own history, not in the list below.
  const programExits = useProgramExits();
  const [historyOpen, setHistoryOpen] = useState(false);

  const {
    data: paginationData,
    isLoading: reportsLoading,
    isFetching: reportsFetching,
    error: reportsError,
  } = useGetReports(paginationRequest);

  useEffect(() => {
    if (idpCountError) toast.error("Gagal memuat jumlah talent per program");
  }, [idpCountError]);

  useEffect(() => {
    if (reportsError) toast.error((reportsError as Error).message || "Gagal memuat data talent");
  }, [reportsError]);

  const programCount = (program: FilterType): number | null => {
    if (!idpCountData) return null;
    switch (program) {
      case "FDP":
        return idpCountData.fdp;
      case "MDP":
        return idpCountData.mdp;
      case "SDP":
        return idpCountData.sdp;
      default:
        return idpCountData.total ?? null;
    }
  };

  const activeBatch = batchData?.find((batch: Batch) => batch.id === paginationRequest.batchId);
  const batchLabel =
    paginationRequest.batchId === -1
      ? "Tanpa Batch"
      : activeBatch
        ? `Batch ${activeBatch.batchNo}`
        : null;
  const hasActiveFilter =
    paginationRequest.filter !== "" || paginationRequest.batchId != null || searchQuery !== "";

  const resetFilters = () => {
    setSearchQuery("");
    setPaginationRequest((prev) => ({
      ...prev,
      filter: "",
      batchId: undefined,
      query: undefined,
      anchorId: 0,
      page: "next",
    }));
  };

  // Pagination navigation
  const navigatePage = (page: PageType) => {
    if (!paginationData) return;
    setPaginationRequest({
      ...paginationRequest,
      page: page,
      anchorId: page == "next" ? paginationData?.last_id : paginationData?.first_id,
    });
  };

  // Navigate to talent profile detail page
  const handleRowClick = (report: IReport) => {
    router.push(`/dashboard/${report.seafarerCode}`);
  };

  const rows = paginationData?.results ?? [];
  const matchingTotal = programCount(paginationRequest.filter);

  return (
    <div className="my-6 space-y-5">
      {/* Header */}
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Talent Dashboard</h1>
          <p className="text-sm text-muted-foreground">
            Talent yang sedang menjalani program. Pilih program untuk menyaring, lalu klik baris
            untuk melihat profilnya.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button type="button" variant="outline" onClick={() => setHistoryOpen(true)}>
            <HistoryIcon />
            Riwayat keluar program
            {programExits.data && programExits.data.length > 0 && (
              <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-semibold text-amber-800">
                {programExits.data.length}
              </span>
            )}
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={handleCitrixSync}
            disabled={citrixSync.isPending}
          >
            <RefreshCwIcon className={citrixSync.isPending ? "animate-spin" : ""} />
            {citrixSync.isPending ? "Menyinkronkan..." : "Sinkronkan CITRIX"}
          </Button>
        </div>
      </div>

      <ProgramExitHistoryDialog
        open={historyOpen}
        onOpenChange={setHistoryOpen}
        exits={programExits.data}
        isLoading={programExits.isLoading}
        error={programExits.error}
      />

      {/* Program tiles = program filter */}
      <div
        className="grid grid-cols-2 gap-3 lg:grid-cols-4"
        role="group"
        aria-label="Filter program"
      >
        {PROGRAMS.map((program) => {
          const active = paginationRequest.filter === program.value;
          const count = programCount(program.value);
          return (
            <button
              key={program.label}
              type="button"
              aria-pressed={active}
              // Clicking the active program again goes back to "Semua"
              onClick={() => applyFilter({ filter: active ? "" : program.value })}
              className={cn(
                "group relative overflow-hidden rounded-xl border bg-card p-4 text-left shadow-sm transition",
                "hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                active ? "border-primary ring-2 ring-primary/20" : "hover:border-foreground/30"
              )}
            >
              <span className={cn("absolute inset-y-0 left-0 w-1.5", program.accent)} />
              <div className="flex items-start justify-between gap-2 pl-2">
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-muted-foreground">{program.label}</p>
                  {count === null ? (
                    <Skeleton className="mt-1 h-9 w-14" />
                  ) : (
                    <p className="text-3xl font-bold tabular-nums leading-tight">{count}</p>
                  )}
                  <p className="mt-1 truncate text-xs text-muted-foreground">{program.hint}</p>
                </div>
                {program.value === "" ? (
                  <UsersIcon className="h-5 w-5 shrink-0 text-muted-foreground" />
                ) : (
                  <span
                    className={cn(
                      "shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium",
                      active
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted text-muted-foreground"
                    )}
                  >
                    {active ? "Aktif" : "Filter"}
                  </span>
                )}
              </div>
            </button>
          );
        })}
      </div>

      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative min-w-[220px] flex-1">
          <SearchIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Cari nama atau seafarer code..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="bg-white pl-9 pr-9 shadow-sm"
          />
          {searchQuery && (
            <button
              type="button"
              aria-label="Hapus pencarian"
              onClick={() => setSearchQuery("")}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              <XIcon className="h-4 w-4" />
            </button>
          )}
        </div>

        <Select
          value={paginationRequest.batchId?.toString() || "all"}
          onValueChange={(val) => {
            const bId = val === "all" ? undefined : val === "none" ? -1 : parseInt(val);
            applyFilter({ batchId: bId });
          }}
        >
          <SelectTrigger className="w-[170px] bg-white" suppressHydrationWarning>
            <SelectValue placeholder="Pilih Batch" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Batch</SelectItem>
            <SelectItem value="none">Tanpa Batch</SelectItem>
            {batchData?.map((batch: Batch) => (
              <SelectItem key={batch.id} value={batch.id.toString()}>
                Batch {batch.batchNo}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={paginationRequest.pageSize.toString()}
          onValueChange={(val) => applyFilter({ pageSize: parseInt(val) })}
        >
          <SelectTrigger className="w-[130px] bg-white" suppressHydrationWarning>
            <SelectValue placeholder="Jumlah baris" />
          </SelectTrigger>
          <SelectContent>
            {PAGE_SIZES.map((size) => (
              <SelectItem key={size} value={size.toString()}>
                {size} baris
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Active filters */}
      {hasActiveFilter && (
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <span className="text-muted-foreground">Filter aktif:</span>
          {paginationRequest.filter !== "" && (
            <FilterChip
              label={`Program ${paginationRequest.filter}`}
              onRemove={() => applyFilter({ filter: "" })}
            />
          )}
          {batchLabel && (
            <FilterChip label={batchLabel} onRemove={() => applyFilter({ batchId: undefined })} />
          )}
          {searchQuery !== "" && (
            <FilterChip label={`"${searchQuery}"`} onRemove={() => setSearchQuery("")} />
          )}
          <Button variant="ghost" size="sm" className="h-7 px-2" onClick={resetFilters}>
            Reset semua
          </Button>
        </div>
      )}

      {/* TABLE */}
      <div
        className={cn(
          "overflow-x-auto rounded-xl border bg-card shadow-sm transition-opacity",
          reportsFetching && !reportsLoading && "opacity-60"
        )}
      >
        <Table>
          <TableHeader>
            <TableRow className="bg-muted/50 hover:bg-muted/50">
              <TableHead className="pl-4">Seafarer Code</TableHead>
              <TableHead>Nama Talent</TableHead>
              <TableHead>Rank</TableHead>
              <TableHead className="text-center">Program</TableHead>
              <TableHead className="text-center">Development Status</TableHead>
              <TableHead className="pr-4 text-right">Talent Readiness</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {reportsLoading ? (
              Array.from({ length: 6 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 6 }).map((__, j) => (
                    <TableCell key={j} className="py-3">
                      <Skeleton className="h-4 w-full" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : reportsError ? (
              <TableRow>
                <TableCell colSpan={6} className="py-10 text-center text-red-600">
                  Gagal memuat data: {(reportsError as Error).message}
                </TableCell>
              </TableRow>
            ) : rows.length > 0 ? (
              rows.map((report) => (
                <TableRow
                  key={report.id}
                  tabIndex={0}
                  className="cursor-pointer hover:bg-muted/60 focus-visible:bg-muted/60 focus-visible:outline-none"
                  onClick={() => handleRowClick(report)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleRowClick(report);
                  }}
                >
                  <TableCell className="pl-4 font-mono text-sm font-semibold">
                    {report.seafarerCode}
                  </TableCell>
                  <TableCell className="font-medium">{report.nama}</TableCell>
                  <TableCell className="text-muted-foreground">{report.jabatan || "-"}</TableCell>
                  <TableCell className="text-center">
                    {report.idpProgram ? (
                      <span
                        className={cn(
                          "inline-block rounded-full px-2.5 py-0.5 text-xs font-semibold",
                          PROGRAM_BADGE[report.idpProgram] ?? "bg-muted text-muted-foreground"
                        )}
                      >
                        {report.idpProgram}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">-</span>
                    )}
                  </TableCell>
                  <TableCell className="text-center">
                    <TrainingStatusBadge
                      status={report.trainingStatus}
                      program={report.idpProgram}
                      exitedProgram={report.exitedProgram}
                    />
                  </TableCell>
                  <TableCell className="pr-4 text-right tabular-nums">
                    {report.totalReadinessUpdateMonths != null
                      ? `${report.totalReadinessUpdateMonths} bulan`
                      : "-"}
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={6} className="py-12 text-center">
                  <p className="font-medium">Tidak ada talent yang cocok</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    Coba ubah kata pencarian atau filter yang dipakai.
                  </p>
                  {hasActiveFilter && (
                    <Button variant="outline" size="sm" className="mt-3" onClick={resetFilters}>
                      Reset filter
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">
          {reportsLoading
            ? "Memuat..."
            : `Menampilkan ${rows.length} talent${matchingTotal != null ? ` dari ${matchingTotal}` : ""}`}
        </p>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            disabled={paginationData?.first_page ?? true}
            onClick={() => navigatePage("prev")}
          >
            <ChevronLeftIcon />
            Sebelumnya
          </Button>
          <Button
            variant="outline"
            disabled={!paginationData?.has_more}
            onClick={() => navigatePage("next")}
          >
            Berikutnya
            <ChevronRightIcon />
          </Button>
        </div>
      </div>
    </div>
  );
}

function FilterChip({ label, onRemove }: { label: string; onRemove: () => void }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full border bg-background py-0.5 pl-2.5 pr-1 text-xs font-medium">
      {label}
      <button
        type="button"
        aria-label={`Hapus filter ${label}`}
        onClick={onRemove}
        className="rounded-full p-0.5 text-muted-foreground hover:bg-muted hover:text-foreground"
      >
        <XIcon className="h-3 w-3" />
      </button>
    </span>
  );
}
