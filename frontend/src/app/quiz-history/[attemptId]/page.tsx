"use client";

import { useParams, useRouter } from "next/navigation";
import { useGetQuizAttempt } from "@/app/quiz/_hooks/useQuiz";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Loader2, ArrowLeft, CheckCircle, XCircle } from "lucide-react";
import Image from "next/image";
import { Badge } from "@/components/ui/badge";
import { isValidImageUrl, resolveImageUrl } from "@/app/lib/image";

export default function QuizAttemptDetailPage() {
  const params = useParams();
  const router = useRouter();
  const attemptId = Number(params.attemptId);

  const { data: attempt, isLoading, error } = useGetQuizAttempt(attemptId);

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  if (error || !attempt) {
    return (
      <div className="flex flex-col justify-center items-center min-h-screen gap-4">
        <p className="text-red-500">Gagal memuat detail assessment.</p>
        <Button onClick={() => router.back()}>Kembali</Button>
      </div>
    );
  }

  return (
    <div className="min-h-screen py-8 px-4 sm:px-6 lg:px-8">
      <div className="max-w-4xl mx-auto space-y-6">
        <Button variant="ghost" onClick={() => router.back()} className="mb-4">
          <ArrowLeft className="mr-2 h-4 w-4" /> Kembali ke Riwayat
        </Button>

        {/* Header Summary */}
        <Card>
          <CardHeader>
            <CardTitle className="flex justify-between items-center">
              <span>Hasil Assessment: {attempt.assessmentTypeName}</span>
              <Badge
                variant={attempt.percentageScore >= 70 ? "default" : "destructive"}
                className={attempt.percentageScore >= 70 ? "bg-green-600" : ""}
              >
                {attempt.percentageScore.toFixed(1)}%
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <p className="text-gray-500">Seafarer Code</p>
              <p className="font-medium">{attempt.seamanCode}</p>
            </div>
            <div>
              <p className="text-gray-500">Tanggal Selesai</p>
              <p className="font-medium">{attempt.completedAtFormatted}</p>
            </div>
            <div>
              <p className="text-gray-500">Skor Total</p>
              <p className="font-medium">
                {attempt.totalScore} / {attempt.maxScore}
              </p>
            </div>
            <div>
              <p className="text-gray-500">Status</p>
              <p className="font-medium">
                {attempt.percentageScore >= 70 ? "Lulus" : "Tidak Lulus"}
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Questions Detail Grouped by Assessment Name */}
        <div className="space-y-8">
          {Object.entries(
            attempt.answers.reduce(
              (groups, answer) => {
                const key = answer.assessmentName || "General Section";
                if (!groups[key]) {
                  groups[key] = [];
                }
                groups[key].push(answer);
                return groups;
              },
              {} as Record<string, typeof attempt.answers>
            )
          ).map(([assessmentName, answers]) => (
            <div key={assessmentName} className="space-y-4">
              <h3 className="text-xl font-bold text-gray-800 border-b pb-2">{assessmentName}</h3>
              <div className="space-y-4">
                {answers.map((answer, index) => (
                  <Card
                    key={answer.questionId}
                    className={`border-l-4 ${answer.isCorrect ? "border-l-green-500" : "border-l-red-500"}`}
                  >
                    <CardHeader className="pb-2">
                      <CardTitle className="text-base font-medium flex items-start gap-2">
                        <span className="text-gray-400 min-w-[24px]">Soal {index + 1}.</span>
                        <div className="flex-1">
                          {answer.questionText}
                          <div className="flex items-center mt-1">
                            {answer.isCorrect ? (
                              <span className="text-xs text-green-600 flex items-center font-bold">
                                <CheckCircle className="h-3 w-3 mr-1" /> Benar (+
                                {answer.scoreEarned})
                              </span>
                            ) : (
                              <span className="text-xs text-red-600 flex items-center font-bold">
                                <XCircle className="h-3 w-3 mr-1" /> Salah ({answer.scoreEarned}{" "}
                                pts)
                              </span>
                            )}
                          </div>
                        </div>
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3 pt-0">
                      {/* Options / Answer Display */}
                      {answer.questionType === "short_answer" ? (
                        <div className="space-y-2">
                          <div className="p-3 rounded-md bg-gray-50 border">
                            <p className="text-xs text-gray-500 mb-1">Jawaban Anda:</p>
                            <p
                              className={
                                answer.isCorrect
                                  ? "text-green-700 font-medium"
                                  : "text-red-700 font-medium"
                              }
                            >
                              {answer.textAnswer || "(Kosong)"}
                            </p>
                          </div>
                          {!answer.isCorrect && (
                            <div className="p-3 rounded-md bg-green-50 border border-green-100">
                              <p className="text-xs text-green-600 mb-1">Jawaban yang benar:</p>
                              <p className="text-green-800">
                                {answer.acceptableAnswers?.join(" atau ")}
                              </p>
                            </div>
                          )}
                        </div>
                      ) : (
                        <div className="space-y-2">
                          {answer.options?.map((opt) => {
                            const isSelected = opt.isSelected;
                            const isCorrectOption = opt.isCorrect; // Option that gives points

                            let bgClass = "bg-white";
                            let borderClass = "border-gray-200";
                            let textClass = "text-gray-700";

                            if (isSelected) {
                              if (isCorrectOption) {
                                bgClass = "bg-green-50";
                                borderClass = "border-green-300";
                                textClass = "text-green-800";
                              } else {
                                bgClass = "bg-red-50";
                                borderClass = "border-red-300";
                                textClass = "text-red-800";
                              }
                            } else if (isCorrectOption && !answer.isCorrect) {
                              // Show missed correct option
                              bgClass = "bg-green-50/50";
                              borderClass = "border-green-200 dashed";
                              textClass = "text-green-600";
                            }

                            return (
                              <div
                                key={opt.optionId}
                                className={`flex items-start p-3 rounded-lg border ${bgClass} ${borderClass} transition-colors`}
                              >
                                <div className="mr-3 mt-0.5">
                                  {isSelected ? (
                                    isCorrectOption ? (
                                      <CheckCircle className="h-4 w-4 text-green-600" />
                                    ) : (
                                      <XCircle className="h-4 w-4 text-red-600" />
                                    )
                                  ) : (
                                    <div className="h-4 w-4 rounded-full border border-gray-300" />
                                  )}
                                </div>
                                <div className={`flex-1 ${textClass}`}>
                                  <div className="flex flex-col">
                                    <span>{opt.optionText}</span>
                                    {opt.optionLetter && (
                                      <span className="text-xs opacity-70">
                                        ({opt.optionLetter})
                                      </span>
                                    )}
                                    {isValidImageUrl(opt.imageUrl) && (
                                      <Image
                                        src={resolveImageUrl(opt.imageUrl as string)}
                                        unoptimized
                                        alt="Option Image"
                                        width={200}
                                        height={150}
                                        className="mt-2 rounded border bg-white object-contain max-h-[150px]"
                                      />
                                    )}
                                  </div>
                                </div>
                                {isCorrectOption && !isSelected && (
                                  <span className="text-xs text-green-600 font-medium ml-2">
                                    Jawaban Benar
                                  </span>
                                )}
                                {isSelected && (
                                  <span className="text-xs font-medium ml-2">
                                    {isCorrectOption ? "Dipilih (Benar)" : "Dipilih (Salah)"}
                                  </span>
                                )}
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
