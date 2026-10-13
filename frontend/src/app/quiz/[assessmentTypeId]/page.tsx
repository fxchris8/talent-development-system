"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useParams } from "next/navigation";
import { useGetQuizData, useSubmitQuiz, QuizAnswerSubmit } from "../_hooks/useQuiz";
import {
  useCheckSeafarerAssignment,
  useIncrementAttempts,
} from "../_hooks/useCheckSeafarerAssignment";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Checkbox } from "@/components/ui/checkbox";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import { useCheckAssessmentTypeStatus } from "@/app/value-assessment/_hooks/useAssessmentTypeStatus";
import Image from "next/image";
import Greetings from "./greetings";
import Completion from "./completion";
import { isValidImageUrl, resolveImageUrl } from "@/app/lib/image";
import { TutorialDisplay } from "@/components/tutorial-display";

export default function QuizPage() {
  const params = useParams();
  const assessmentTypeId = Number(params.assessmentTypeId);

  const { data: quizData, isLoading, error } = useGetQuizData(assessmentTypeId);
  const { data: statusData, isLoading: isStatusLoading } =
    useCheckAssessmentTypeStatus(assessmentTypeId);
  const submitMutation = useSubmitQuiz();
  const incrementAttemptsMutation = useIncrementAttempts();

  // Step navigation: 0 = greetings, 1...n = quiz sections, n+1 = completion
  const [currentStep, setCurrentStep] = useState(0);
  const [seamanCode, setSeamanCode] = useState("");
  const [email, setEmail] = useState("");
  const [answers, setAnswers] = useState<Record<number, QuizAnswerSubmit>>({});

  // Verification state for seafarer code input (step 1)
  const [isVerifying, setIsVerifying] = useState(false);
  const [isVerified, setIsVerified] = useState(false);
  const [verificationError, setVerificationError] = useState("");
  const [autoFilledData, setAutoFilledData] = useState({ name: "", rank: "", vessel: "" });
  const [querySeafarerCode, setQuerySeafarerCode] = useState<string>("");

  // Client-side flag for localStorage
  const [isClient, setIsClient] = useState(false);

  // Question navigation state (for one-question-at-a-time display)
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);

  // Track which quiz sections have had their tutorial dismissed
  const [tutorialDismissed, setTutorialDismissed] = useState<Record<number, boolean>>({});

  // Timer state (per section)
  const [timerSecondsLeft, setTimerSecondsLeft] = useState<number | null>(null);
  const timerIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const timerExpiredRef = useRef<Record<number, boolean>>({});

  // localStorage key with expiry 3 days
  const STORAGE_KEY = `quizFormData_${assessmentTypeId}`;
  const EXPIRY_MINUTES = 3 * 24 * 60; // 3 days (4320 minutes)

  // API call to check seafarer assignment
  const assignmentQuery = useCheckSeafarerAssignment(querySeafarerCode, assessmentTypeId);

  // Effect to handle assignment query results
  useEffect(() => {
    if (assignmentQuery.data) {
      const data = assignmentQuery.data;

      if (!data.isAssigned) {
        setVerificationError(
          data.message || "Anda tidak diassign untuk assessment ini. Silakan hubungi administrator."
        );
        setAutoFilledData({ name: "", rank: "", vessel: "" });
        setIsVerified(false);
        setIsVerifying(false);
        return;
      }

      // If assigned but no personal data found
      if (!data.personalData) {
        setVerificationError(
          "Data pribadi tidak ditemukan. Silakan hubungi administrator untuk memperbarui data Anda."
        );
        setAutoFilledData({ name: "", rank: "", vessel: "" });
        setIsVerified(false);
        setIsVerifying(false);
        return;
      }

      // Check if attempts have been exceeded
      if (data.maxAttempts !== null && data.attemptsCount >= data.maxAttempts) {
        setVerificationError(
          `Anda sudah melebihi batas maksimal attempts (${data.attemptsCount}/${data.maxAttempts}). Hubungi administrator untuk bantuan lebih lanjut.`
        );
        setAutoFilledData({ name: "", rank: "", vessel: "" });
        setIsVerified(false);
        setIsVerifying(false);
        return;
      }

      // If assigned and personal data exists, populate fields
      setAutoFilledData({
        name: data.personalData.nama || "",
        rank: data.personalData.jabatan || "",
        vessel: data.personalData.vesselName || "",
      });

      setIsVerified(true);
      setVerificationError("");
      setIsVerifying(false);
    } else if (assignmentQuery.error) {
      // Check if it's a 401 error (unauthorized/not found)
      const error = assignmentQuery.error as unknown as {
        response?: { status?: number };
        status?: number;
        message?: string;
      };
      const isUnauthorized =
        error?.response?.status === 401 ||
        error?.status === 401 ||
        (error?.message && error.message.includes("401"));

      const errorMessage = isUnauthorized
        ? "User tidak ditemukan atau tidak diassign"
        : assignmentQuery.error instanceof Error
          ? assignmentQuery.error.message
          : "Terjadi kesalahan koneksi";

      setVerificationError(errorMessage);
      setIsVerified(false);
      setIsVerifying(false);
    }

    if (assignmentQuery.isLoading) {
      setIsVerifying(true);
    }
  }, [assignmentQuery.data, assignmentQuery.error, assignmentQuery.isLoading]);

  // Load data from localStorage on mount
  useEffect(() => {
    setIsClient(true);
    try {
      const item = window.localStorage.getItem(STORAGE_KEY);
      if (item) {
        const parsed = JSON.parse(item);

        // Check if data has expiry field and hasn't expired
        if (parsed.expiresAt) {
          const now = new Date().getTime();
          if (now <= parsed.expiresAt) {
            const data = parsed.value;
            if (data && (data.email || data.seamanCode || data.currentStep > 0)) {
              setCurrentStep(data.currentStep || 0);
              setSeamanCode(data.seamanCode || "");
              setEmail(data.email || "");
              setAnswers(data.answers || {});
              setAutoFilledData(data.autoFilledData || { name: "", rank: "", vessel: "" });
            }
          } else {
            // Data expired, remove from localStorage
            window.localStorage.removeItem(STORAGE_KEY);
          }
        }
      }
    } catch (error) {
      console.warn("Error loading stored quiz data:", error);
    }
  }, [STORAGE_KEY]);

  const totalQuizSteps = (quizData?.assessments?.length || 0) + 2; // +2 for greetings and seaman code

  // Save data to localStorage whenever state changes
  useEffect(() => {
    // Prevent saving if we are at the completion step or beyond
    if (currentStep >= totalQuizSteps && totalQuizSteps > 2) {
      return;
    }

    if (isClient && (email || seamanCode || currentStep > 0)) {
      try {
        const now = new Date().getTime();
        const expiryTime = now + EXPIRY_MINUTES * 60 * 1000; // Convert minutes to ms

        const dataToSave = {
          currentStep,
          seamanCode,
          email,
          answers,
          autoFilledData,
          timestamp: new Date().toISOString(),
        };

        const storageData = {
          value: dataToSave,
          expiresAt: expiryTime,
        };

        window.localStorage.setItem(STORAGE_KEY, JSON.stringify(storageData));
      } catch (error) {
        console.warn("Error saving quiz data:", error);
      }
    }
  }, [
    isClient,
    currentStep,
    seamanCode,
    email,
    answers,
    autoFilledData,
    STORAGE_KEY,
    EXPIRY_MINUTES,
    totalQuizSteps,
  ]);

  const formatTimerTime = (seconds: number): string => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    if (h > 0) return `${h}:${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
    return `${m}:${s.toString().padStart(2, "0")}`;
  };

  const handleGreetingsNext = (data: { email: string; consent: boolean }) => {
    setEmail(data.email);
    setCurrentStep(1); // Move to quiz sections
  };

  const handleStartQuiz = async () => {
    if (!seamanCode.trim()) {
      toast.error("Please enter your Seaman Code to start.");
      return;
    }

    try {
      await incrementAttemptsMutation.mutateAsync({
        seafarerCode: seamanCode,
        assessmentTypeId: assessmentTypeId,
      });
      setCurrentStep(2); // Move to first quiz section
    } catch (error) {
      toast.error("Gagal memulai quiz. Silakan coba lagi.");
      console.error("Failed to start quiz:", error);
    }
  };

  const handleAnswerChange = (
    questionId: number,
    value: string | { id: number; checked: boolean },
    type: string
  ) => {
    setAnswers((prev) => {
      const currentAnswer = prev[questionId] || { questionId };

      if (type === "single_choice") {
        return {
          ...prev,
          [questionId]: { ...currentAnswer, selectedOptions: [Number(value)] },
        };
      } else if (type === "short_answer") {
        return {
          ...prev,
          [questionId]: { ...currentAnswer, textAnswer: value as string },
        };
      } else if (type === "multiple_choice" || type === "match_choice") {
        const valObj = value as { id: number; checked: boolean };
        const optionId = Number(valObj.id);
        const isChecked = valObj.checked;
        let newSelected = currentAnswer.selectedOptions || [];

        if (isChecked) {
          if (type === "match_choice" && newSelected.length >= 2) {
            toast.error("Maksimal 2 jawaban untuk tipe ini");
            return prev;
          }
          newSelected = [...newSelected, optionId];
        } else {
          newSelected = newSelected.filter((id) => id !== optionId);
        }

        return {
          ...prev,
          [questionId]: { ...currentAnswer, selectedOptions: newSelected },
        };
      }
      return prev;
    });
  };

  // Question navigation functions
  const handleQuestionNext = () => {
    if (currentAssessment && currentQuestionIndex < currentAssessment.questions.length - 1) {
      setCurrentQuestionIndex((prev) => prev + 1);
      window.scrollTo(0, 0);
    }
  };

  const handleQuestionPrev = () => {
    if (currentQuestionIndex > 0) {
      setCurrentQuestionIndex((prev) => prev - 1);
      window.scrollTo(0, 0);
    }
  };

  const jumpToQuestion = (index: number) => {
    setCurrentQuestionIndex(index);
    window.scrollTo(0, 0);
  };

  // Get current assessment section (currentStep - 2 because step 0=greetings, step 1=seaman code input)
  const quizStepIndex = currentStep - 2;
  const currentAssessment = quizData?.assessments?.[quizStepIndex];

  // Reset currentQuestionIndex when moving to a different section
  useEffect(() => {
    setCurrentQuestionIndex(0);
    window.scrollTo(0, 0);
  }, [quizStepIndex]);

  const handleNext = useCallback(() => {
    if (currentStep < totalQuizSteps - 1) {
      setCurrentStep((prev) => prev + 1);
      window.scrollTo(0, 0);
    } else {
      // Last quiz section, submit
      handleSubmit();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentStep, totalQuizSteps]);

  const handlePrev = () => {
    if (currentStep > 1) {
      // Can't go back from greetings
      setCurrentStep((prev) => prev - 1);
      window.scrollTo(0, 0);
    }
  };

  // Function to clear stored data (when quiz is completed)
  const clearStoredData = () => {
    try {
      window.localStorage.removeItem(STORAGE_KEY);
      // Clear all section timer keys
      const assessmentCount = quizData?.assessments?.length ?? 0;
      for (let i = 0; i < assessmentCount; i++) {
        window.localStorage.removeItem(`quizTimer_${assessmentTypeId}_${i}`);
      }
    } catch (error) {
      console.warn("Error clearing stored data:", error);
    }
  };

  // Initialize/restore section timer when entering a section

  useEffect(() => {
    if (!currentAssessment?.usingTimer || !currentAssessment?.timerLimitMinutes) {
      setTimerSecondsLeft(null);
      if (timerIntervalRef.current) clearInterval(timerIntervalRef.current);
      return;
    }

    // Wait until tutorial is dismissed (if there is one)
    const hasTutorial = !!currentAssessment.tutorialContent;
    if (hasTutorial && !tutorialDismissed[quizStepIndex]) {
      setTimerSecondsLeft(null);
      if (timerIntervalRef.current) clearInterval(timerIntervalRef.current);
      return;
    }

    const timerKey = `quizTimer_${assessmentTypeId}_${quizStepIndex}`;
    const existing = window.localStorage.getItem(timerKey);
    let expiresAt: number;

    if (existing) {
      const data = JSON.parse(existing) as { expiresAt: number };
      expiresAt = data.expiresAt;
    } else {
      expiresAt = Date.now() + currentAssessment.timerLimitMinutes * 60 * 1000;
      window.localStorage.setItem(timerKey, JSON.stringify({ value: true, expiresAt }));
    }

    const updateTimer = () => {
      const remaining = Math.max(0, Math.floor((expiresAt - Date.now()) / 1000));
      setTimerSecondsLeft(remaining);
      if (remaining <= 0 && timerIntervalRef.current) {
        clearInterval(timerIntervalRef.current);
      }
    };

    updateTimer();
    if (timerIntervalRef.current) clearInterval(timerIntervalRef.current);
    timerIntervalRef.current = setInterval(updateTimer, 1000);

    return () => {
      if (timerIntervalRef.current) clearInterval(timerIntervalRef.current);
    };
  }, [quizStepIndex, tutorialDismissed, currentAssessment, assessmentTypeId]);

  // Auto-proceed/submit when timer expires
  useEffect(() => {
    if (
      timerSecondsLeft === 0 &&
      currentAssessment?.usingTimer &&
      !timerExpiredRef.current[quizStepIndex] &&
      (currentAssessment?.questions?.length ?? 0) > 0
    ) {
      timerExpiredRef.current[quizStepIndex] = true;
      toast.warning("Waktu habis! Melanjutkan otomatis.");
      handleNext();
    }
  }, [timerSecondsLeft, currentAssessment, quizStepIndex, handleNext]);

  const handleSubmit = async () => {
    const payload = {
      seamanCode,
      email,
      assessmentTypeId,
      answers: Object.values(answers),
    };

    try {
      await submitMutation.mutateAsync(payload);
      clearStoredData(); // Clear saved progress after successful submission
      setCurrentStep(totalQuizSteps); // Move to completion
      window.scrollTo(0, 0);
    } catch {
      toast.error("Failed to submit quiz. Please try again.");
    }
  };

  if (isLoading || isStatusLoading) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  if (error || !quizData) {
    return (
      <div className="flex flex-col justify-center items-center min-h-screen gap-4">
        <p className="text-red-500">Failed to load quiz data.</p>
        <Button onClick={() => window.location.reload()}>Retry</Button>
      </div>
    );
  }

  // Step 0: Greetings
  if (currentStep === 0) {
    const isScheduled = statusData?.startTime && statusData?.startTime !== "";
    const isClosed = statusData ? !statusData.isOpen || !isScheduled : false;
    const closedMessage = !isScheduled
      ? "Assessment ini belum dijadwalkan. Silakan tunggu informasi lebih lanjut."
      : statusData?.openMessage || "";

    return (
      <Greetings
        onNext={handleGreetingsNext}
        assessmentTypeName={quizData.assessmentTypeName}
        isAssessmentClosed={isClosed}
        closedMessage={closedMessage}
        startTime={statusData?.startTimeFormatted}
        endTime={statusData?.endTimeFormatted}
      />
    );
  }

  // Step 1: Seafarer Code Input
  if (currentStep === 1) {
    const handleVerify = () => {
      if (!seamanCode.trim()) {
        setVerificationError("Seafarer code harus diisi");
        return;
      }

      setIsVerifying(true);
      setVerificationError("");

      // Trigger API call to check seafarer assignment
      setQuerySeafarerCode(seamanCode);
    };

    const handleSeafarerCodeChange = (value: string) => {
      setSeamanCode(value);
      if (value !== seamanCode && isVerified) {
        setIsVerified(false);
        setVerificationError("");
        setAutoFilledData({ name: "", rank: "", vessel: "" });
        setQuerySeafarerCode(""); // Reset query to stop API call
      }
    };

    return (
      <div className="min-h-screen bg-gray-50 py-8">
        <div className="max-w-4xl mx-auto px-6">
          {/* Header Section */}
          <div className="bg-white rounded-lg shadow-sm border p-8 mb-3">
            <div className="flex justify-between items-center mb-6">
              <Image
                width={64}
                height={64}
                src="/images/logo1.png"
                alt="Logo Kiri"
                className="h-10 w-auto md:h-16"
              />
              <div className="text-center">
                <h1 className="text-lg md:text-3xl font-bold uppercase text-gray-800 mb-2">
                  {quizData.assessmentTypeName}
                </h1>
              </div>
              <Image
                width={64}
                height={64}
                src="/images/logo2.png"
                alt="Logo Kanan"
                className="h-10 w-auto md:h-16"
              />
            </div>
          </div>

          {/* Seafarer Code Input */}
          <div className="bg-white rounded-lg shadow-sm border p-8">
            <h2 className="font-bold text-xl mb-6 text-gray-800 border-b pb-3">Identitas Diri</h2>

            <div className="space-y-6">
              <div className="space-y-2">
                <Label htmlFor="seafarerCode" className="font-medium text-gray-700">
                  Seafarer Code <span className="text-red-500">*</span>
                </Label>
                <div className="flex gap-2">
                  <Input
                    id="seafarerCode"
                    placeholder="Masukkan seafarer code"
                    value={seamanCode}
                    onChange={(e) => handleSeafarerCodeChange(e.target.value)}
                    className="border-gray-300 focus:border-gray-500 focus:ring-gray-500"
                  />
                  <Button
                    type="button"
                    onClick={handleVerify}
                    disabled={isVerifying || !seamanCode.trim()}
                    className="px-4 py-2 bg-gray-800 hover:bg-gray-700 text-white font-medium disabled:bg-gray-400"
                  >
                    {isVerifying ? "Verifikasi..." : "Verifikasi"}
                  </Button>
                </div>
                {verificationError && (
                  <p className="text-sm text-red-600 mt-1">{verificationError}</p>
                )}
                {isVerified && (
                  <p className="text-sm text-green-600 mt-1">
                    ✓ Seafarer code berhasil diverifikasi
                  </p>
                )}
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2 md:col-span-2">
                  <Label htmlFor="nama" className="font-medium text-gray-700">
                    Nama Lengkap <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="nama"
                    placeholder="Nama akan terisi otomatis setelah verifikasi seafarer code"
                    value={autoFilledData.name}
                    readOnly
                    className="border-gray-300 bg-gray-50 text-gray-700"
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="rank" className="font-medium text-gray-700">
                    Rank <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="rank"
                    placeholder="Rank akan terisi otomatis setelah verifikasi seafarer code"
                    value={autoFilledData.rank}
                    readOnly
                    className="border-gray-300 bg-gray-50 text-gray-700"
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="vessel" className="font-medium text-gray-700">
                    Nama Vessel / Nama Akademi Pelayaran <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="vessel"
                    placeholder="Nama vessel akan terisi otomatis setelah verifikasi seafarer code"
                    value={autoFilledData.vessel}
                    readOnly
                    className="border-gray-300 bg-gray-50 text-gray-700"
                  />
                </div>
              </div>

              <div className="flex justify-between pt-4">
                <Button
                  type="button"
                  onClick={() => setCurrentStep(0)}
                  variant="outline"
                  className="px-8 py-2 border-gray-300 text-gray-700 hover:bg-gray-50"
                >
                  Kembali
                </Button>
                <Button
                  onClick={handleStartQuiz}
                  disabled={!isVerified}
                  className={`px-8 py-2 text-white font-medium ${
                    isVerified
                      ? "bg-green-600 hover:bg-green-700"
                      : "bg-gray-400 cursor-not-allowed"
                  }`}
                >
                  Mulai Test
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Completion Step
  if (currentStep === totalQuizSteps) {
    return (
      <Completion
        assessmentTypeName={quizData.assessmentTypeName}
        seamanCode={seamanCode}
        email={email}
      />
    );
  }

  // Quiz Sections (Step 2+)
  const currentQuestion = currentAssessment?.questions[currentQuestionIndex];
  const userAnswer = currentQuestion ? answers[currentQuestion.questionId] : undefined;
  const totalQuestions = currentAssessment?.questions.length || 0;

  // Show tutorial for this section if it has content and hasn't been dismissed
  if (currentAssessment?.tutorialContent && !tutorialDismissed[quizStepIndex]) {
    return (
      <TutorialDisplay
        assessmentName={currentAssessment.assessmentName}
        content={currentAssessment.tutorialContent}
        timerMinutes={currentAssessment.tutorialTimerMinutes ?? undefined}
        onProceed={() => setTutorialDismissed((prev) => ({ ...prev, [quizStepIndex]: true }))}
      />
    );
  }
  const answeredCount =
    currentAssessment?.questions.filter((q) => answers[q.questionId] !== undefined).length || 0;

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      {/* Floating Timer - sama seperti value-assessment */}
      {currentAssessment?.usingTimer && timerSecondsLeft !== null && (
        <div className="fixed top-4 right-4 z-50 p-4 rounded-lg border-2 shadow-lg bg-blue-50 border-blue-200">
          <div className="mb-2 text-xs font-medium text-gray-600">
            {currentAssessment.assessmentName}
            <span className="ml-2 text-gray-500">
              (Durasi: {currentAssessment.timerLimitMinutes} menit)
            </span>
          </div>
          <div className="flex items-center space-x-3">
            <div className="flex items-center space-x-2">
              <div
                className={`w-3 h-3 rounded-full ${
                  timerSecondsLeft > 0 ? "bg-blue-500 animate-pulse" : "bg-gray-400"
                }`}
              />
              <span className="text-sm font-medium text-gray-700">
                {timerSecondsLeft > 0 ? "Timer Aktif" : "Timer Selesai"}
              </span>
            </div>
            <div className="text-2xl font-mono font-bold text-blue-600">
              {formatTimerTime(timerSecondsLeft)}
            </div>
          </div>
          {timerSecondsLeft <= 300 && timerSecondsLeft > 0 && (
            <div className="mt-2 text-xs text-blue-800 font-medium animate-pulse">
              ⚠️ Waktu hampir habis!
            </div>
          )}
        </div>
      )}

      <div className="max-w-6xl mx-auto px-6">
        {/* Header Section */}
        <div className="bg-white rounded-lg shadow-sm border p-8 mb-3">
          <div className="flex justify-between items-center mb-6">
            <Image
              width={64}
              height={64}
              src="/images/logo1.png"
              alt="Logo Kiri"
              className="h-10 w-auto md:h-16"
            />
            <div className="text-center">
              <h1 className="text-lg md:text-3xl font-bold uppercase text-gray-800 mb-2">
                {quizData.assessmentTypeName}
              </h1>
              <p className="text-sm text-gray-500">
                Section {quizStepIndex + 1} of {quizData.assessments.length}:{" "}
                {currentAssessment?.assessmentName}
              </p>
            </div>
            <Image
              width={64}
              height={64}
              src="/images/logo2.png"
              alt="Logo Kanan"
              className="h-10 w-auto md:h-16"
            />
          </div>
        </div>

        {/* Instructions */}
        <div className="bg-white rounded-lg shadow-sm border p-8 mb-3">
          <h2 className="text-2xl font-bold text-gray-800 mb-6">Panduan Pengisian:</h2>
          <div className="space-y-3 text-gray-700">
            <p>1. Bacalah dengan cermat setiap pertanyaan yang disajikan.</p>
            <p>2. Pilih jawaban yang paling sesuai untuk setiap pertanyaan.</p>
            <p>3. Anda dapat navigasi antar soal menggunakan tombol navigasi atau sidebar.</p>
            <p>4. Kerjakan asesmen ini secara mandiri tanpa berdiskusi dengan orang lain.</p>
            {currentAssessment?.usingTimer && (
              <p>
                5. <strong>Waktu pengerjaan: {currentAssessment.timerLimitMinutes} Menit</strong>
              </p>
            )}
          </div>
        </div>

        <div className="flex flex-col-reverse md:flex-row gap-6">
          {/* Question Navigation Sidebar */}
          <div className="w-full md:w-64 bg-white rounded-lg shadow-sm border p-6">
            <h3 className="font-bold text-lg mb-4 text-gray-800">Navigasi Soal</h3>

            {/* Progress Summary */}
            <div className="mb-6 p-4 bg-gray-50 rounded-lg">
              <div className="text-sm text-gray-600 mb-2">Progress:</div>
              <div className="text-lg font-bold text-gray-800">
                {answeredCount} / {totalQuestions}
              </div>
              <div className="text-sm text-gray-500">soal terjawab</div>
            </div>

            {/* Question Numbers Grid */}
            <div className="grid grid-cols-5 gap-2">
              {currentAssessment?.questions.map((q, index) => {
                const isAnswered = answers[q.questionId] !== undefined;
                const isCurrent = index === currentQuestionIndex;

                return (
                  <button
                    key={index}
                    onClick={() => jumpToQuestion(index)}
                    className={`
                      w-10 h-10 rounded-lg border-2 font-medium text-sm transition-all
                      ${
                        isCurrent
                          ? "bg-blue-500 text-white border-blue-500"
                          : isAnswered
                            ? "bg-gray-800 text-white border-gray-800"
                            : "bg-white text-gray-600 border-gray-300 hover:border-gray-400"
                      }
                    `}
                  >
                    {index + 1}
                  </button>
                );
              })}
            </div>

            {/* Legend */}
            <div className="mt-4 space-y-2 text-sm">
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 bg-blue-500 rounded"></div>
                <span className="text-gray-600">Soal saat ini</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 bg-gray-800 rounded"></div>
                <span className="text-gray-600">Sudah dijawab</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 bg-white border-2 border-gray-300 rounded"></div>
                <span className="text-gray-600">Belum dijawab</span>
              </div>
            </div>
          </div>

          {/* Main Question Area */}
          <div className="flex-1">
            <div className="bg-white rounded-lg shadow-sm border p-8 w-full">
              <div className="mb-6">
                <div className="flex justify-between items-center mb-4">
                  <h3 className="text-lg font-bold text-gray-800">
                    Soal {currentQuestionIndex + 1} dari {totalQuestions}
                  </h3>
                </div>

                <div className="text-gray-700 text-lg leading-relaxed mb-6">
                  {isValidImageUrl(currentQuestion?.imageUrl) && (
                    <div className="mb-4">
                      <Image
                        src={resolveImageUrl(currentQuestion?.imageUrl as string)}
                        unoptimized
                        width={400}
                        height={300}
                        alt={`Gambar soal ${currentQuestionIndex + 1}`}
                        className="rounded-lg border bg-gray-100 object-contain max-h-[300px]"
                      />
                    </div>
                  )}
                  <div>{currentQuestion?.questionText}</div>
                  {currentQuestion?.questionType === "multiple_choice" && (
                    <span className="ml-2 text-xs text-gray-400 font-normal">
                      (Multiple Choice)
                    </span>
                  )}
                  {currentQuestion?.questionType === "match_choice" && (
                    <span className="ml-2 text-xs text-gray-400 font-normal">
                      (Select 2 Matches)
                    </span>
                  )}
                </div>

                {/* Render answer options based on question type */}
                {currentQuestion?.questionType === "single_choice" && (
                  <RadioGroup
                    value={userAnswer?.selectedOptions?.[0]?.toString() || ""}
                    onValueChange={(val) =>
                      handleAnswerChange(currentQuestion.questionId, val, "single_choice")
                    }
                  >
                    <div className="space-y-4">
                      {currentQuestion.options.map((option) => (
                        <div
                          key={option.optionId}
                          className="flex items-center space-x-3 p-4 rounded-lg border hover:bg-gray-50 cursor-pointer"
                          onClick={() =>
                            handleAnswerChange(
                              currentQuestion.questionId,
                              option.optionId.toString(),
                              "single_choice"
                            )
                          }
                        >
                          <RadioGroupItem
                            value={option.optionId.toString()}
                            id={`option-${option.optionId}`}
                            className="mt-1"
                            onClick={(e) => e.stopPropagation()}
                          />
                          <Label
                            htmlFor={`option-${option.optionId}`}
                            className="flex-1 cursor-pointer text-gray-700 leading-relaxed"
                            onClick={(e) => e.stopPropagation()}
                          >
                            <div className="flex flex-col gap-2">
                              {isValidImageUrl(option.imageUrl) && (
                                <Image
                                  src={resolveImageUrl(option.imageUrl as string)}
                                  unoptimized
                                  width={200}
                                  height={150}
                                  alt={`Opsi ${option.optionText}`}
                                  className="rounded border bg-white object-contain max-h-[150px]"
                                />
                              )}
                              <span>{option.optionText}</span>
                            </div>
                          </Label>
                        </div>
                      ))}
                    </div>
                  </RadioGroup>
                )}

                {(currentQuestion?.questionType === "multiple_choice" ||
                  currentQuestion?.questionType === "match_choice") && (
                  <div className="space-y-4">
                    {currentQuestion.options.map((option) => {
                      const isChecked =
                        userAnswer?.selectedOptions?.includes(option.optionId) || false;
                      return (
                        <div
                          key={option.optionId}
                          className="flex items-center space-x-3 p-4 rounded-lg border hover:bg-gray-50 cursor-pointer"
                          onClick={() =>
                            handleAnswerChange(
                              currentQuestion.questionId,
                              { id: option.optionId, checked: !isChecked },
                              currentQuestion.questionType
                            )
                          }
                        >
                          <Checkbox
                            checked={isChecked}
                            onCheckedChange={(checked) =>
                              handleAnswerChange(
                                currentQuestion.questionId,
                                { id: option.optionId, checked: checked as boolean },
                                currentQuestion.questionType
                              )
                            }
                            id={`option-${option.optionId}`}
                            className="mt-1"
                            onClick={(e) => e.stopPropagation()}
                          />
                          <Label
                            htmlFor={`option-${option.optionId}`}
                            className="flex-1 cursor-pointer text-gray-700 leading-relaxed"
                            onClick={(e) => e.stopPropagation()}
                          >
                            <div className="flex flex-col gap-2">
                              {isValidImageUrl(option.imageUrl) && (
                                <Image
                                  src={resolveImageUrl(option.imageUrl as string)}
                                  unoptimized
                                  width={200}
                                  height={150}
                                  alt={`Opsi ${option.optionText}`}
                                  className="rounded border bg-white object-contain max-h-[150px]"
                                />
                              )}
                              <span>{option.optionText}</span>
                            </div>
                          </Label>
                        </div>
                      );
                    })}
                  </div>
                )}

                {currentQuestion?.questionType === "short_answer" && (
                  <div className="max-w-xl">
                    <Input
                      placeholder="Type your answer here..."
                      value={userAnswer?.textAnswer || ""}
                      onChange={(e) =>
                        handleAnswerChange(
                          currentQuestion.questionId,
                          e.target.value,
                          "short_answer"
                        )
                      }
                      className="bg-white border-gray-300 focus:border-gray-500 focus:ring-gray-500"
                    />
                  </div>
                )}
              </div>

              {/* Navigation Buttons */}
              <div className="flex flex-col-reverse md:flex-row gap-3 justify-between pt-6 border-t w-full">
                <Button
                  onClick={currentQuestionIndex === 0 ? handlePrev : handleQuestionPrev}
                  variant="outline"
                  className="px-6 py-2 cursor-pointer"
                  disabled={submitMutation.isPending}
                >
                  {currentQuestionIndex === 0
                    ? currentStep === 2
                      ? "Kembali ke Identitas"
                      : "Kembali ke Section Sebelumnya"
                    : "Soal Sebelumnya"}
                </Button>

                <div className="flex gap-3">
                  {currentQuestionIndex < totalQuestions - 1 ? (
                    <Button
                      onClick={handleQuestionNext}
                      className="px-6 py-2 bg-gray-800 hover:bg-gray-700 w-full cursor-pointer"
                      disabled={submitMutation.isPending}
                    >
                      Soal Berikutnya
                    </Button>
                  ) : currentStep === totalQuizSteps - 1 ? (
                    <Button
                      onClick={handleNext}
                      disabled={submitMutation.isPending}
                      className="px-6 py-2 bg-green-600 hover:bg-green-700 cursor-pointer"
                    >
                      {submitMutation.isPending ? "Menyimpan..." : "Selesai"}
                    </Button>
                  ) : (
                    <Button
                      onClick={handleNext}
                      disabled={submitMutation.isPending}
                      className="px-6 py-2 bg-gray-800 hover:bg-gray-700 cursor-pointer"
                    >
                      Lanjut ke Section Berikutnya
                    </Button>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
