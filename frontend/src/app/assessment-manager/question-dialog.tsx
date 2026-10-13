"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";

import { Plus, Trash2, Upload, X } from "lucide-react";
import Image from "next/image";
import { toast } from "sonner";
import { api } from "@/app/lib/api";
import { useUploadAssessmentImage } from "./_hooks/useAssessment";
import { isValidImageUrl, resolveImageUrl } from "@/app/lib/image";
import { useGetAspectsByAssessmentId } from "./_hooks/useAspect";
import { AspectResponse } from "@/types/aspect";

interface Option {
  optionId?: number;
  optionLetter: string;
  optionText: string;
  score: number;
  scorePercentage: number; // Moodle-style: +100 for correct, negative for wrong
  isImage: number;
  imageUrl?: string;
  imageFile?: File | null;
  imagePreview?: string;
  action?: "create" | "update" | "delete";
  isNew?: boolean;
}

interface Question {
  questionId?: number;
  assessmentId: number;
  questionText: string;
  category?: string;
  isImage?: string;
  imageUrl?: string;
  options?: Option[];
  aspectId?: number;
  questionType?: string;
  acceptableAnswers?: string;
}

const QUESTION_TYPES = [
  { value: "single_choice", label: "Pilihan Tunggal (Single Choice)" },
  { value: "multiple_choice", label: "Pilihan Ganda (Multiple Choice)" },
  { value: "match_choice", label: "Pilihan Berpasangan (2 Jawaban Benar)" },
  { value: "short_answer", label: "Isian Singkat (Short Answer)" },
] as const;

interface QuestionDialogProps {
  open: boolean;
  onClose: (refresh?: boolean) => void;
  question?: Question | null;
  assessmentId: number;
  role: string; // Keep for backward compatibility with getRoleDefaults
  categories: string[];
}

const getRoleDefaults = (role: string) => {
  switch (role) {
    case "va_1":
      return {
        optionCount: 3,
        optionLetters: ["a", "b", "c"],
        defaultScores: [4, 2, 0],
        defaultTexts: ["Opsi A", "Opsi B", "Opsi C"],
      };
    case "va_2":
      return {
        optionCount: 2,
        optionLetters: ["a", "b"],
        defaultScores: [1, 0],
        defaultTexts: ["Benar", "Salah"],
      };
    case "va_3":
      return {
        optionCount: 4,
        optionLetters: ["a", "b", "c", "d"],
        defaultScores: [1, 2, 3, 4],
        defaultTexts: [
          "Sangat Tidak Menggambarkan Diri Saya",
          "Tidak Menggambarkan Diri Saya",
          "Menggambarkan Diri Saya",
          "Sangat Menggambarkan Diri Saya",
        ],
      };
    default: // Crew evaluation roles
      return {
        optionCount: 4,
        optionLetters: ["a", "b", "c", "d"],
        defaultScores: [2, 0, 0, 0], // First option correct by default
        defaultTexts: ["Opsi A", "Opsi B", "Opsi C", "Opsi D"],
      };
  }
};

export default function QuestionDialog({
  open,
  onClose,
  question,
  assessmentId,
  role,
}: QuestionDialogProps) {
  const [formData, setFormData] = useState({
    questionText: "",
    category: "",
    isImage: "0",
    imageUrl: "",
    aspectId: 0,
    questionType: "single_choice",
    acceptableAnswers: "", // JSON array string for short_answer
  });
  const [options, setOptions] = useState<Option[]>([]);

  const [loading, setLoading] = useState(false);
  const [questionImageFile, setQuestionImageFile] = useState<File | null>(null);
  const [questionImagePreview, setQuestionImagePreview] = useState<string>("");

  const uploadImageMutation = useUploadAssessmentImage();
  const { data: aspectsData } = useGetAspectsByAssessmentId(assessmentId);

  useEffect(() => {
    if (open) {
      if (question) {
        // Editing existing question
        // Infer question type from options if it appears to be single_choice but has multiple correct answers
        // This handles legacy data or cases where questionType might be missing
        // Also handle cases where scorePercentage is 0 but score is > 0
        let inferredType = question.questionType || "single_choice";
        if (
          (!question.questionType || question.questionType === "single_choice") &&
          question.options &&
          question.options.length > 0
        ) {
          const correctCount = question.options.filter(
            (o) => (o.scorePercentage || o.score || 0) > 0
          ).length;
          if (
            correctCount === 2 &&
            question.options.every(
              (o) =>
                (o.scorePercentage || o.score || 0) === 0 ||
                (o.scorePercentage || o.score || 0) === 50
            )
          ) {
            inferredType = "match_choice";
          } else if (correctCount > 1) {
            inferredType = "multiple_choice";
          }
        }

        setFormData({
          questionText: question.questionText,
          category: question.category || "",
          // Soal lama bisa punya gambar walau flag-nya "0": ikuti keberadaan gambarnya,
          // supaya gambar tampil (dan tidak terhapus diam-diam saat disimpan).
          isImage: isValidImageUrl(question.imageUrl) ? "1" : question.isImage || "0",
          imageUrl: question.imageUrl || "",
          aspectId: question.aspectId || 0,
          questionType: inferredType,
          acceptableAnswers: question.acceptableAnswers || "",
        });

        // If question has existing image, don't show preview (keep as URL only)
        setQuestionImageFile(null);
        setQuestionImagePreview("");

        let existingOptions = (question.options || []).map((option) => ({
          ...option,
          scorePercentage: option.scorePercentage || option.score || 0,
          action: "update" as const,
          isNew: false,
        }));

        // Recalculate percentages for multiple/match choice to ensure correct display
        if (inferredType === "multiple_choice" || inferredType === "match_choice") {
          const selectedIndices = existingOptions
            .map((opt, i) => (opt.scorePercentage > 0 ? i : -1))
            .filter((i) => i !== -1);

          if (selectedIndices.length > 0) {
            const newPercentage = 100 / selectedIndices.length;
            existingOptions = existingOptions.map((opt, i) => {
              if (selectedIndices.includes(i)) {
                return { ...opt, scorePercentage: newPercentage };
              }
              return opt;
            });
          }
        }

        setOptions(existingOptions);
      } else {
        // Creating new question
        const defaults = getRoleDefaults(role);
        setFormData({
          questionText: "",
          category: "",
          isImage: "0",
          imageUrl: "",
          aspectId: 0,
          questionType: "single_choice",
          acceptableAnswers: "",
        });

        // Reset image upload state for new question
        setQuestionImageFile(null);
        setQuestionImagePreview("");

        const defaultOptions: Option[] = defaults.optionLetters.map((letter, index) => ({
          optionLetter: letter,
          optionText: defaults.defaultTexts[index],
          score: defaults.defaultScores[index],
          scorePercentage: defaults.defaultScores[index] > 0 ? 100 : 0, // Default: first correct option is 100%
          isImage: 0,
          action: "create" as const,
          isNew: true,
        }));

        setOptions(defaultOptions);
      }
    }
  }, [open, question, role]);

  const handleQuestionImageChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file size
    const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB
    if (file.size > MAX_FILE_SIZE) {
      toast.error("Ukuran file tidak boleh lebih dari 10MB");
      return;
    }

    // Validate file type
    const allowedTypes = ["image/jpeg", "image/png", "image/gif", "image/webp"];
    if (!allowedTypes.includes(file.type)) {
      toast.error("Tipe file harus berupa gambar (JPEG, PNG, GIF, WebP)");
      return;
    }

    setQuestionImageFile(file);

    // Create preview
    const reader = new FileReader();
    reader.onloadend = () => {
      setQuestionImagePreview(reader.result as string);
    };
    reader.readAsDataURL(file);
  };

  const handleRemoveQuestionImage = () => {
    setQuestionImageFile(null);
    setQuestionImagePreview("");
    setFormData({ ...formData, imageUrl: "" });
    const input = document.getElementById("questionImage") as HTMLInputElement | null;
    if (input) input.value = "";
  };

  const handleSubmit = async () => {
    if (!formData.questionText.trim()) {
      toast.error("Question text is required");
      return;
    }

    if (formData.questionType !== "short_answer" && options.length === 0) {
      toast.error("At least one option is required");
      return;
    }

    if (formData.questionType === "short_answer" && !formData.acceptableAnswers) {
      toast.error("Acceptable answers are required for short answer questions");
      return;
    }

    if (role === "va_1" && !formData.category) {
      toast.error("Category is required for VA_1 questions");
      return;
    }

    if (formData.questionType === "match_choice") {
      const correctOptions = options.filter((opt) => opt.scorePercentage > 0);
      if (correctOptions.length !== 2) {
        toast.error("Untuk Pilihan Berpasangan, harus ada tepat 2 jawaban benar.");
        return;
      }
    }

    if (formData.questionType === "multiple_choice") {
      const correctOptions = options.filter((opt) => opt.scorePercentage > 0);
      if (correctOptions.length < 1) {
        toast.error("Pilih setidaknya satu jawaban benar.");
        return;
      }
    }

    setLoading(true);
    try {
      // "Memiliki Gambar: Tidak" berarti gambar yang tersimpan ikut dihapus
      let imageUrl = formData.isImage === "1" ? formData.imageUrl : "";

      // Upload image if one was selected
      if (questionImageFile && formData.isImage === "1") {
        try {
          const uploadResult = await uploadImageMutation.mutateAsync(questionImageFile);
          imageUrl = uploadResult.imageUrl;
        } catch {
          toast.error("Gagal mengunggah gambar");
          setLoading(false);
          return;
        }
      }

      // Upload option images if any
      let optionsWithUploadedImages = await Promise.all(
        options.map(async (option) => {
          let optionImageUrl = option.imageUrl || "";

          // If option has a new image file, upload it
          if (option.imageFile) {
            try {
              const uploadResult = await uploadImageMutation.mutateAsync(option.imageFile);
              optionImageUrl = uploadResult.imageUrl;
            } catch (error) {
              toast.error(`Gagal mengunggah gambar untuk opsi ${option.optionLetter}`);
              throw error;
            }
          }

          return {
            ...option,
            imageUrl: optionImageUrl,
            isImage: optionImageUrl ? 1 : 0,
          };
        })
      );

      // Normalize scores for match_choice to ensure 50-50 split regardless of previous state
      if (formData.questionType === "match_choice") {
        optionsWithUploadedImages = optionsWithUploadedImages.map((opt) => ({
          ...opt,
          scorePercentage: opt.scorePercentage > 0 ? 50 : 0,
          score: opt.scorePercentage > 0 ? 1 : 0,
        }));
      }

      if (question?.questionId) {
        // Update existing question using combined endpoint
        // Convert acceptable answers from comma-separated to JSON array
        let acceptableAnswersJson: string | null = null;
        if (formData.questionType === "short_answer" && formData.acceptableAnswers) {
          const answers = formData.acceptableAnswers
            .split(",")
            .map((a) => a.trim())
            .filter((a) => a);
          acceptableAnswersJson = JSON.stringify(answers);
        }

        const updateData = {
          role,
          assessmentId,
          questionText: formData.questionText,
          category: role === "va_1" ? formData.category : null,
          isImage: formData.isImage,
          imageUrl: imageUrl || null,
          aspectId: formData.aspectId || null,
          questionType: formData.questionType,
          acceptableAnswers: acceptableAnswersJson,
          options: optionsWithUploadedImages.map((option) => {
            // Determine action based on option state
            let action = option.action || "update";

            // If option has no ID, it's a create operation
            if (!option.optionId) {
              action = "create";
            }

            return {
              optionId: option.optionId || null,
              optionLetter: option.optionLetter,
              optionText: option.optionText,
              score: option.score,
              scorePercentage: option.scorePercentage,
              isImage: option.isImage,
              imageUrl: option.imageUrl || null,
              action: action,
            };
          }),
        };

        await api.put(`/api/questions-with-options/${question.questionId}`, updateData);
        toast.success("Pertanyaan berhasil diperbarui");
      } else {
        // Create new question using combined endpoint
        // Convert acceptable answers from comma-separated to JSON array
        let createAcceptableAnswersJson: string | null = null;
        if (formData.questionType === "short_answer" && formData.acceptableAnswers) {
          const answers = formData.acceptableAnswers
            .split(",")
            .map((a) => a.trim())
            .filter((a) => a);
          createAcceptableAnswersJson = JSON.stringify(answers);
        }

        const createData = {
          role,
          assessmentId,
          questionText: formData.questionText,
          category: role === "va_1" ? formData.category : "",
          isImage: formData.isImage,
          imageUrl: imageUrl || "",
          aspectId: formData.aspectId || null,
          questionType: formData.questionType,
          acceptableAnswers: createAcceptableAnswersJson,
          options: optionsWithUploadedImages.map((option) => ({
            optionLetter: option.optionLetter,
            optionText: option.optionText,
            score: option.score,
            scorePercentage: option.scorePercentage,
            isImage: option.isImage,
            imageUrl: option.imageUrl || null,
          })),
        };

        await api.post("/api/questions-with-options", createData);
        toast.success("Pertanyaan berhasil dibuat");
      }

      onClose(true);
    } catch (error) {
      console.error("Error saving question:", error);
      const errorMessage =
        (error as { response?: { data?: { error?: string } } })?.response?.data?.error ||
        "Gagal menyimpan pertanyaan";
      toast.error(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const addOption = () => {
    const nextLetter = String.fromCharCode(97 + options.length); // a, b, c, d, etc.
    setOptions([
      ...options,
      {
        optionLetter: nextLetter,
        optionText: `Opsi ${nextLetter.toUpperCase()}`,
        score: 0,
        scorePercentage: 0,
        isImage: 0,
        action: "create" as const,
        isNew: true,
      },
    ]);
  };

  const removeOption = (index: number) => {
    const optionToRemove = options[index];

    if (optionToRemove.isNew) {
      // If it's a newly added option, just remove it from the array
      setOptions(options.filter((_, i) => i !== index));
    } else {
      // If it's an existing option, mark it for deletion but keep in array for backend processing
      // We'll filter it out when sending to backend but keep the record for deletion
      setOptions(options.filter((_, i) => i !== index));
    }
  };

  const updateOption = (index: number, field: keyof Option, value: string | boolean | number) => {
    const updatedOptions = [...options];
    updatedOptions[index] = { ...updatedOptions[index], [field]: value };
    setOptions(updatedOptions);
  };

  const updateOptionFields = (index: number, updates: Partial<Option>) => {
    const updatedOptions = [...options];
    updatedOptions[index] = { ...updatedOptions[index], ...updates };
    setOptions(updatedOptions);
  };

  const handleMultipleChoiceChange = (index: number, checked: boolean) => {
    const currentOptions = [...options];
    const selectedIndices = new Set<number>();

    // Determine which indices will be selected after this change
    currentOptions.forEach((opt, i) => {
      if (i === index) {
        if (checked) selectedIndices.add(i);
      } else {
        if (opt.scorePercentage > 0) selectedIndices.add(i);
      }
    });

    const count = selectedIndices.size;
    const newPercentage = count > 0 ? 100 / count : 0;

    const newOptions = currentOptions.map((opt, i) => {
      if (selectedIndices.has(i)) {
        return { ...opt, scorePercentage: newPercentage, score: 1 };
      } else {
        return { ...opt, scorePercentage: 0, score: 0 };
      }
    });

    setOptions(newOptions);
  };

  const handleOptionImageChange = (e: React.ChangeEvent<HTMLInputElement>, index: number) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file size
    const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB
    if (file.size > MAX_FILE_SIZE) {
      toast.error("Ukuran file tidak boleh lebih dari 10MB");
      return;
    }

    // Validate file type
    const allowedTypes = ["image/jpeg", "image/png", "image/gif", "image/webp"];
    if (!allowedTypes.includes(file.type)) {
      toast.error("Tipe file harus berupa gambar (JPEG, PNG, GIF, WebP)");
      return;
    }

    // Create preview
    const reader = new FileReader();
    reader.onloadend = () => {
      const updatedOptions = [...options];
      updatedOptions[index] = {
        ...updatedOptions[index],
        imageFile: file,
        imagePreview: reader.result as string,
      };
      setOptions(updatedOptions);
    };
    reader.readAsDataURL(file);
  };

  const handleRemoveOptionImage = (index: number) => {
    const updatedOptions = [...options];
    updatedOptions[index] = {
      ...updatedOptions[index],
      imageFile: null,
      imagePreview: "",
      imageUrl: "",
      isImage: 0,
    };
    setOptions(updatedOptions);
    const input = document.getElementById(`optionImage-${index}`) as HTMLInputElement | null;
    if (input) input.value = "";
  };

  // Gambar yang ditampilkan: file baru yang dipilih, atau gambar yang sudah tersimpan.
  const questionImageSrc =
    questionImagePreview ||
    (isValidImageUrl(formData.imageUrl) ? resolveImageUrl(formData.imageUrl) : "");

  const getOptionImageSrc = (option: Option): string =>
    option.imagePreview ||
    (isValidImageUrl(option.imageUrl) ? resolveImageUrl(option.imageUrl as string) : "");

  return (
    <Dialog open={open} onOpenChange={() => onClose()}>
      <DialogContent className="!w-[95vw] !max-w-[1400px] !h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="sr-only">
            {question?.questionId ? "Perbarui Pertanyaan" : "Buat Pertanyaan Baru"}
          </DialogTitle>
          <div className="flex justify-center items-center mb-6">
            <h1 className="text-2xl font-bold uppercase">
              {question?.questionId ? "Perbarui Pertanyaan" : "Buat Pertanyaan Baru"}
            </h1>
          </div>
        </DialogHeader>

        <div className="grid grid-cols-2 gap-6 mt-6">
          {/* Left Column - Question Details */}
          <div className="flex flex-col gap-6">
            <div className="border rounded-xl shadow-sm p-6 bg-white">
              <h2 className="font-bold text-lg mb-4">Detail Pertanyaan</h2>
              <div className="space-y-4">
                <div>
                  <Label htmlFor="questionText" className="text-sm font-medium">
                    Teks Pertanyaan <span className="text-red-500">*</span>
                  </Label>
                  <Textarea
                    id="questionText"
                    value={formData.questionText}
                    onChange={(e) => setFormData({ ...formData, questionText: e.target.value })}
                    placeholder="Masukkan teks pertanyaan..."
                    className="min-h-[120px] mt-2"
                  />
                </div>

                <div>
                  <Label htmlFor="questionType" className="text-sm font-medium">
                    Tipe Pertanyaan <span className="text-red-500">*</span>
                  </Label>
                  <Select
                    value={formData.questionType}
                    onValueChange={(value) => setFormData({ ...formData, questionType: value })}
                  >
                    <SelectTrigger className="mt-2">
                      <SelectValue placeholder="Pilih tipe pertanyaan..." />
                    </SelectTrigger>
                    <SelectContent>
                      {QUESTION_TYPES.map((type) => (
                        <SelectItem key={type.value} value={type.value}>
                          {type.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-gray-500 mt-1">
                    {formData.questionType === "single_choice" &&
                      "Satu jawaban benar dari beberapa pilihan"}
                    {formData.questionType === "multiple_choice" &&
                      "Beberapa jawaban benar, skor persentase per opsi"}
                    {formData.questionType === "match_choice" &&
                      "Harus memilih tepat 2 jawaban yang benar"}
                    {formData.questionType === "short_answer" &&
                      "Jawaban teks bebas, cocokkan dengan jawaban yang diterima"}
                  </p>
                </div>

                {formData.questionType === "short_answer" && (
                  <div>
                    <Label htmlFor="acceptableAnswers" className="text-sm font-medium">
                      Jawaban yang Diterima (pisahkan dengan koma)
                    </Label>
                    <Input
                      id="acceptableAnswers"
                      value={formData.acceptableAnswers}
                      onChange={(e) =>
                        setFormData({ ...formData, acceptableAnswers: e.target.value })
                      }
                      placeholder="jawaban1, jawaban2, jawaban3..."
                      className="mt-2"
                    />
                    <p className="text-xs text-gray-500 mt-1">
                      Jawaban tidak case-sensitive. Pisahkan beberapa jawaban dengan koma.
                    </p>
                  </div>
                )}

                {aspectsData && aspectsData.length > 0 && (
                  <div>
                    <Label htmlFor="aspectId" className="text-sm font-medium">
                      Aspek Penilaian
                    </Label>
                    <Select
                      value={formData.aspectId.toString()}
                      onValueChange={(value) =>
                        setFormData({
                          ...formData,
                          aspectId: parseInt(value),
                        })
                      }
                    >
                      <SelectTrigger className="mt-2">
                        <SelectValue placeholder="Pilih aspek..." />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="0">Tidak Ada Aspek</SelectItem>
                        {aspectsData.map((aspect: AspectResponse) => (
                          <SelectItem key={aspect.id} value={aspect.id.toString()}>
                            {aspect.name} ({aspect.weight}%)
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <Label htmlFor="isImage" className="text-sm font-medium">
                      Memiliki Gambar
                    </Label>
                    <Select
                      value={formData.isImage}
                      onValueChange={(value) => setFormData({ ...formData, isImage: value })}
                    >
                      <SelectTrigger className="mt-2">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="0">Tidak</SelectItem>
                        <SelectItem value="1">Ya</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  {formData.isImage === "1" && (
                    <div className="col-span-2 gap-2">
                      <Label className="text-sm font-medium">Upload Gambar</Label>

                      {questionImageSrc && (
                        <div className="relative w-full h-40 bg-gray-100 rounded-lg overflow-hidden mt-2 mb-3">
                          <Image
                            src={questionImageSrc}
                            alt="Gambar soal"
                            fill
                            unoptimized
                            className="object-contain"
                          />
                          <button
                            onClick={handleRemoveQuestionImage}
                            disabled={loading}
                            title="Hapus gambar"
                            aria-label="Hapus gambar soal"
                            className="absolute top-2 right-2 flex items-center gap-1 bg-red-500 text-white rounded-full px-2 py-1 text-xs hover:bg-red-600 disabled:opacity-50"
                            type="button"
                          >
                            <X className="h-4 w-4" />
                            Hapus
                          </button>
                        </div>
                      )}

                      <div className="flex items-center justify-center w-full">
                        <label
                          htmlFor="questionImage"
                          className="flex flex-col items-center justify-center w-full h-32 border-2 border-dashed border-gray-300 rounded-lg cursor-pointer hover:border-gray-400 bg-gray-50"
                        >
                          <div className="flex flex-col items-center justify-center pt-5 pb-6">
                            <Upload className="h-6 w-6 text-gray-400 mb-2" />
                            <p className="text-sm text-gray-500">Klik untuk upload (Max 10MB)</p>
                          </div>
                          <Input
                            id="questionImage"
                            type="file"
                            accept="image/*"
                            onChange={handleQuestionImageChange}
                            disabled={loading}
                            className="hidden"
                          />
                        </label>
                      </div>
                      <p className="text-xs text-gray-500 mt-2">Format: JPEG, PNG, GIF, WebP</p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Right Column - Options (Hidden for Short Answer) */}
          {formData.questionType !== "short_answer" && (
            <div className="flex flex-col gap-6">
              <div className="border rounded-xl shadow-sm p-6 bg-white h-full">
                <div className="flex items-center justify-between mb-4">
                  <h2 className="font-bold text-lg">Opsi Jawaban</h2>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={addOption}
                    className="flex items-center gap-2"
                  >
                    <Plus className="h-4 w-4" />
                    Tambah Opsi
                  </Button>
                </div>

                <div className="space-y-3 h-full overflow-y-auto">
                  {options.map((option, index) => (
                    <div key={index} className="border rounded-lg p-4 bg-gray-50">
                      <div className="flex items-center justify-between mb-3">
                        <Badge variant="outline" className="bg-white">
                          Opsi {option.optionLetter.toUpperCase()}
                        </Badge>
                        {options.length > 1 && (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => removeOption(index)}
                            className="text-red-600 hover:text-red-700"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        )}
                      </div>

                      <div className="space-y-3">
                        <div>
                          <Label className="text-sm font-medium">Teks Opsi</Label>
                          <Input
                            value={option.optionText}
                            onChange={(e) => updateOption(index, "optionText", e.target.value)}
                            placeholder="Masukkan teks opsi..."
                            className="mt-1"
                          />
                        </div>
                        <div>
                          {formData.questionType === "single_choice" ? (
                            <>
                              <Label className="text-sm font-medium">Skor</Label>
                              <Input
                                type="number"
                                value={option.score === 0 ? "" : option.score}
                                onChange={(e) => {
                                  const value = e.target.value;
                                  updateOption(index, "score", value === "" ? 0 : parseInt(value));
                                }}
                                onWheel={(e) => e.currentTarget.blur()}
                                placeholder="0"
                                className="mt-1 [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                              />
                            </>
                          ) : formData.questionType === "match_choice" ? (
                            <div className="flex items-center space-x-2 mt-4">
                              <Checkbox
                                id={`correct-${index}`}
                                checked={option.scorePercentage > 0}
                                onCheckedChange={(checked) => {
                                  if (checked) {
                                    const currentCheckedCount = options.filter(
                                      (opt) => opt.scorePercentage > 0
                                    ).length;
                                    if (currentCheckedCount >= 2) {
                                      toast.error("Hanya boleh memilih tepat 2 jawaban benar.");
                                      return;
                                    }
                                  }
                                  updateOptionFields(index, {
                                    scorePercentage: checked ? 50 : 0,
                                    score: checked ? 1 : 0,
                                  });
                                }}
                              />
                              <Label
                                htmlFor={`correct-${index}`}
                                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                              >
                                Jawaban Benar
                              </Label>
                            </div>
                          ) : formData.questionType === "multiple_choice" ? (
                            <div className="flex items-center space-x-2 mt-4">
                              <Checkbox
                                id={`correct-mc-${index}`}
                                checked={option.scorePercentage > 0}
                                onCheckedChange={(checked) => {
                                  handleMultipleChoiceChange(index, checked as boolean);
                                }}
                              />
                              <Label
                                htmlFor={`correct-mc-${index}`}
                                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                              >
                                Jawaban Benar (
                                {option.scorePercentage > 0
                                  ? `${parseFloat(option.scorePercentage.toFixed(2))}%`
                                  : "0%"}
                                )
                              </Label>
                            </div>
                          ) : (
                            <>
                              <Label className="text-sm font-medium">Persentase Skor (%)</Label>
                              <Input
                                type="number"
                                value={option.scorePercentage === 0 ? "" : option.scorePercentage}
                                onChange={(e) => {
                                  const value = e.target.value;
                                  updateOption(
                                    index,
                                    "scorePercentage",
                                    value === "" ? 0 : parseFloat(value)
                                  );
                                }}
                                onWheel={(e) => e.currentTarget.blur()}
                                placeholder="Contoh: 33.33 atau -50"
                                className="mt-1 [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                              />
                              <p className="text-[10px] text-gray-500 mt-1">
                                (+) untuk jawaban benar, (-) untuk jawaban salah
                              </p>
                            </>
                          )}
                        </div>

                        {/* Image Upload for Option */}
                        <div>
                          <Label className="text-sm font-medium mb-2 block">
                            Gambar Opsi (Opsional)
                          </Label>

                          {getOptionImageSrc(option) && (
                            <div className="relative w-full h-28 bg-gray-100 rounded-lg overflow-hidden mb-2">
                              <Image
                                src={getOptionImageSrc(option)}
                                alt={`Gambar Opsi ${option.optionLetter}`}
                                fill
                                unoptimized
                                className="object-contain"
                              />
                              <button
                                onClick={() => handleRemoveOptionImage(index)}
                                disabled={loading}
                                title="Hapus gambar"
                                aria-label={`Hapus gambar opsi ${option.optionLetter}`}
                                className="absolute top-1 right-1 flex items-center gap-1 bg-red-500 text-white rounded-full px-2 py-1 text-xs hover:bg-red-600 disabled:opacity-50"
                                type="button"
                              >
                                <X className="h-3 w-3" />
                                Hapus
                              </button>
                            </div>
                          )}

                          <label
                            htmlFor={`optionImage-${index}`}
                            className="flex items-center justify-center w-full h-24 border-2 border-dashed border-gray-300 rounded-lg cursor-pointer hover:border-gray-400 bg-gray-50"
                          >
                            <div className="flex flex-col items-center justify-center pt-3 pb-3">
                              <Upload className="h-5 w-5 text-gray-400 mb-1" />
                              <p className="text-xs text-gray-500">Klik untuk upload</p>
                            </div>
                            <Input
                              id={`optionImage-${index}`}
                              type="file"
                              accept="image/*"
                              onChange={(e) => handleOptionImageChange(e, index)}
                              disabled={loading}
                              className="hidden"
                            />
                          </label>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="mt-6 pt-4 border-t">
          <div className="flex justify-end gap-3 w-full">
            <Button variant="outline" onClick={() => onClose()}>
              Batal
            </Button>
            <Button onClick={handleSubmit} disabled={loading}>
              {loading
                ? "Menyimpan..."
                : question?.questionId
                  ? "Perbarui Pertanyaan"
                  : "Buat Pertanyaan"}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
