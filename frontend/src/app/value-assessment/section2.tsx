"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { ValueAssessmentData } from "./page";
import { useGetAssessmentByRole, usePostAssessmentResults } from "./_hooks/useAssessment";
import { TutorialDisplay } from "@/components/tutorial-display";
import { useCountdown } from "@/hooks/use-session-storage";
import { useTimerPauseResume } from "@/hooks/useTimerPauseResume";
import { useStorageCountdown } from "@/hooks/use-local-storage";
import Image from "next/image";
import { isValidImageUrl, resolveImageUrl } from "@/app/lib/image";

interface Section2Props {
  onNext: () => void;
  onBack: () => void;
  assessmentData: ValueAssessmentData;
  updateAssessmentData: (data: Partial<ValueAssessmentData>) => void;
}

export default function Section2({
  onNext,
  onBack,
  assessmentData,
  updateAssessmentData,
}: Section2Props) {
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [answers, setAnswers] = useState<{ [questionId: number]: number }>(
    assessmentData.section2Answers ?? {}
  );
  const timeOutRef = useRef(false);
  const storageTimeOutRef = useRef(false);
  const [tutorialDismissed, setTutorialDismissed] = useState(
    assessmentData.section2TutorialDismissed ?? false
  );

  // Fetch assessment data using the hook
  const { data: assessment, isLoading, error } = useGetAssessmentByRole("va_2");

  // POST mutation hook
  const { mutate, isPending } = usePostAssessmentResults(onNext);

  // Timer setup untuk section
  const timerMinutes = assessment?.timerLimitMinutes ?? 60;
  const { timeLeft } = useCountdown(
    assessmentData.section2StartTime,
    timerMinutes,
    assessmentData.section2PauseTimestamp
  );

  // Timer sync is now handled by the main page component
  // This section only needs to store its timer duration

  // Store timer minutes in assessmentData for progress bar to use
  useEffect(() => {
    if (timerMinutes && assessmentData.section2TimerMinutes !== timerMinutes) {
      updateAssessmentData({ section2TimerMinutes: timerMinutes });
    }
  }, [timerMinutes, assessmentData.section2TimerMinutes, updateAssessmentData]);

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
    if (assessment && !assessment.tutorialContent && !assessmentData.section2StartTime) {
      updateAssessmentData({ section2StartTime: new Date().toISOString() });
    }
  }, [assessment, assessmentData.section2StartTime, updateAssessmentData]);

  // Stable pause/resume callbacks to prevent hook re-setup
  const handlePause = useCallback(() => {
    const now = new Date().toISOString();
    console.log(`⏸️ Section 2 paused`);
    updateAssessmentData({ section2PauseTimestamp: now });
  }, [updateAssessmentData]);

  const handleResume = useCallback(() => {
    console.log(`▶️ Section 2 resumed`);
    updateAssessmentData({ section2PauseTimestamp: undefined });
  }, [updateAssessmentData]);

  // Handle pause/resume when tab loses focus or visibility changes
  useTimerPauseResume(
    assessmentData.currentStep === 4, // isActive - true when user is on section 2
    handlePause,
    handleResume
  );

  // Overall assessment timer (24 hours)
  const { isExpired: isOverallExpired } = useStorageCountdown("valueAssessmentFormData");

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

    // For public assessment, we only track the option selected
    const newAnswers = {
      ...answers,
      [currentQuestion.questionId]: optionId,
    };
    console.log("New Answers:", newAnswers);
    setAnswers(newAnswers);
    // Update parent immediately when answer changes
    updateAssessmentData({ section2Answers: newAnswers });
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
      role: "va_2",
      answers: filteredAnswers,
    });
  }, [assessment?.questions, answers, assessmentData.seafarerCode, mutate, timeLeft]);

  // Auto-submit when timer reaches 0
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
          <p className="text-gray-600">Tidak ada soal yang tersedia untuk Section 2</p>
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
            section2TutorialDismissed: true,
            ...(!assessmentData.section2StartTime && {
              section2StartTime: new Date().toISOString(),
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
                Value Assessment Section 2
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
              Pada asesmen bagian 2 ini akan terdiri dari 60 pasang pernyataan, masing-masing berisi
              dua pernyataan (a dan b) bacalah masing-masing pernyataan dengan cermat.
            </p>
            <p>
              Pilihlah satu pernyataan yang paling menggambarkan sikap atau kebiasaan Anda dalam
              situasi kerja. Bila tidak ada satu pun dari pasangan pernyataan yang cocok, pilihlah
              yang Anda anggap benar.
            </p>
            <p>
              Jawablah dengan jujur dan sesuai dengan diri Anda, bukan berdasarkan jawaban yang Anda
              anggap ideal. Tidak ada jawaban benar atau salah dalam tes ini.
            </p>
          </div>

          <div className="bg-blue-50 border-l-4 border-blue-400 p-4">
            <h3 className="font-bold text-blue-800 mb-2">Contoh:</h3>
            <div className="space-y-2 text-blue-700">
              <p>
                <strong>a.</strong> saya adalah pekerja keras
              </p>
              <p>
                <strong>b.</strong> saya tidak mudah murung
              </p>
            </div>
            <p className="text-blue-600 text-sm mt-2">
              Dalam hal ini apabila pernyataan &quot;a&quot; merupakan diri Anda maka pilihlah
              jawaban &quot;a&quot;, begitu pun sebaliknya.
            </p>
          </div>
        </div>

        <div className="flex flex-col-reverse md:flex-row gap-6">
          {/* Question Navigation Sidebar */}
          <div className="w-full md:w-64 bg-white rounded-lg shadow-sm border p-6">
            <h3 className="font-bold text-lg mb-4 text-gray-800">Navigasi Soal</h3>

            {/* Progress Summary */}
            <div className="mb-6 p-4 bg-whit rounded-lg w-full">
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
                <div className="flex justify-between items-center mb-6">
                  <h3 className="text-lg font-bold text-gray-800">
                    Pernyataan {currentQuestionIndex + 1} dari {assessment?.questions?.length ?? 0}
                  </h3>
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
                  <div className="space-y-6">
                    {currentOptions.map((option) => (
                      <div
                        key={option.optionId}
                        className="flex items-center space-x-4 p-6 rounded-lg border-2 hover:bg-gray-50 transition-colors cursor-pointer"
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
                          <div className="flex items-center gap-3">
                            <span className="font-bold text-lg text-gray-800 min-w-6">
                              {option.optionLetter}.
                            </span>
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
                          </div>
                        </Label>
                      </div>
                    ))}
                  </div>
                </RadioGroup>
              </div>

              {/* Navigation Buttons */}
              <div className="flex flex-col-reverse gap-3 md:flex-row justify-between pt-6 border-t w-full">
                <Button
                  onClick={currentQuestionIndex === 0 ? onBack : handlePrevious}
                  variant="outline"
                  className="px-6 py-2 cursor-pointer"
                >
                  {currentQuestionIndex === 0 ? "Kembali ke Section 1" : "Soal Sebelumnya"}
                </Button>

                <div className="flex gap-3">
                  {currentQuestionIndex < (assessment?.questions?.length ?? 1) - 1 ? (
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
                      {isPending ? "Menyimpan..." : "Selesai Section 2"}
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
