"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import type { CESAssessmentData } from "./types";
import Image from "next/image";
import { CheckCircle } from "lucide-react";

interface CompletionProps {
  assessmentData: CESAssessmentData;
  clearStoredData: () => void;
  role: string;
}

export default function Completion({ assessmentData, clearStoredData, role }: CompletionProps) {
  useEffect(() => {
    clearStoredData();
  }, [clearStoredData]);

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-4xl mx-auto px-6">
        <div className="bg-white rounded-lg shadow-sm border p-8 mb-3">
          <div className="flex justify-between items-center mb-6">
            <Image
              width={64}
              height={64}
              src="/images/logo1.png"
              alt="Logo Kiri"
              className="h-16 w-auto"
            />
            <div className="text-center">
              <h1 className="text-3xl font-bold uppercase text-gray-800 mb-2">
                Crew Evaluation System
              </h1>
              <p className="text-lg text-gray-600">{role.replace(/_/g, " ").toUpperCase()}</p>
            </div>
            <Image
              width={64}
              height={64}
              src="/images/logo2.png"
              alt="Logo Kanan"
              className="h-16 w-auto"
            />
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border p-8">
          <div className="text-center">
            <CheckCircle className="h-20 w-20 text-green-500 mx-auto mb-6" />
            <h2 className="text-3xl font-bold text-gray-800 mb-4">Assessment Selesai</h2>
            <p className="text-gray-700 mb-6">
              Terima kasih, <strong>{assessmentData.fullName}</strong>. Hasil assessment Anda sudah
              tersimpan.
            </p>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-6 my-6 text-left text-sm space-y-2">
              <p>
                <strong>Seafarer Code:</strong> {assessmentData.seafarerCode}
              </p>
              <p>
                <strong>Rank:</strong> {assessmentData.rank}
              </p>
              <p>
                <strong>Total Soal Dijawab:</strong> {Object.keys(assessmentData.answers).length}
              </p>
            </div>
            <Button
              onClick={() => {
                const basePath = process.env.NEXT_PUBLIC_BASE_PATH || "";
                window.location.href = `${basePath}/new-recruiter/crew-value-assessment/` + role;
              }}
            >
              Tutup
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
