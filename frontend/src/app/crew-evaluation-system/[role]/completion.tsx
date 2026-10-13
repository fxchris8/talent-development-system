"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import type { CESAssessmentData } from "../types";
import Image from "next/image";
import { CheckCircle } from "lucide-react";

interface CompletionProps {
  assessmentData: CESAssessmentData;
  clearStoredData: () => void;
  role: string;
}

export default function Completion({ assessmentData, clearStoredData, role }: CompletionProps) {
  const formatRoleName = (roleSlug: string) => {
    return roleSlug
      .split("_")
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(" ");
  };

  // Clear stored data when component mounts (assessment completed)
  useEffect(() => {
    clearStoredData();
  }, [clearStoredData]);

  const handleClose = () => {
    // Close window or redirect
    window.close();
    // Fallback if window.close() doesn't work
    setTimeout(() => {
      window.location.href = "/crew-evaluation-system";
    }, 100);
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
              className="h-16 w-auto"
            />
            <div className="text-center">
              <h1 className="text-3xl font-bold uppercase text-gray-800 mb-2">
                Crew Evaluation System
              </h1>
              <p className="text-lg text-gray-600">{formatRoleName(role)}</p>
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

        {/* Completion Content */}
        <div className="bg-white rounded-lg shadow-sm border p-8">
          <div className="text-center">
            <div className="mb-6">
              <CheckCircle className="h-20 w-20 text-green-500 mx-auto" />
            </div>

            <h2 className="text-3xl font-bold text-gray-800 mb-4">Assessment Selesai!</h2>

            <div className="space-y-4 text-gray-700 max-w-2xl mx-auto">
              <p className="text-lg">
                Terima kasih, <strong>{assessmentData.fullName}</strong>!
              </p>

              <p>
                Anda telah berhasil menyelesaikan <strong>Crew Evaluation System Assessment</strong>{" "}
                untuk role <strong>{formatRoleName(role)}</strong>.
              </p>

              <div className="bg-blue-50 border border-blue-200 rounded-lg p-6 my-6">
                <h3 className="font-bold text-blue-800 mb-3">Informasi Submission</h3>
                <div className="text-left space-y-2 text-sm">
                  <p>
                    <strong>Seafarer Code:</strong> {assessmentData.seafarerCode}
                  </p>
                  <p>
                    <strong>Role:</strong> {formatRoleName(role)}
                  </p>
                  <p>
                    <strong>Vessel:</strong> {assessmentData.vesselName}
                  </p>
                  <p>
                    <strong>Waktu Selesai:</strong>{" "}
                    {new Date().toLocaleString("id-ID", {
                      dateStyle: "full",
                      timeStyle: "medium",
                    })}
                  </p>
                  <p>
                    <strong>Total Soal Dijawab:</strong>{" "}
                    {Object.keys(assessmentData.answers).length}
                  </p>
                </div>
              </div>

              <p className="text-gray-600">
                Hasil assessment Anda telah tersimpan dan akan diproses oleh tim kami. Anda akan
                menerima notifikasi lebih lanjut mengenai hasil assessment melalui email yang
                terdaftar.
              </p>

              <p className="text-gray-600 mt-4">Jika ada pertanyaan, silakan hubungi:</p>

              <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
                <p className="font-medium text-gray-800">shiplearningdevelopment.spil@gmail.com</p>
              </div>
            </div>

            <div className="mt-8 pt-6 border-t">
              <Button
                onClick={handleClose}
                className="px-8 py-3 bg-gray-800 hover:bg-gray-700 text-white font-medium"
              >
                Tutup
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
