"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { useCountdown } from "@/hooks/use-session-storage";
import { useStorageCountdown } from "@/hooks/use-local-storage";
import { useTimerPauseResume } from "@/hooks/useTimerPauseResume";
import { ValueAssessmentData } from "./page";
import Image from "next/image";
import { useGetAssessmentByRole, usePostAssessmentResults } from "./_hooks/useAssessment";
import { isValidImageUrl, resolveImageUrl } from "@/app/lib/image";
import { TutorialDisplay } from "@/components/tutorial-display";

interface Section1Props {
  onNext: () => void;
  onBack: () => void;
  assessmentData: ValueAssessmentData;
  updateAssessmentData: (data: Partial<ValueAssessmentData>) => void;
}

export default function Section1({
  onNext,
  onBack,
  assessmentData,
  updateAssessmentData,
}: Section1Props) {
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [answers, setAnswers] = useState<{ [questionId: number]: number }>(
    assessmentData.section1Answers ?? {}
  );
  const storageTimeOutRef = useRef(false);
  const [tutorialDismissed, setTutorialDismissed] = useState(
    assessmentData.section1TutorialDismissed ?? false
  );

  // Debug: Log currentStep from assessmentData
  useEffect(() => {
    console.log(
      `[Section1] Mounted - currentStep: ${
        assessmentData.currentStep
      }, isActive: ${assessmentData.currentStep === 3}`
    );
  }, [assessmentData.currentStep]);

  // Fetch assessment data using the hook
  const { data: assessment, isLoading, error } = useGetAssessmentByRole("va_1");

  // POST mutation hook
  const { mutate, isPending } = usePostAssessmentResults(onNext);

  // Use countdown hook with timer from fetched assessment data (section timer)
  const timerMinutes = assessment?.timerLimitMinutes ?? 30;
  const { timeLeft } = useCountdown(
    assessmentData.section1StartTime,
    timerMinutes,
    assessmentData.section1PauseTimestamp
  );

  // Timer sync is now handled by the main page component
  // This section only needs to store its timer duration

  // Store timer minutes in assessmentData for progress bar to use
  useEffect(() => {
    if (timerMinutes && assessmentData.section1TimerMinutes !== timerMinutes) {
      updateAssessmentData({ section1TimerMinutes: timerMinutes });
    }
  }, [timerMinutes, assessmentData.section1TimerMinutes, updateAssessmentData]);

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
    if (assessment && !assessment.tutorialContent && !assessmentData.section1StartTime) {
      updateAssessmentData({ section1StartTime: new Date().toISOString() });
    }
  }, [assessment, assessmentData.section1StartTime, updateAssessmentData]);

  // Stable pause/resume callbacks to prevent hook re-setup
  const handlePause = useCallback(() => {
    const now = new Date().toISOString();
    console.log(
      `%c⏸️ [Section1] PAUSED at: ${now}`,
      "color: red; font-weight: bold; font-size: 14px;"
    );
    console.log(
      `%c⏸️ [Section1] Current timeLeft before pause: ${timeLeft}s`,
      "color: red; font-size: 12px;"
    );
    console.log(
      `%c⏸️ [Section1] Setting section1PauseTimestamp...`,
      "color: red; font-size: 12px;"
    );
    updateAssessmentData({ section1PauseTimestamp: now });
  }, [updateAssessmentData, timeLeft]);

  const handleResume = useCallback(() => {
    const resumeTime = new Date().toISOString();
    console.log(
      `%c▶️ [Section1] RESUMED at: ${resumeTime}`,
      "color: green; font-weight: bold; font-size: 14px;"
    );
    console.log(
      `%c▶️ [Section1] Clearing section1PauseTimestamp...`,
      "color: green; font-size: 12px;"
    );
    updateAssessmentData({ section1PauseTimestamp: undefined });
  }, [updateAssessmentData]);

  // Setup pause/resume detection
  useTimerPauseResume(
    assessmentData.currentStep === 3, // isActive - true when user is on section 1
    handlePause,
    handleResume
  );

  // Debug: Log whenever pauseTimestamp changes
  useEffect(() => {
    console.log(`[Section1] pauseTimestamp changed:`, assessmentData.section1PauseTimestamp);
    console.log();
  }, [assessmentData.section1PauseTimestamp]);

  // Overall assessment timer (24 hours)
  const { isExpired: isOverallExpired } = useStorageCountdown("valueAssessmentFormData");

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

    // Filter out undefined values and ensure all values are numbers
    const filteredAnswers: { [questionId: number]: number } = {};
    Object.entries(answers).forEach(([questionId, optionId]) => {
      const numQuestionId = parseInt(questionId);
      if (optionId !== undefined && !isNaN(numQuestionId) && !isNaN(optionId)) {
        filteredAnswers[numQuestionId] = optionId;
      }
    });

    console.log("Original answers:", answers);
    console.log("Filtered answers:", filteredAnswers);

    // Submit answers using mutation
    mutate({
      seafarerCode: assessmentData.seafarerCode,
      role: "va_1",
      answers: filteredAnswers,
    });
  }, [assessment?.questions, answers, assessmentData.seafarerCode, mutate, timeLeft]);

  // Auto submit when time runs out - use ref to avoid infinite loop
  const timeOutRef = useRef(false);

  useEffect(() => {
    if (
      timeLeft === 0 &&
      !timeOutRef.current &&
      assessment?.questions &&
      assessment.questions.length > 0
    ) {
      timeOutRef.current = true;
      handleSubmit();
    }
  }, [timeLeft, handleSubmit, assessment?.questions]);

  // Auto-submit when overall assessment expires (24 hours)
  useEffect(() => {
    if (
      isOverallExpired &&
      !storageTimeOutRef.current &&
      assessment?.questions &&
      assessment.questions.length > 0
    ) {
      storageTimeOutRef.current = true;
      toast.warning("Waktu assessment telah habis. Form akan di-submit otomatis.");
      handleSubmit();
    }
  }, [isOverallExpired, handleSubmit, assessment?.questions]);

  const currentQuestion = assessment?.questions
    ? assessment.questions[currentQuestionIndex]
    : undefined;
  const currentOptions = currentQuestion?.options ?? [];

  const handleAnswerChange = (optionId: number) => {
    console.log("Selected Option ID:", currentQuestion?.questionId, optionId);

    if (!currentQuestion?.questionId) {
      console.log("No current question ID, skipping");
      return;
    }

    if (isNaN(optionId)) {
      console.log("Invalid option ID (NaN), skipping");
      return;
    }

    // For public assessment, we only track the option selected, not the score
    const newAnswers = {
      ...answers,
      [currentQuestion.questionId]: optionId,
    };
    console.log("New Answers:", newAnswers);
    setAnswers(newAnswers);
    // Update parent immediately when answer changes
    updateAssessmentData({ section1Answers: newAnswers });
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
          <p className="text-gray-600">Tidak ada soal yang tersedia untuk Section 1</p>
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
            section1TutorialDismissed: true,
            ...(!assessmentData.section1StartTime && {
              section1StartTime: new Date().toISOString(),
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
                Value Assessment Section 1
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
          <div className="space-y-3 text-gray-700">
            <p>1. Bacalah dengan cermat setiap situasi yang disajikan.</p>
            <p>
              2. Pilih satu jawaban yang paling menggambarkan tindakan yang akan Anda ambil dalam
              situasi tersebut.
            </p>
            <p>
              3. Tidak ada jawaban benar atau salah, pilihlah jawaban yang paling menggambarkan diri
              Anda dan sesuai dengan kebiasaan Anda bekerja, bukan jawaban yang Anda anggap ideal.
            </p>
            <p>4. Kerjakan asesmen ini secara mandiri tanpa berdiskusi dengan orang lain.</p>
            <p>
              5. <strong>Waktu pengerjaan: 30 Menit</strong>
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
                      src={resolveImageUrl(currentQuestion?.imageUrl as string)}
                      unoptimized
                      width={300}
                      height={200}
                      alt={"Gambar " + currentQuestionIndex}
                      className="h-auto max-w-full rounded object-contain"
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
                    console.log("RadioGroup value changed:", value);
                    const optionId = parseInt(value, 10);
                    if (!isNaN(optionId)) {
                      handleAnswerChange(optionId);
                    } else {
                      console.log("Invalid value for parseInt:", value);
                    }
                  }}
                >
                  <div className="space-y-4">
                    {currentOptions.map((option) => (
                      <div
                        key={option.optionId}
                        className="flex items-center space-x-3 p-4 rounded-lg border hover:bg-gray-50 cursor-pointer"
                        onClick={() => {
                          console.log("Div clicked for option:", option.optionId);
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
                                src={resolveImageUrl(option?.imageUrl as string)}
                                unoptimized
                                width={300}
                                height={200}
                                alt={"Gambar " + option.optionId}
                                className="h-auto max-w-full rounded object-contain"
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
                      {isPending ? "Menyimpan..." : "Selesai Section 1"}
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
