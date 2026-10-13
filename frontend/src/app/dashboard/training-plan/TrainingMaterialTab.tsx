"use client";

import { useEffect, useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { authFetch } from "../../lib/auth-fetch";
import { useNarrationPlayer } from "./_hooks/useNarrationPlayer";
import { formatNarrationTime } from "./_hooks/narrationTime";
import { FileText, RefreshCw, Edit, Trash2, Search, Volume2, Square } from "lucide-react";
import { Input } from "@/components/ui/input";

interface ITraining {
  no: number;
  competencyTypeId: number;
  lvl: number;
  deskripsi_perilaku?: string;
  tools_training: string;
  kode: string;
  topik_training: string;
  referensi?: string;
  generated_file_url?: string | null;
  generated_pdf_url?: string | null;
  generated_quiz_url?: string | null;
  generated_at?: string | null;
  competencyType?: {
    id: number;
    code: string;
    name: string;
    description?: string;
    category: string;
  };
}

interface ICompetencyType {
  id: number;
  code: string;
  name: string;
  description: string;
  category: string;
  isActive: boolean;
}

interface GroupedTraining {
  competencyCode: string;
  competencyName: string;
  deskripsi_perilaku?: string;
  trainings: ITraining[];
}

export default function TrainingMaterialTab() {
  const [data, setData] = useState<ITraining[]>([]);
  const [groupedData, setGroupedData] = useState<GroupedTraining[]>([]);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState<number | null>(null);
  const [generatingQuiz, setGeneratingQuiz] = useState<number | null>(null);
  const [selectedTraining, setSelectedTraining] = useState<ITraining | null>(null);
  const [tempReferensi, setTempReferensi] = useState("");
  const [regenerateDialog, setRegenerateDialog] = useState<ITraining | null>(null);
  const [editingTraining, setEditingTraining] = useState<ITraining | null>(null);
  const [editForm, setEditForm] = useState({
    lvl: 0,
    kode: "",
    topik_training: "",
    deskripsi_perilaku: "",
    tools_training: "",
  });
  const [editLoading, setEditLoading] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingTraining, setDeletingTraining] = useState<ITraining | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [expandedGroups, setExpandedGroups] = useState<string[]>([]);

  const [competencyTypes, setCompetencyTypes] = useState<ICompetencyType[]>([]);
  const [createForm, setCreateForm] = useState({
    competency_type_id: 0,
    topik_training: "",
    lvl: 1,
    tools_training: "",
    deskripsi_perilaku: "",
    kode: "",
  });
  const [createLoading, setCreateLoading] = useState(false);

  const apiUrl =
    typeof window === "undefined"
      ? process.env.INTERNAL_API_ENDPOINT
      : process.env.NEXT_PUBLIC_API_ENDPOINT;

  const narration = useNarrationPlayer(apiUrl ?? "", (message) => toast.error(message));

  useEffect(() => {
    const fetchTrainings = async () => {
      try {
        const res = await fetch(`${apiUrl}/trainings`);
        const json = await res.json();
        setData(json.data);
        groupTrainings(json.data);
      } catch {
        setData([]);
      } finally {
        setLoading(false);
      }
    };

    const fetchCompetencyTypes = async () => {
      try {
        const res = await fetch(`${apiUrl}/api/competency-types/active`);
        const json = await res.json();
        setCompetencyTypes(json.data || []);
      } catch {
        setCompetencyTypes([]);
      }
    };

    fetchTrainings();
    fetchCompetencyTypes();
  }, [apiUrl]);

  const groupTrainings = (trainings: ITraining[]) => {
    const grouped: { [key: string]: GroupedTraining } = {};

    trainings.forEach((training) => {
      const competencyCode = training.competencyType?.code || "UNKNOWN";

      if (!grouped[competencyCode]) {
        grouped[competencyCode] = {
          competencyCode: competencyCode,
          competencyName: training.competencyType?.name || "Unknown Competency",
          deskripsi_perilaku: training.deskripsi_perilaku,
          trainings: [],
        };
      }
      grouped[competencyCode].trainings.push(training);
    });

    // Sort groups by competencyCode and trainings by level (descending)
    const sortedGroups = Object.values(grouped)
      .sort((a, b) => a.competencyCode.localeCompare(b.competencyCode))
      .map((group) => ({
        ...group,
        trainings: group.trainings.sort((a, b) => b.lvl - a.lvl),
      }));

    setGroupedData(sortedGroups);
  };

  const filteredGroupedData = useMemo(
    () =>
      groupedData
        .map((group) => ({
          ...group,
          trainings: group.trainings.filter((training) =>
            training.topik_training.toLowerCase().includes(searchTerm.toLowerCase())
          ),
        }))
        .filter((group) => group.trainings.length > 0),
    [groupedData, searchTerm]
  );

  useEffect(() => {
    if (searchTerm) {
      const matchedGroups = filteredGroupedData.map((group) => group.competencyCode);
      setExpandedGroups(matchedGroups);
    }
    // Note: Don't reset to [] when searchTerm is empty, to preserve manually expanded groups
  }, [searchTerm, filteredGroupedData]);

  const handleGenerate = async (item: ITraining, isRegenerate = false) => {
    // The material is written about the topic, so a training without one cannot be generated.
    if (!item.topik_training?.trim()) {
      setRegenerateDialog(null);
      toast.error(
        "Training ini belum punya topik. Isi Topik Training lewat tombol edit, lalu generate lagi."
      );
      return;
    }

    setGenerating(item.no);
    setRegenerateDialog(null);

    // Save current expanded groups before refresh
    const currentExpandedGroups = [...expandedGroups];

    try {
      // Combine keyword (deskripsi_perilaku) with optional referensi
      let combinedReferensi = item.deskripsi_perilaku || "";

      // Use existing referensi or generate default
      const referensiToUse = item.referensi || generateDefaultReferensi(item);

      if (referensiToUse) {
        combinedReferensi = combinedReferensi
          ? `${combinedReferensi}\n\nReferensi Tambahan:\n${referensiToUse}`
          : referensiToUse;
      }

      const res = await authFetch(`${apiUrl}/trainings/generate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          no: item.no,
          kode: item.kode,
          topik_training: item.topik_training,
          kompetensi: item.competencyType?.name || "",
          referensi: combinedReferensi,
          lvl: item.lvl,
          tools_training: item.tools_training,
          old_file_url: isRegenerate ? item.generated_file_url : "",
          old_pdf_url: isRegenerate ? item.generated_pdf_url : "",
        }),
      });

      const result = await res.json();
      if (res.ok) {
        const message = isRegenerate
          ? "Materi Berhasil Di-regenerate!"
          : "Materi Berhasil Digenerate!";
        toast.success(message + " PPTX dan PDF telah tersedia untuk diunduh.");

        // Reload data from server to get persisted URLs
        const refreshRes = await fetch(`${apiUrl}/trainings`);
        const refreshJson = await refreshRes.json();
        setData(refreshJson.data);
        groupTrainings(refreshJson.data);

        // Restore expanded groups after refresh
        setExpandedGroups(currentExpandedGroups);
      } else {
        toast.error(result.error || "Gagal generate materi.");
      }
    } catch {
      toast.error("Terjadi kesalahan saat generate materi.");
    } finally {
      setGenerating(null);
    }
  };

  const handleGenerateQuiz = async (item: ITraining) => {
    // Check if material has been generated
    if (!item.generated_pdf_url) {
      toast.error("Harap generate materi terlebih dahulu sebelum membuat quiz!");
      return;
    }

    setGeneratingQuiz(item.no);

    // Save current expanded groups before refresh
    const currentExpandedGroups = [...expandedGroups];

    try {
      // First, fetch the material content from the PDF URL
      // For simplicity, we'll use the combined referensi as material content
      let materialContent = item.deskripsi_perilaku || "";
      if (item.referensi) {
        materialContent = materialContent
          ? `${materialContent}\n\n${item.referensi}`
          : item.referensi;
      }

      const res = await authFetch(`${apiUrl}/trainings/generate-quiz`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          no: item.no,
          kode: item.kode,
          topik_training: item.topik_training,
          kompetensi: item.competencyType?.name || "",
          material_content: materialContent,
          old_quiz_url: item.generated_quiz_url || "",
        }),
      });

      const result = await res.json();
      if (res.ok) {
        toast.success("Quiz berhasil digenerate! PDF quiz telah tersedia untuk diunduh.");

        const refreshRes = await fetch(`${apiUrl}/trainings`);
        const refreshJson = await refreshRes.json();
        setData(refreshJson.data);
        groupTrainings(refreshJson.data);

        // Restore expanded groups after refresh
        setExpandedGroups(currentExpandedGroups);
      } else {
        toast.error(result.error || "Gagal generate quiz.");
      }
    } catch {
      toast.error("Terjadi kesalahan saat generate quiz.");
    } finally {
      setGeneratingQuiz(null);
    }
  };

  const openCreateDialog = () => {
    setCreateForm({
      competency_type_id: 0,
      topik_training: "",
      lvl: 1,
      tools_training: "",
      deskripsi_perilaku: "",
      kode: "",
    });
    setCreateDialogOpen(true);
  };

  const handleCreateTraining = async () => {
    if (!createForm.competency_type_id) {
      toast.error("Pilih Competency Type terlebih dahulu!");
      return;
    }
    if (
      !createForm.topik_training ||
      !createForm.tools_training ||
      !createForm.deskripsi_perilaku
    ) {
      toast.error("Semua field harus diisi!");
      return;
    }

    setCreateLoading(true);

    // Save current expanded groups before refresh
    const currentExpandedGroups = [...expandedGroups];

    try {
      const res = await authFetch(`${apiUrl}/trainings`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(createForm),
      });

      const result = await res.json();
      if (res.ok) {
        toast.success("Training berhasil ditambahkan!");

        const refreshRes = await fetch(`${apiUrl}/trainings`);
        const refreshJson = await refreshRes.json();
        setData(refreshJson.data);
        groupTrainings(refreshJson.data);

        // Restore expanded groups after refresh
        setExpandedGroups(currentExpandedGroups);

        setCreateDialogOpen(false);
      } else {
        toast.error(result.error || "Gagal menambahkan training.");
      }
    } catch {
      toast.error("Terjadi kesalahan saat menambahkan training.");
    } finally {
      setCreateLoading(false);
    }
  };

  // Generate default referensi text based on training data
  const generateDefaultReferensi = (item: ITraining): string => {
    const topikTraining = item.topik_training || "topik ini";
    const competencyName = item.competencyType?.name || "kompetensi ini";
    const competencyDesc = item.competencyType?.description || "kompetensi yang relevan";
    const level = item.lvl || 1;
    const deskripsiPerilaku = item.deskripsi_perilaku || "mencapai kompetensi yang diharapkan";

    return `buatkan slide training online untuk judul ${topikTraining} untuk mengungkap kompetensi ${competencyName} yaitu ${competencyDesc} terutama untuk level ${level} yang ${deskripsiPerilaku}

buatkan ppt antara 13-15 slide ini sebagai ppt video training online, dengan peserta trainingnya adalah perwira kapal container (SPIL) dengan susunan slidenya

yaitu slide 1: judul, slide 2: objective, slide 3: konsep penjelasan detail topik training, slide 4: challenges/tantangan, slide 5-9: 1 tool yang mengungkap ${topikTraining} (jelaskan steps2nya secara detail untuk perslide), slide 11: studi kasus, slide 12: pemecahan masalah dalam studi kasus menggunakan metode/tools yang dijelaskan, slide 13: penutup/closing.`;
  };

  const handleSaveReferensi = async () => {
    if (!selectedTraining) return;

    try {
      // Save referensi to database via API
      const res = await authFetch(`${apiUrl}/trainings/${selectedTraining.no}/referensi`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ referensi: tempReferensi }),
      });

      if (res.ok) {
        toast.success("Referensi berhasil disimpan!");

        // Update local state with new referensi
        const updatedData = data.map((item) =>
          item.no === selectedTraining.no ? { ...item, referensi: tempReferensi } : item
        );
        setData(updatedData);
        groupTrainings(updatedData);

        setSelectedTraining(null);
        setTempReferensi("");
      } else {
        const result = await res.json();
        toast.error(result.error || "Gagal menyimpan referensi.");
      }
    } catch {
      toast.error("Terjadi kesalahan saat menyimpan referensi.");
    }
  };

  const openRegenerateDialog = (item: ITraining) => {
    setRegenerateDialog(item);
  };

  const openEditDialog = (item: ITraining) => {
    setEditingTraining(item);
    setEditForm({
      lvl: item.lvl,
      kode: item.kode,
      topik_training: item.topik_training,
      deskripsi_perilaku: item.deskripsi_perilaku || "",
      tools_training: item.tools_training,
    });
  };

  const handleSaveEdit = async () => {
    if (!editingTraining) return;

    setEditLoading(true);

    // Save current expanded groups before refresh
    const currentExpandedGroups = [...expandedGroups];

    try {
      const res = await authFetch(`${apiUrl}/trainings/${editingTraining.no}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(editForm),
      });

      const result = await res.json();
      if (res.ok) {
        toast.success("Data training berhasil diupdate!");

        const updatedData = data.map((row) =>
          row.no === editingTraining.no ? { ...row, ...editForm } : row
        );
        setData(updatedData);
        groupTrainings(updatedData);

        // Restore expanded groups after refresh
        setExpandedGroups(currentExpandedGroups);

        setEditingTraining(null);
      } else {
        toast.error(result.error || "Gagal update training.");
      }
    } catch {
      toast.error("Terjadi kesalahan saat update training.");
    } finally {
      setEditLoading(false);
    }
  };

  const openDeleteDialog = (item: ITraining) => {
    setDeletingTraining(item);
    setDeleteDialogOpen(true);
  };

  const handleDelete = async () => {
    if (!deletingTraining) return;

    setDeleteLoading(true);

    // Save current expanded groups before refresh
    const currentExpandedGroups = [...expandedGroups];

    try {
      const res = await authFetch(`${apiUrl}/trainings/${deletingTraining.no}`, {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
        },
      });

      const result = await res.json();
      if (res.ok) {
        toast.success("Training berhasil dihapus!");

        const updatedData = data.filter((row) => row.no !== deletingTraining.no);
        setData(updatedData);
        groupTrainings(updatedData);

        // Restore expanded groups after refresh
        setExpandedGroups(currentExpandedGroups);

        setDeleteDialogOpen(false);
        setDeletingTraining(null);
      } else {
        toast.error(result.error || "Gagal delete training.");
      }
    } catch {
      toast.error("Terjadi kesalahan saat delete training.");
    } finally {
      setDeleteLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
        <div className="space-y-1">
          <CardTitle>Training Materials</CardTitle>
          <CardDescription>
            Generate dan kelola materi training berdasarkan kompetensi
          </CardDescription>
        </div>
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
            <Input
              type="text"
              placeholder="Search training topic..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-10 w-64"
            />
          </div>
          <Button onClick={openCreateDialog} className="gap-2">
            <FileText className="w-4 h-4" />
            Add New Training
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {loading ? (
            <div className="text-center py-12">
              <p className="text-gray-500">Memuat data training...</p>
            </div>
          ) : (
            <Accordion
              type="multiple"
              className="space-y-4"
              value={expandedGroups}
              onValueChange={setExpandedGroups}
            >
              {filteredGroupedData.map((group) => (
                <AccordionItem
                  key={group.competencyCode}
                  value={group.competencyCode}
                  className="border rounded-lg bg-white shadow-sm"
                >
                  <AccordionTrigger className="px-6 py-4 hover:no-underline hover:bg-gray-50">
                    <div className="flex items-center justify-between w-full pr-4">
                      <div className="flex items-center gap-4">
                        <div className="bg-gray-100 text-gray-700 px-3 py-1 rounded-md font-bold text-sm">
                          {group.competencyCode}
                        </div>
                        <div className="text-left">
                          <h3 className="font-semibold text-lg">{group.competencyName}</h3>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 text-sm text-gray-500">
                        <FileText className="w-4 h-4" />
                        <span>{group.trainings.length} trainings</span>
                      </div>
                    </div>
                  </AccordionTrigger>
                  <AccordionContent className="px-6 pb-4">
                    <div className="overflow-x-auto">
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="border-b bg-gray-50">
                            <th className="py-3 px-3 text-left font-semibold">Level</th>
                            <th className="py-3 px-3 text-left font-semibold">Kode</th>
                            <th className="py-3 px-3 text-left font-semibold">Topik Training</th>
                            <th className="py-3 px-3 text-left font-semibold">Tools/Framework</th>
                            <th className="py-3 px-3 text-center font-semibold">Referensi</th>
                            <th className="py-3 px-3 text-center font-semibold">Generate</th>
                            <th className="py-3 px-3 text-center font-semibold">Files</th>
                            <th className="py-3 px-3 text-center font-semibold">Actions</th>
                          </tr>
                        </thead>
                        <tbody>
                          {group.trainings.map((item) => (
                            <tr key={item.no} className="border-b hover:bg-gray-50">
                              <td className="py-3 px-3">
                                <span className="inline-flex items-center justify-center w-8 h-8 bg-gray-200 rounded-full font-semibold text-sm">
                                  {item.lvl}
                                </span>
                              </td>
                              <td className="py-3 px-3 font-medium">{item.kode}</td>
                              <td className="py-3 px-3">{item.topik_training}</td>
                              <td className="py-3 px-3 text-gray-600">{item.tools_training}</td>
                              <td className="py-3 px-3 text-center">
                                <Button
                                  variant="secondary"
                                  size="sm"
                                  onClick={() => {
                                    setSelectedTraining(item);
                                    // Use existing referensi or generate default
                                    setTempReferensi(
                                      item.referensi || generateDefaultReferensi(item)
                                    );
                                  }}
                                >
                                  {item.referensi ? "Edit" : "Tambah"}
                                </Button>
                              </td>
                              <td className="py-3 px-3 text-center">
                                <div className="flex flex-wrap items-center justify-center gap-1.5">
                                  {item.generated_file_url || item.generated_pdf_url ? (
                                    <Button
                                      size="sm"
                                      variant="outline"
                                      onClick={() => openRegenerateDialog(item)}
                                      disabled={generating === item.no}
                                      className="gap-2"
                                    >
                                      <RefreshCw className="w-4 h-4" />
                                      {generating === item.no ? "Regenerating..." : "Regenerate"}
                                    </Button>
                                  ) : (
                                    <Button
                                      size="sm"
                                      onClick={() => handleGenerate(item, false)}
                                      disabled={generating === item.no}
                                    >
                                      {generating === item.no ? "Generating..." : "Generate"}
                                    </Button>
                                  )}
                                  <Button
                                    size="sm"
                                    title="Generate quiz dari materi (butuh PDF)"
                                    variant={item.generated_quiz_url ? "outline" : "default"}
                                    onClick={() => handleGenerateQuiz(item)}
                                    disabled={generatingQuiz === item.no || !item.generated_pdf_url}
                                    className="gap-2"
                                  >
                                    {generatingQuiz === item.no ? (
                                      "Generating..."
                                    ) : item.generated_quiz_url ? (
                                      <>
                                        <RefreshCw className="w-4 h-4" />
                                        Quiz
                                      </>
                                    ) : (
                                      "Quiz"
                                    )}
                                  </Button>
                                  <Button
                                    size="sm"
                                    title={
                                      narration.loadingNo === item.no
                                        ? "Suara sedang dibuat, klik untuk membatalkan"
                                        : "Dengarkan narasi materi (Bahasa Indonesia)"
                                    }
                                    variant={
                                      narration.playingNo === item.no ? "destructive" : "outline"
                                    }
                                    onClick={() =>
                                      item.generated_file_url &&
                                      narration.toggle(item.no, item.generated_file_url)
                                    }
                                    disabled={!item.generated_file_url}
                                    className="gap-2"
                                  >
                                    {narration.loadingNo === item.no ? (
                                      <>
                                        <RefreshCw className="w-4 h-4 animate-spin" />
                                        Menyiapkan suara...
                                      </>
                                    ) : narration.playingNo === item.no ? (
                                      <>
                                        <Square className="w-4 h-4" />
                                        Stop
                                        {narration.progress &&
                                          ` (${formatNarrationTime(narration.progress.current)}${
                                            narration.progress.duration
                                              ? ` / ${formatNarrationTime(narration.progress.duration)}`
                                              : ""
                                          })`}
                                      </>
                                    ) : (
                                      <>
                                        <Volume2 className="w-4 h-4" />
                                        Dengarkan
                                      </>
                                    )}
                                  </Button>
                                </div>
                              </td>
                              <td className="py-3 px-3 text-center">
                                <div className="flex flex-wrap gap-1.5 justify-center">
                                  {!item.generated_file_url &&
                                    !item.generated_pdf_url &&
                                    !item.generated_quiz_url && (
                                      <span className="text-gray-400 text-sm">Belum ada</span>
                                    )}
                                  {item.generated_file_url ? (
                                    <Button
                                      asChild
                                      size="sm"
                                      variant="default"
                                      className="bg-blue-600 hover:bg-blue-700"
                                    >
                                      <a
                                        href={item.generated_file_url}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="gap-2"
                                      >
                                        <FileText className="w-4 h-4" />
                                        PPT
                                      </a>
                                    </Button>
                                  ) : null}
                                  {item.generated_pdf_url ? (
                                    <Button
                                      asChild
                                      size="sm"
                                      variant="default"
                                      className="bg-green-600 hover:bg-green-700"
                                    >
                                      <a
                                        href={item.generated_pdf_url}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="gap-2"
                                      >
                                        <FileText className="w-4 h-4" />
                                        PDF
                                      </a>
                                    </Button>
                                  ) : null}
                                  {item.generated_quiz_url ? (
                                    <Button
                                      asChild
                                      size="sm"
                                      variant="default"
                                      className="bg-purple-600 hover:bg-purple-700"
                                    >
                                      <a
                                        href={item.generated_quiz_url}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="gap-2"
                                      >
                                        <FileText className="w-4 h-4" />
                                        Quiz
                                      </a>
                                    </Button>
                                  ) : null}
                                </div>
                              </td>
                              <td className="py-3 px-3 text-center">
                                <div className="flex gap-2 justify-center">
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => openEditDialog(item)}
                                    className="h-8 w-8 p-0 hover:bg-blue-50 hover:text-blue-600"
                                  >
                                    <Edit className="h-4 w-4" />
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => openDeleteDialog(item)}
                                    className="h-8 w-8 p-0 hover:bg-red-50 hover:text-red-600"
                                  >
                                    <Trash2 className="h-4 w-4" />
                                  </Button>
                                </div>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>
          )}

          {/* Dialog untuk Edit Training Data */}
          <Dialog
            open={editingTraining !== null}
            onOpenChange={(open) => !open && setEditingTraining(null)}
          >
            <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
              <DialogHeader>
                <DialogTitle>Edit Training</DialogTitle>
                <DialogDescription>
                  Kode: <span className="font-semibold">{editingTraining?.kode}</span>
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-4 py-4">
                <div className="space-y-2">
                  <label className="text-sm font-semibold">Level</label>
                  <Input
                    type="number"
                    value={editForm.lvl}
                    onChange={(e) =>
                      setEditForm({ ...editForm, lvl: parseInt(e.target.value) || 0 })
                    }
                    placeholder="Masukkan level"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-semibold">Kode</label>
                  <Input
                    type="text"
                    value={editForm.kode}
                    onChange={(e) => setEditForm({ ...editForm, kode: e.target.value })}
                    placeholder="Masukkan kode"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-semibold">Topik Training</label>
                  <Input
                    type="text"
                    value={editForm.topik_training}
                    onChange={(e) => setEditForm({ ...editForm, topik_training: e.target.value })}
                    placeholder="Masukkan topik training"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-semibold">Deskripsi Perilaku</label>
                  <Textarea
                    value={editForm.deskripsi_perilaku}
                    onChange={(e) =>
                      setEditForm({ ...editForm, deskripsi_perilaku: e.target.value })
                    }
                    placeholder="Masukkan deskripsi perilaku"
                    className="h-24 resize-none"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-semibold">Tools/Framework</label>
                  <Input
                    type="text"
                    value={editForm.tools_training}
                    onChange={(e) => setEditForm({ ...editForm, tools_training: e.target.value })}
                    placeholder="Masukkan tools/framework"
                  />
                </div>
              </div>

              <DialogFooter className="mt-6">
                <Button variant="outline" onClick={() => setEditingTraining(null)}>
                  Batal
                </Button>
                <Button onClick={handleSaveEdit} disabled={editLoading}>
                  {editLoading ? "Menyimpan..." : "Simpan Perubahan"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          {/* Dialog untuk Referensi */}
          <Dialog
            open={selectedTraining !== null}
            onOpenChange={(open) => !open && setSelectedTraining(null)}
          >
            <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
              <DialogHeader>
                <DialogTitle>Kelola Referensi</DialogTitle>
                <DialogDescription>
                  Topik: <span className="font-semibold">{selectedTraining?.topik_training}</span>
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-6 py-4">
                {/* Bagian 1: Keyword (Fixed - dari deskripsi_perilaku) */}
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <h3 className="font-semibold text-lg">Keyword</h3>
                  </div>
                  <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
                    <p className="text-sm whitespace-pre-wrap">
                      {selectedTraining?.deskripsi_perilaku || "Tidak ada keyword"}
                    </p>
                  </div>
                </div>

                <div className="border-t"></div>

                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <h3 className="font-semibold text-lg">Referensi Tambahan</h3>
                    <span className="text-xs bg-green-200 text-green-700 px-2 py-1 rounded">
                      Opsional
                    </span>
                  </div>

                  <Textarea
                    placeholder="Tulis referensi tambahan di sini..."
                    value={tempReferensi}
                    onChange={(e) => setTempReferensi(e.target.value)}
                    className="mt-2 h-32 resize-none"
                  />
                  <p className="text-xs text-gray-500">{tempReferensi.length} karakter</p>
                </div>
              </div>

              <DialogFooter className="mt-6">
                <Button variant="outline" onClick={() => setSelectedTraining(null)}>
                  Batal
                </Button>
                <Button onClick={handleSaveReferensi}>Simpan Referensi</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          {/* Dialog Konfirmasi Regenerate */}
          <AlertDialog
            open={regenerateDialog !== null}
            onOpenChange={(open: boolean) => !open && setRegenerateDialog(null)}
          >
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Regenerate Materi Training?</AlertDialogTitle>
                <AlertDialogDescription>
                  Materi training untuk{" "}
                  <span className="font-semibold">{regenerateDialog?.topik_training}</span> sudah
                  pernah digenerate sebelumnya. Apakah Anda ingin me-regenerate dengan referensi
                  yang baru? File lama (PPTX & PDF) akan dihapus dan diganti dengan yang baru.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Batal</AlertDialogCancel>
                <AlertDialogAction
                  onClick={() => regenerateDialog && handleGenerate(regenerateDialog, true)}
                >
                  Ya, Regenerate
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>

          {/* Dialog Konfirmasi Delete */}
          <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Hapus Training?</AlertDialogTitle>
                <AlertDialogDescription>
                  Apakah Anda yakin ingin menghapus training{" "}
                  <span className="font-semibold">{deletingTraining?.topik_training}</span>?
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Batal</AlertDialogCancel>
                <AlertDialogAction
                  onClick={handleDelete}
                  disabled={deleteLoading}
                  className="bg-red-600 hover:bg-red-700"
                >
                  {deleteLoading ? "Menghapus..." : "Ya, Hapus"}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>

          {/* Dialog Create New Training */}
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
              <DialogHeader>
                <DialogTitle>Tambah Training Baru</DialogTitle>
                <DialogDescription>Isi semua field untuk membuat training baru</DialogDescription>
              </DialogHeader>

              <div className="space-y-4 py-4">
                <div className="space-y-2">
                  <label className="text-sm font-semibold">Competency Type</label>
                  <select
                    value={createForm.competency_type_id}
                    onChange={(e) =>
                      setCreateForm({
                        ...createForm,
                        competency_type_id: parseInt(e.target.value),
                      })
                    }
                    className="w-full px-3 py-2 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value={0}>Pilih Competency Type</option>
                    {competencyTypes.map((ct) => (
                      <option key={ct.id} value={ct.id}>
                        {ct.code} - {ct.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-semibold">Level</label>
                  <Input
                    type="number"
                    value={createForm.lvl}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, lvl: parseInt(e.target.value) || 1 })
                    }
                    placeholder="Masukkan level (default: 1)"
                    min="1"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-semibold">Kode</label>
                  <Input
                    type="text"
                    value={createForm.kode}
                    onChange={(e) => setCreateForm({ ...createForm, kode: e.target.value })}
                    placeholder="Contoh: LDC-001"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-semibold">Topik Training</label>
                  <Input
                    type="text"
                    value={createForm.topik_training}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, topik_training: e.target.value })
                    }
                    placeholder="Masukkan topik training"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-semibold">Deskripsi Perilaku</label>
                  <Textarea
                    value={createForm.deskripsi_perilaku}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, deskripsi_perilaku: e.target.value })
                    }
                    placeholder="Masukkan deskripsi perilaku/keyword"
                    className="h-24 resize-none"
                  />
                </div>

                <div className="space-y-2">
                  <label className="text-sm font-semibold">Tools/Framework</label>
                  <Input
                    type="text"
                    value={createForm.tools_training}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, tools_training: e.target.value })
                    }
                    placeholder="Contoh: React, Angular, Node.js"
                  />
                </div>
              </div>

              <DialogFooter className="mt-6">
                <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
                  Batal
                </Button>
                <Button onClick={handleCreateTraining} disabled={createLoading}>
                  {createLoading ? "Menyimpan..." : "Tambah Training"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </CardContent>
    </Card>
  );
}
