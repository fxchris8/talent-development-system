"use client";
import { useState, useMemo, useEffect } from "react";
import { useAssignments } from "./_hooks/useAssigments";
import { useCatalogs } from "./_hooks/useCatalogs";
import { useBatches } from "../master-report/_hooks/useBatch";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableCell,
  TableBody,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationPrevious,
  PaginationNext,
  PaginationLink,
} from "@/components/ui/pagination";
import { PlusIcon, EditIcon, TrashIcon, Search } from "lucide-react";
import { toast } from "sonner";
import { IAssignmentFlat } from "@/types/global-types";

export default function AssignmentTable({ batchId: externalBatchId }: { batchId?: string | null }) {
  const {
    loading,
    assignments,
    fetchAll,
    createAssignment,
    updateAssignment,
    deleteAssignment,
    searchQuery,
    setSearchQuery,
    pageSize,
    setPageSize,
    filterAssessment,
    setFilterAssessment,
    filterStatus,
    setFilterStatus,
    filterBatch,
    setFilterBatch,
    currentPage,
    navigatePage,
    filteredCount,
  } = useAssignments();

  const { assessments, users } = useCatalogs(filterBatch);
  const { batches } = useBatches();

  // Sync external batchId to internal filterBatch
  useEffect(() => {
    if (externalBatchId !== undefined) {
      setFilterBatch(externalBatchId ?? "ALL");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [externalBatchId]);

  const [openDialog, setOpenDialog] = useState(false);
  const [editing, setEditing] = useState<IAssignmentFlat | null>(null);
  const [form, setForm] = useState({
    seafarer_codes: [] as string[],
    assessment_type_id: "",
    status: "ASSIGNED",
    userSearch: "",
  });

  // Filter user di dialog berdasarkan search
  const filteredUsers = useMemo(() => {
    const q = form.userSearch.toLowerCase();
    return users.filter(
      (u) => u.nama?.toLowerCase().includes(q) || u.seafarerCode?.toLowerCase().includes(q)
    );
  }, [form.userSearch, users]);

  const handleEdit = (a: IAssignmentFlat) => {
    setEditing(a);
    setForm({
      seafarer_codes: [a.seafarerCode],
      assessment_type_id: a.assessmentTypeId.toString(),
      status: a.status || "ASSIGNED",
      userSearch: "",
    });
    setOpenDialog(true);
  };

  const handleSubmit = async () => {
    if (!form.assessment_type_id || form.seafarer_codes.length === 0) {
      toast.error("Pilih minimal satu seafarer dan assessment!");
      return;
    }

    try {
      if (editing) {
        await updateAssignment(editing.id || 0, {
          id: editing.id || 0, //  add this line
          assessmentTypeId: Number(form.assessment_type_id),
          status: form.status,
        });
        toast.success("Assignment berhasil diperbarui!");
      } else {
        // bulk add
        for (const code of form.seafarer_codes) {
          await createAssignment({
            seafarerCode: code,
            assessmentTypeId: Number(form.assessment_type_id),
            batchId:
              filterBatch && filterBatch !== "ALL" && filterBatch !== "-1"
                ? Number(filterBatch)
                : null,
            status: form.status,
            createdBy: "SYSTEM",
          });
        }
        toast.success(`${form.seafarer_codes.length} assignment berhasil dibuat!`);
      }

      await fetchAll();
      setEditing(null);
      setOpenDialog(false);
      setForm({
        seafarer_codes: [],
        assessment_type_id: "",
        status: "ASSIGNED",
        userSearch: "",
      });
    } catch (err) {
      console.error(err);
      toast.error("Gagal menyimpan assignment");
    }
  };

  const handleDelete = async (id: number) => {
    if (window.confirm("Yakin ingin menghapus assignment ini?")) {
      await deleteAssignment(id);
    }
  };

  return (
    <div className="mt-8 p-4 m-6 space-y-6 ">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-3">
          <h2 className="text-xl font-semibold">Assignments</h2>
          {externalBatchId !== undefined &&
            filterBatch &&
            filterBatch !== "ALL" &&
            (() => {
              const b = batches.find((bt) => bt.id.toString() === filterBatch);
              return b ? (
                <span className="px-2.5 py-0.5 rounded-full bg-blue-100 text-blue-800 text-xs font-medium">
                  Batch {b.batchNo}
                </span>
              ) : null;
            })()}
        </div>
      </div>

      {/* Search bar */}
      {/* Search + Filters */}
      <div className="flex flex-wrap justify-between items-center gap-3 mb-6">
        {/* Search */}
        <div className="flex gap-4 items-center">
          <div className="relative w-[350px] max-w-full">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
            <Input
              placeholder="Cari nama, seafarer code, assessment..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>
        </div>

        {/* FILTERS */}
        <div className="flex flex-wrap items-center gap-3">
          {/* Filter Assessment */}
          <Select value={filterAssessment} onValueChange={setFilterAssessment}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="All Assessments" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">All Assessments</SelectItem>
              {assessments.map((a) => (
                <SelectItem key={a.assessmentId} value={a.assessmentId?.toString() || ""}>
                  {a.assessmentName}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* Filter Status */}
          <Select value={filterStatus} onValueChange={setFilterStatus}>
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder="All Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">All Status</SelectItem>
              <SelectItem value="ASSIGNED">Assigned</SelectItem>
              <SelectItem value="IN_PROGRESS">In Progress</SelectItem>
              <SelectItem value="COMPLETED">Completed</SelectItem>
            </SelectContent>
          </Select>

          {/* Filter Batch – hidden when batch is controlled from parent */}
          {externalBatchId === undefined && (
            <Select value={filterBatch} onValueChange={setFilterBatch}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="Semua Batch" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ALL">Semua Batch</SelectItem>
                <SelectItem value="-1">Tanpa Batch</SelectItem>
                {batches.map((b) => (
                  <SelectItem key={b.id} value={b.id.toString()}>
                    Batch {b.batchNo}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}

          {/* Page Size selector */}
          <Select value={pageSize.toString()} onValueChange={(val) => setPageSize(Number(val))}>
            <SelectTrigger className="w-[120px]">
              <SelectValue placeholder="Page Size" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="10">10 rows</SelectItem>
              <SelectItem value="20">20 rows</SelectItem>
              <SelectItem value="50">50 rows</SelectItem>
              <SelectItem value="100">100 rows</SelectItem>
              <SelectItem value="-1">Show All</SelectItem>
            </SelectContent>
          </Select>

          {/* New Assignment */}
          <Button
            onClick={() => {
              setEditing(null);
              setForm({
                seafarer_codes: [],
                assessment_type_id: "",
                status: "ASSIGNED",
                userSearch: "",
              });
              setOpenDialog(true);
            }}
          >
            <PlusIcon className="w-4 h-4 mr-2" /> New Assignment
          </Button>
        </div>
      </div>

      {/* Pagination Dashboard */}
      <div className="flex items-center justify-between mt-4 text-xs text-muted-foreground bg-slate-50 p-3 rounded-lg border border-slate-100">
        <div className="flex items-center gap-2">
          <span>
            Menampilkan <strong>{assignments.length}</strong> dari <strong>{filteredCount}</strong>{" "}
            records
          </span>
          {pageSize !== -1 && <span className="text-slate-300">|</span>}
          {pageSize !== -1 && (
            <span>
              Halaman <strong>{currentPage}</strong> dari{" "}
              <strong>{Math.ceil(filteredCount / pageSize)}</strong>
            </span>
          )}
        </div>

        {pageSize !== -1 && (
          <Pagination className="mx-0 w-auto">
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious
                  onClick={(e) => {
                    e.preventDefault();
                    if (currentPage > 1) navigatePage("prev");
                  }}
                  className={cn(
                    "cursor-pointer",
                    currentPage <= 1 && "pointer-events-none opacity-50"
                  )}
                />
              </PaginationItem>
              <PaginationItem>
                <PaginationLink isActive className="cursor-default hover:bg-background">
                  {currentPage}
                </PaginationLink>
              </PaginationItem>
              <PaginationItem>
                <PaginationNext
                  onClick={(e) => {
                    e.preventDefault();
                    if (currentPage < Math.ceil(filteredCount / pageSize)) navigatePage("next");
                  }}
                  className={cn(
                    "cursor-pointer",
                    currentPage >= Math.ceil(filteredCount / pageSize) &&
                      "pointer-events-none opacity-50"
                  )}
                />
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        )}
      </div>

      {/* Table */}
      <div className="overflow-auto border rounded-lg ">
        <Table>
          <TableHeader className="sticky top-0 bg-background z-10">
            <TableRow>
              <TableHead className="text-center">No</TableHead>
              <TableHead className="text-center">Nama</TableHead>
              <TableHead className="text-center">Seafarer Code</TableHead>
              <TableHead className="text-center">Assessment</TableHead>
              <TableHead className="text-center">Status</TableHead>
              <TableHead className="text-center">Attempts</TableHead>
              <TableHead className="text-center">Action</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center">
                  Loading...
                </TableCell>
              </TableRow>
            ) : assignments.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center text-gray-400">
                  Tidak ada data assignment.
                </TableCell>
              </TableRow>
            ) : (
              assignments.map((a, no) => (
                <TableRow key={a.id}>
                  <TableCell className="text-center">{no + 1}</TableCell>
                  <TableCell className="text-center">{a.nama}</TableCell>
                  <TableCell className="text-center">{a.seafarerCode}</TableCell>
                  <TableCell className="text-center">{a.assessmentType}</TableCell>
                  <TableCell className="text-center">
                    <span
                      className={`px-2 py-1 rounded text-xs font-medium ${
                        a.status === "ASSIGNED"
                          ? "bg-blue-200 text-blue-800"
                          : a.status === "COMPLETED"
                            ? "bg-green-100 text-green-800"
                            : "bg-gray-100 text-gray-800"
                      }`}
                    >
                      {a.status}
                    </span>
                  </TableCell>
                  <TableCell className="text-center">{a.attempts}</TableCell>
                  <TableCell className="text-center">
                    <div className="flex justify-center gap-2">
                      <Button size="sm" variant="outline" onClick={() => handleEdit(a)}>
                        <EditIcon className="w-4 h-4" />
                      </Button>
                      <Button size="sm" variant="destructive" onClick={() => handleDelete(a.id)}>
                        <TrashIcon className="w-4 h-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Dialog */}
      <Dialog open={openDialog} onOpenChange={setOpenDialog}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Assignment" : "Tambah Assignment"}</DialogTitle>
          </DialogHeader>

          <div className="space-y-4">
            {/* Assessment */}
            <div>
              <label className="text-sm font-medium mb-1 block">Assessment Type *</label>
              <select
                className="w-full border px-2 py-2 rounded-md"
                value={form.assessment_type_id}
                onChange={(e) => setForm({ ...form, assessment_type_id: e.target.value })}
              >
                <option value="">Pilih Assessment</option>
                {assessments.map((a) => (
                  <option key={a.assessmentId} value={a.assessmentId}>
                    {a.assessmentName}
                  </option>
                ))}
              </select>
            </div>

            {/* Seafarers (multi-select) */}
            {!editing && (
              <div>
                <label className="text-sm font-medium mb-1 block">Pilih Seafarers *</label>
                <Input
                  placeholder="Cari seafarer..."
                  value={form.userSearch}
                  onChange={(e) => setForm({ ...form, userSearch: e.target.value })}
                  className="mb-2"
                />
                <div className="max-h-48 overflow-auto border rounded-md">
                  {filteredUsers.map((u) => (
                    <label
                      key={u.seafarerCode}
                      className="flex items-center gap-2 px-3 py-1.5 hover:bg-accent cursor-pointer"
                    >
                      <input
                        type="checkbox"
                        checked={form.seafarer_codes.includes(u.seafarerCode!)}
                        onChange={() => {
                          const selected = form.seafarer_codes.includes(u.seafarerCode!);
                          setForm({
                            ...form,
                            seafarer_codes: selected
                              ? form.seafarer_codes.filter((c) => c !== u.seafarerCode)
                              : [...form.seafarer_codes, u.seafarerCode!],
                          });
                        }}
                      />
                      <span>
                        {u.nama} <span className="text-gray-500 text-xs">({u.seafarerCode})</span>
                      </span>
                    </label>
                  ))}
                  {filteredUsers.length === 0 && (
                    <p className="text-center text-gray-400 py-3">Tidak ada seafarer ditemukan</p>
                  )}
                </div>
              </div>
            )}

            {/* Status */}
            <div>
              <label className="text-sm font-medium mb-1 block">Status</label>
              <select
                className="w-full border px-2 py-2 rounded-md"
                value={form.status}
                onChange={(e) => setForm({ ...form, status: e.target.value })}
              >
                <option value="ASSIGNED">ASSIGNED</option>
                <option value="IN_PROGRESS">IN_PROGRESS</option>
                <option value="COMPLETED">COMPLETED</option>
              </select>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setOpenDialog(false)}>
              Batal
            </Button>
            <Button onClick={handleSubmit} disabled={loading}>
              {loading ? "Menyimpan..." : editing ? "Update" : "Simpan"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
