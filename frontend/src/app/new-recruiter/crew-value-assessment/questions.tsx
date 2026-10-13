"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { useCountdown } from "@/hooks/use-session-storage";
import { useTimerPauseResume } from "@/hooks/useTimerPauseResume";
import { CESAssessmentData } from "./types";
import Image from "next/image";
import { useGetAssessmentByRole, usePostCESResults } from "./_hooks/useAssessment";
import { isValidImageUrl, resolveImageUrl } from "../../lib/image";
import { TutorialDisplay } from "@/components/tutorial-display";

interface QuestionsProps {
  onNext: () => void;
  onBack: () => void;
  assessmentData: CESAssessmentData;
  updateAssessmentData: (data: Partial<CESAssessmentData>) => void;
  role: string;
}

export default function Questions({
  onNext,
  onBack,
  assessmentData,
  updateAssessmentData,
  role,
}: QuestionsProps) {
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [answers, setAnswers] = useState<{ [questionId: number]: number }>(
    assessmentData.answers ?? {}
  );
  const [tutorialDismissed, setTutorialDismissed] = useState(
    assessmentData.tutorialDismissed ?? false
  );

  const formatRoleName = (roleSlug: string) => {
    return roleSlug
      .split("_")
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(" ");
  };

  // Fetch assessment data using the hook with dynamic role
  const { data: assessment, isLoading, error } = useGetAssessmentByRole(role);

  // POST mutation hook
  const { mutate, isPending } = usePostCESResults(onNext);

  // Use countdown hook with timer from fetched assessment data
  const timerMinutes = assessment?.timerLimitMinutes ?? 60;
  const { timeLeft } = useCountdown(assessmentData.assessmentStartTime, timerMinutes, undefined);

  // Store timer minutes and usingTimer in assessmentData
  useEffect(() => {
    if (timerMinutes && assessmentData.timerMinutes !== timerMinutes) {
      updateAssessmentData({ timerMinutes: timerMinutes });
    }
  }, [timerMinutes, assessmentData.timerMinutes, updateAssessmentData]);

  useEffect(() => {
    if (
      assessment?.usingTimer !== undefined &&
      assessmentData.usingTimer !== assessment.usingTimer
    ) {
      updateAssessmentData({ usingTimer: assessment.usingTimer });
    }
  }, [assessment?.usingTimer, assessmentData.usingTimer, updateAssessmentData]);

  // Set assessment start time when assessment loads and there is no tutorial to read
  useEffect(() => {
    if (assessment && !assessment.tutorialContent && !assessmentData.assessmentStartTime) {
      updateAssessmentData({ assessmentStartTime: new Date().toISOString() });
    }
  }, [assessment, assessmentData.assessmentStartTime, updateAssessmentData]);

  // Stable pause/resume callbacks
  const handlePause = useCallback(() => {
    const now = new Date().toISOString();
    console.log(`[CES Questions] PAUSED at: ${now}`);
    updateAssessmentData({ pauseTimestamp: now });
  }, [updateAssessmentData]);

  const handleResume = useCallback(() => {
    const resumeTime = new Date().toISOString();
    console.log(`[CES Questions] RESUMED at: ${resumeTime}`);
    updateAssessmentData({ pauseTimestamp: undefined });
  }, [updateAssessmentData]);

  // Setup pause/resume detection
  useTimerPauseResume(assessmentData.currentStep === 3, handlePause, handleResume);

  const handleSubmit = useCallback(() => {
    if (!assessment?.questions) {
      toast.error("Data soal tidak tersedia");
      return;
    }

    // Check if all questions are answered
    const unansweredQuestions = assessment.questions.filter(
      (q) => answers[q.questionId] === undefined
    );
    if (unansweredQuestions.length > 0 && timeLeft > 0) {
      toast.error(`Masih ada ${unansweredQuestions.length} soal yang belum dijawab`);
      return;
    }

    // Filter out undefined values
    const filteredAnswers: { [questionId: number]: number } = {};
    Object.entries(answers).forEach(([questionId, optionId]) => {
      const numQuestionId = parseInt(questionId);
      if (optionId !== undefined && !isNaN(numQuestionId) && !isNaN(optionId)) {
        filteredAnswers[numQuestionId] = optionId;
      }
    });

    // Submit answers using mutation
    mutate({
      token: assessmentData.token,
      role: role,
      answers: filteredAnswers,
    });
  }, [assessment?.questions, answers, assessmentData.token, mutate, timeLeft, role]);

  // Auto submit when time runs out
  const timeOutRef = useRef(false);

  useEffect(() => {
    if (
      timeLeft === 0 &&
      !timeOutRef.current &&
      assessment?.questions &&
      assessment.questions.length > 0
    ) {
      timeOutRef.current = true;
      toast.warning("Waktu habis! Assessment akan di-submit otomatis.");
      handleSubmit();
    }
  }, [timeLeft, handleSubmit, assessment?.questions]);

  const currentQuestion = assessment?.questions
    ? assessment.questions[currentQuestionIndex]
    : undefined;
  const currentOptions = currentQuestion?.options ?? [];

  const handleAnswerChange = (optionId: number) => {
    if (!currentQuestion?.questionId) {
      return;
    }

    if (isNaN(optionId)) {
      return;
    }

    const newAnswers = {
      ...answers,
      [currentQuestion.questionId]: optionId,
    };
    setAnswers(newAnswers);
    updateAssessmentData({ answers: newAnswers });
  };

  const handleNext = () => {
    if (assessment?.questions && currentQuestionIndex < assessment.questions.length - 1) {
      setCurrentQuestionIndex((prev) => prev + 1);
    }
  };

  const handlePrevious = () => {
    if (currentQuestionIndex > 0) {
      setCurrentQuestionIndex((prev) => prev - 1);
    }
  };

  const jumpToQuestion = (index: number) => {
    setCurrentQuestionIndex(index);
  };

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
          <p className="text-red-600">Gagal memuat data assessment: {error.message}</p>
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
          <p className="text-gray-600">
            Tidak ada soal yang tersedia untuk role {formatRoleName(role)}
          </p>
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
            tutorialDismissed: true,
            ...(!assessmentData.assessmentStartTime && {
              assessmentStartTime: new Date().toISOString(),
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
                Crew Evaluation System
              </h1>
              <p className="text-sm md:text-lg text-gray-600">{formatRoleName(role)}</p>
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
            <p>1. Bacalah dengan cermat setiap soal yang disajikan.</p>
            <p>2. Pilih satu jawaban yang paling sesuai dengan pengetahuan dan pengalaman Anda.</p>
            <p>3. Kerjakan asesmen ini secara mandiri tanpa berdiskusi dengan orang lain.</p>
            <p>
              4. <strong>Waktu pengerjaan: {assessment?.timerLimitMinutes || 60} Menit</strong>
            </p>
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
            <div className="bg-white rounded-lg shadow-sm border p-8 w-full">
              <div className="mb-6">
                <div className="flex justify-between items-center mb-4">
                  <h3 className="text-lg font-bold text-gray-800">
                    Soal {currentQuestionIndex + 1} dari {assessment?.questions?.length ?? 0}
                  </h3>
                </div>

                <div className="text-gray-700 text-lg leading-relaxed mb-6 flex flex-col gap-2">
                  {currentQuestion?.isImage === 1 && isValidImageUrl(currentQuestion?.imageUrl) && (
                    <Image
                      src={resolveImageUrl(currentQuestion.imageUrl as string)}
                      width={300}
                      height={200}
                      unoptimized
                      className="h-auto max-w-full rounded object-contain"
                      alt={"Gambar " + currentQuestionIndex}
                    />
                  )}
                  {currentQuestion?.questionText}
                </div>

                <RadioGroup
                  key={`question-${currentQuestion?.questionId}`}
                  value={
                    currentQuestion?.questionId !== undefined &&
                    answers[currentQuestion.questionId] !== undefined
                      ? String(answers[currentQuestion.questionId])
                      : undefined
                  }
                  onValueChange={(value) => {
                    const optionId = parseInt(value, 10);
                    if (!isNaN(optionId)) {
                      handleAnswerChange(optionId);
                    }
                  }}
                >
                  <div className="space-y-4">
                    {currentOptions.map((option) => (
                      <div
                        key={option.optionId}
                        className="flex items-center space-x-3 p-4 rounded-lg border hover:bg-gray-50 cursor-pointer"
                        onClick={() => {
                          handleAnswerChange(option.optionId);
                        }}
                      >
                        <RadioGroupItem
                          value={option.optionId.toString()}
                          id={`option-${option.optionId}`}
                          className="mt-1"
                        />
                        <Label
                          htmlFor={`option-${option.optionId}`}
                          className="flex-1 cursor-pointer text-gray-700 leading-relaxed"
                        >
                          <span className="font-medium">{option.optionLetter}.</span>{" "}
                          <div className="flex flex-col gap-2">
                            {isValidImageUrl(option.imageUrl) && (
                              <Image
                                src={resolveImageUrl(option.imageUrl as string)}
                                width={300}
                                height={200}
                                unoptimized
                                className="h-auto max-w-full rounded object-contain"
                                alt={"Gambar " + option.optionId}
                              />
                            )}
                            {option.optionText}
                          </div>
                        </Label>
                      </div>
                    ))}
                  </div>
                </RadioGroup>
              </div>

              {/* Navigation Buttons */}
              <div className="flex flex-col-reverse md:flex-row gap-3 justify-between pt-6 border-t w-full">
                <Button
                  onClick={currentQuestionIndex === 0 ? onBack : handlePrevious}
                  variant="outline"
                  className="px-6 py-2 cursor-pointer"
                >
                  {currentQuestionIndex === 0 ? "Kembali ke Identitas" : "Soal Sebelumnya"}
                </Button>

                <div className="flex gap-3">
                  {currentQuestionIndex < (assessment?.questions?.length ?? 1) - 1 ? (
                    <Button
                      onClick={handleNext}
                      className="px-6 py-2 bg-gray-800 hover:bg-gray-700 w-full cursor-pointer"
                    >
                      Soal Berikutnya
                    </Button>
                  ) : (
                    <Button
                      onClick={handleSubmit}
                      disabled={isPending}
                      className="px-6 py-2 bg-green-600 hover:bg-green-700 cursor-pointer"
                    >
                      {isPending ? "Menyimpan..." : "Selesai"}
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
