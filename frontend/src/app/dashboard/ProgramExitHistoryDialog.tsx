"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowRightIcon, SearchIcon } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { ProgramExit } from "./_hooks/useProgramExits";

const PROGRAM_BADGE: Record<string, string> = {
  FDP: "bg-sky-100 text-sky-800",
  MDP: "bg-violet-100 text-violet-800",
  SDP: "bg-emerald-100 text-emerald-800",
};

// How the person left the list of active participants, in words an admin can act on.
const REASONS: Record<ProgramExit["reason"], { label: string; hint: string; style: string }> = {
  completed: {
    label: "Lulus program",
    hint: "Semua training di program ini sudah diselesaikan.",
    style: "bg-green-100 text-green-800",
  },
  promoted: {
    label: "Naik jabatan sebelum selesai",
    hint: "Naik jabatan saat training program belum selesai, jadi otomatis keluar dari program.",
    style: "bg-amber-100 text-amber-800",
  },
};

function formatDate(value: string | null): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" });
}

interface ProgramExitHistoryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  exits: ProgramExit[] | undefined;
  isLoading: boolean;
  error: Error | null;
}

// History of people who left an IDP program because they were promoted.
export default function ProgramExitHistoryDialog({
  open,
  onOpenChange,
  exits,
  isLoading,
  error,
}: ProgramExitHistoryDialogProps) {
  const router = useRouter();
  const [query, setQuery] = useState("");

  const rows = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return exits ?? [];
    return (exits ?? []).filter(
      (exit) =>
        exit.nama.toLowerCase().includes(q) ||
        exit.seafarerCode.toLowerCase().includes(q) ||
        exit.exitedProgram.toLowerCase().includes(q)
    );
  }, [exits, query]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] w-[95vw] flex-col gap-4 sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle>Riwayat Keluar Program</DialogTitle>
          <DialogDescription>
            Talent yang sudah tidak menjalani program IDP-nya, jadi tidak tampil di daftar
            dashboard. <span className="font-medium text-green-700">Lulus program</span> berarti
            semua training sudah selesai.{" "}
            <span className="font-medium text-amber-700">Naik jabatan sebelum selesai</span> berarti
            keluar otomatis karena naik jabatan. Untuk memasukkan seseorang ke program berikutnya,
            ubah IDP Program-nya di Master Report.
          </DialogDescription>
        </DialogHeader>

        <div className="relative">
          <SearchIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Cari nama, seafarer code, atau program..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        <div className="min-h-0 flex-1 overflow-auto rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/50 hover:bg-muted/50">
                <TableHead className="pl-4">Talent</TableHead>
                <TableHead className="text-center">Program</TableHead>
                <TableHead>Keterangan</TableHead>
                <TableHead>Naik jabatan</TableHead>
                <TableHead>Tanggal</TableHead>
                <TableHead className="pr-4">Jabatan sekarang</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 4 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 6 }).map((__, j) => (
                      <TableCell key={j} className="py-3">
                        <Skeleton className="h-4 w-full" />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : error ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-10 text-center text-red-600">
                    Gagal memuat riwayat. Coba tutup lalu buka lagi.
                  </TableCell>
                </TableRow>
              ) : rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-10 text-center text-muted-foreground">
                    {query
                      ? "Tidak ada riwayat yang cocok dengan pencarian."
                      : "Belum ada talent yang keluar dari program."}
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((exit) => (
                  <TableRow
                    key={exit.reportId}
                    tabIndex={0}
                    title="Buka profil talent"
                    className="cursor-pointer hover:bg-muted/60 focus-visible:bg-muted/60 focus-visible:outline-none"
                    onClick={() => router.push(`/dashboard/${exit.seafarerCode}`)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") router.push(`/dashboard/${exit.seafarerCode}`);
                    }}
                  >
                    <TableCell className="pl-4">
                      <p className="font-medium">{exit.nama}</p>
                      <p className="font-mono text-xs text-muted-foreground">{exit.seafarerCode}</p>
                    </TableCell>
                    <TableCell className="text-center">
                      <span
                        className={cn(
                          "inline-block rounded-full px-2.5 py-0.5 text-xs font-semibold",
                          PROGRAM_BADGE[exit.exitedProgram] ?? "bg-muted text-muted-foreground"
                        )}
                      >
                        {exit.exitedProgram || "-"}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span
                        title={(REASONS[exit.reason] ?? REASONS.promoted).hint}
                        className={cn(
                          "inline-block whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-semibold",
                          (REASONS[exit.reason] ?? REASONS.promoted).style
                        )}
                      >
                        {(REASONS[exit.reason] ?? REASONS.promoted).label}
                      </span>
                    </TableCell>
                    <TableCell>
                      {!exit.promoted ? (
                        <span className="text-muted-foreground">Belum naik jabatan</span>
                      ) : exit.fromPosition && exit.toPosition ? (
                        <span className="inline-flex flex-wrap items-center gap-1.5">
                          <span className="text-muted-foreground">{exit.fromPosition}</span>
                          <ArrowRightIcon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                          <span className="font-medium">{exit.toPosition}</span>
                        </span>
                      ) : (
                        <span
                          className="text-muted-foreground"
                          title="Riwayat mutasi CITRIX tidak memuat kenaikan jabatan ini"
                        >
                          Tidak tercatat di CITRIX
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="whitespace-nowrap">
                      {exit.promotedAt ? (
                        <span title="Tanggal naik jabatan di CITRIX">
                          {formatDate(exit.promotedAt)}
                        </span>
                      ) : exit.completedAt ? (
                        <span title="Tanggal program diselesaikan">
                          {formatDate(exit.completedAt)}
                        </span>
                      ) : (
                        <span title="Tanggal sistem mencatat keluar dari program">
                          {formatDate(exit.exitedAt)}
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="pr-4">
                      <p>{exit.currentPosition || "-"}</p>
                      {exit.vesselName && (
                        <p className="text-xs text-muted-foreground">{exit.vesselName}</p>
                      )}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {!isLoading && !error && (
          <p className="text-sm text-muted-foreground">
            {rows.length} dari {exits?.length ?? 0} talent
          </p>
        )}
      </DialogContent>
    </Dialog>
  );
}
