"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { ValueAssessmentData } from "./page";
import { useGetAssessmentByRole, usePostAssessmentResults } from "./_hooks/useAssessment";
import { TutorialDisplay } from "@/components/tutorial-display";
import Image from "next/image";
import { useCountdown } from "@/hooks/use-session-storage";
import { useTimerPauseResume } from "@/hooks/useTimerPauseResume";
import { useStorageCountdown } from "@/hooks/use-local-storage";
import { isValidImageUrl, resolveImageUrl } from "@/app/lib/image";

interface Section3Props {
  onNext: () => void;
  onBack: () => void;
  assessmentData: ValueAssessmentData;
  updateAssessmentData: (data: Partial<ValueAssessmentData>) => void;
}

const LIKERT_SCALE = [
  { value: "1", label: "1 - Sangat Tidak Menggambarkan Diri Saya" },
  { value: "2", label: "2 - Tidak Menggambarkan Diri Saya" },
  { value: "3", label: "3 - Menggambarkan Diri Saya" },
  { value: "4", label: "4 - Sangat Menggambarkan Diri Saya" },
];

export default function Section3({
  onNext,
  onBack,
  assessmentData,
  updateAssessmentData,
}: Section3Props) {
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [answers, setAnswers] = useState<{ [questionId: number]: number }>(
    assessmentData.section3Answers ?? {}
  );
  const timeOutRef = useRef(false);
  const [tutorialDismissed, setTutorialDismissed] = useState(
    assessmentData.section3TutorialDismissed ?? false
  );

  // Fetch assessment data using the hook
  const { data: assessment, isLoading, error } = useGetAssessmentByRole("va_3");

  // POST mutation hook
  const { mutate, isPending, isSuccess } = usePostAssessmentResults(onNext);

  // Timer dari assessment data
  const timerMinutes = assessment?.timerLimitMinutes ?? 30;

  const { timeLeft } = useCountdown(
    assessmentData.section3StartTime,
    timerMinutes,
    assessmentData.section3PauseTimestamp
  );

  // Timer sync is now handled by the main page component
  // This section only needs to store its timer duration

  // Store timer minutes in assessmentData for progress bar to use
  useEffect(() => {
    if (timerMinutes && assessmentData.section3TimerMinutes !== timerMinutes) {
      updateAssessmentData({ section3TimerMinutes: timerMinutes });
    }
  }, [timerMinutes, assessmentData.section3TimerMinutes, updateAssessmentData]);

  // Store usingTimer in assessmentData
  useEffect(() => {
    if (
      assessment?.usingTimer !== undefined &&
      assessmentData.usingTimer !== assessment.usingTimer
    ) {
      updateAssessmentData({ usingTimer: assessment.usingTimer });
    }
  }, [assessment?.usingTimer, assessmentData.usingTimer, updateAssessmentData]);

  // Set section start time when assessment loads and there is no tutorial to read
  useEffect(() => {
    if (assessment && !assessment.tutorialContent && !assessmentData.section3StartTime) {
      updateAssessmentData({ section3StartTime: new Date().toISOString() });
    }
  }, [assessment, assessmentData.section3StartTime, updateAssessmentData]);

  // Stable pause/resume callbacks to prevent hook re-setup
  const handlePause = useCallback(() => {
    const now = new Date().toISOString();
    console.log(`⏸️ Section 3 paused`);
    updateAssessmentData({ section3PauseTimestamp: now });
  }, [updateAssessmentData]);

  const handleResume = useCallback(() => {
    console.log(`▶️ Section 3 resumed`);
    updateAssessmentData({ section3PauseTimestamp: undefined });
  }, [updateAssessmentData]);

  // Handle pause/resume when tab loses focus or visibility changes
  useTimerPauseResume(
    assessmentData.currentStep === 5, // isActive - true when user is on section 3
    handlePause,
    handleResume
  );

  // Overall assessment timer (24 hours)
  const { isExpired: isOverallExpired } = useStorageCountdown("valueAssessmentFormData");

  const currentQuestion = assessment?.questions?.[currentQuestionIndex];

  const handleAnswerChange = (questionId: number, value: string) => {
    console.log("Debug:", questionId, value);
    const newAnswers = {
      ...answers,
      [questionId]: parseInt(value, 10),
    };
    setAnswers(newAnswers);

    updateAssessmentData({ section3Answers: newAnswers });
  };

  const handleNext = () => {
    if (currentQuestionIndex < (assessment?.questions?.length ?? 0) - 1) {
      setCurrentQuestionIndex(currentQuestionIndex + 1);
    }
  };

  const handlePrevious = () => {
    if (currentQuestionIndex > 0) {
      setCurrentQuestionIndex(currentQuestionIndex - 1);
    }
  };

  const jumpToQuestion = (index: number) => {
    setCurrentQuestionIndex(index);
  };

  const handleSubmit = useCallback(() => {
    if (!assessment?.questions) {
      toast.error("Soal belum dimuat");
      return;
    }

    // Check for unanswered questions
    const unansweredQuestions = assessment.questions.filter((q) => !(q.questionId in answers));

    if (unansweredQuestions.length > 0 && timeLeft > 0) {
      toast.error(`Masih ada ${unansweredQuestions.length} soal yang belum dijawab`);
      return;
    }

    // Filter out undefined values and ensure all values are numbers
    const filteredAnswers: { [questionId: number]: number } = {};
    Object.entries(answers).forEach(([questionId, optionId]) => {
      const numQuestionId = parseInt(questionId);
      if (optionId !== undefined && !isNaN(numQuestionId) && !isNaN(optionId)) {
        filteredAnswers[numQuestionId] = optionId;
      }
    });

    // Submit answers using mutation
    mutate({
      seafarerCode: assessmentData.seafarerCode,
      role: "va_3",
      answers: filteredAnswers,
    });

    if (isSuccess) {
      // Remove localStorage when assessment submission is successful
      localStorage.removeItem("valueAssessmentFormData");
    }
  }, [assessment?.questions, answers, assessmentData.seafarerCode, mutate, timeLeft, isSuccess]);

  // Auto-submit ketika waktu section habis (timeLeft === 0)
  useEffect(() => {
    if (
      timeLeft === 0 &&
      !timeOutRef.current &&
      assessment?.questions &&
      assessment.questions.length > 0
    ) {
      timeOutRef.current = true;
      console.log("Section 3 timer habis, auto-submit...");
      handleSubmit();
    }
  }, [timeLeft, handleSubmit, assessment?.questions]);

  // Auto-submit ketika assessment expires (24 jam)
  useEffect(() => {
    if (isOverallExpired && !timeOutRef.current && assessment?.questions) {
      timeOutRef.current = true;
      toast.warning("Waktu assessment telah habis. Form akan di-submit otomatis.");
      handleSubmit();
    }
  }, [isOverallExpired, handleSubmit, assessment?.questions]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-800 mx-auto mb-4"></div>
          <p className="text-gray-600">Memuat soal...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <p className="text-gray-600">Gagal memuat soal</p>
          <Button onClick={onBack} className="mt-4">
            Kembali
          </Button>
        </div>
      </div>
    );
  }

  if (!assessment?.questions || assessment.questions.length === 0) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <p className="text-gray-600">Tidak ada soal yang tersedia untuk Section 3</p>
          <Button onClick={onBack} className="mt-4">
            Kembali
          </Button>
        </div>
      </div>
    );
  }

  if (assessment?.tutorialContent && !tutorialDismissed) {
    return (
      <TutorialDisplay
        assessmentName={assessment.assessmentName}
        content={assessment.tutorialContent}
        timerMinutes={assessment.tutorialTimerMinutes ?? undefined}
        onProceed={() => {
          setTutorialDismissed(true);
          updateAssessmentData({
            section3TutorialDismissed: true,
            ...(!assessmentData.section3StartTime && {
              section3StartTime: new Date().toISOString(),
            }),
          });
        }}
      />
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
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
                Value Assessment Section 3
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

        {/* Instructions */}
        <div className="bg-white rounded-lg shadow-sm border p-8 mb-3">
          <h2 className="text-2xl font-bold text-gray-800 mb-6">Panduan Pengisian:</h2>
          <div className="space-y-3 text-gray-700 mb-6">
            <p>
              Pada bagian ini, terdapat sejumlah pernyataan mengenai diri Anda. Bacalah dengan
              seksama setiap pernyataan yang ada. Jawablah setiap pernyataan sesuai dengan diri
              Anda. Tidak ada jawaban benar dan salah dalam asesmen ini.
            </p>
          </div>

          <div className="bg-blue-50 border-l-4 border-blue-400 p-4">
            <h3 className="font-bold text-blue-800 mb-3">
              Setiap pernyataan terdiri skala 1 hingga 4 dengan keterangan sebagai berikut:
            </h3>
            <div className="space-y-2 text-blue-700">
              {LIKERT_SCALE.map((scale) => (
                <p key={scale.value}>
                  <strong>{scale.label}</strong>
                </p>
              ))}
            </div>
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
                {Object.keys(answers).length} / {assessment?.questions?.length ?? 0}
              </div>
              <div className="text-sm text-gray-500">soal terjawab</div>
            </div>

            {/* Question Numbers Grid */}
            <div className="grid grid-cols-5 gap-2">
              {assessment?.questions?.map((q, index) => {
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
            <div className="bg-white rounded-lg shadow-sm border p-8">
              <div className="mb-6">
                <div className="flex justify-between items-center mb-6">
                  <h3 className="text-lg font-bold text-gray-800">
                    Pernyataan {currentQuestionIndex + 1} dari {assessment?.questions?.length ?? 0}
                  </h3>
                </div>

                {/* Question Statement */}
                <div className="mb-8 p-6 bg-gray-50 rounded-lg border-l-4 border-blue-400 flex flex-col gap-2">
                  {currentQuestion?.isImage === 1 && isValidImageUrl(currentQuestion?.imageUrl) && (
                    <Image
                      src={resolveImageUrl(currentQuestion?.imageUrl as string)}
                      unoptimized
                      width={300}
                      height={200}
                      alt={"Gambar " + currentQuestionIndex}
                      className="h-auto max-w-full rounded object-contain"
                    />
                  )}
                  <p className="text-lg text-gray-800 leading-relaxed">
                    {currentQuestion?.questionText}
                  </p>
                </div>

                {/* Likert Scale Options */}
                <RadioGroup
                  value={
                    currentQuestion?.questionId
                      ? answers[currentQuestion.questionId]?.toString() || ""
                      : ""
                  }
                  onValueChange={(value) =>
                    currentQuestion && handleAnswerChange(currentQuestion.questionId, value)
                  }
                >
                  <div className="space-y-4">
                    {currentQuestion?.options.map((scale) => (
                      <div
                        key={scale.optionId}
                        className="flex items-center space-x-4 p-4 rounded-lg border-2 hover:bg-gray-50 transition-colors cursor-pointer"
                        onClick={() => {
                          handleAnswerChange(
                            currentQuestion?.questionId || 0,
                            scale.optionId.toString()
                          );
                        }}
                      >
                        <RadioGroupItem
                          value={scale.optionId.toString()}
                          id={`scale-${scale.optionId}`}
                        />
                        <Label
                          htmlFor={`scale-${scale.optionId}`}
                          className="flex-1 cursor-pointer text-gray-700 text-lg"
                        >
                          {scale.optionText}
                        </Label>
                      </div>
                    ))}
                  </div>
                </RadioGroup>
              </div>

              {/* Navigation Buttons */}
              <div className="flex flex-col-reverse md:flex-row gap-3 justify-between pt-6 border-t">
                <Button
                  onClick={handlePrevious}
                  variant="outline"
                  className="px-6 py-2 cursor-pointer"
                  disabled={currentQuestionIndex === 0}
                >
                  Soal Sebelumnya
                </Button>

                <div className="flex gap-3">
                  {currentQuestionIndex < (assessment?.questions?.length ?? 0) - 1 ? (
                    <Button
                      onClick={handleNext}
                      className="px-6 py-2 bg-gray-800 hover:bg-gray-700 cursor-pointer"
                    >
                      Soal Berikutnya
                    </Button>
                  ) : (
                    <Button
                      onClick={handleSubmit}
                      disabled={isPending}
                      className="px-6 py-2 bg-green-600 hover:bg-green-700 cursor-pointer"
                    >
                      {isPending ? "Menyelesaikan..." : "Selesaikan Assessment"}
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
