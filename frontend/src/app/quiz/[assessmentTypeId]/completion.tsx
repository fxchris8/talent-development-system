"use client";

import { CheckCircle } from "lucide-react";
import Image from "next/image";

interface CompletionProps {
  assessmentTypeName: string;
  seamanCode: string;
  email: string;
}

export default function Completion({ assessmentTypeName, seamanCode, email }: CompletionProps) {
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
                {assessmentTypeName}
              </h1>
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

        {/* Completion Message */}
        <div className="bg-white rounded-lg shadow-sm border p-8 mb-3">
          <div className="text-center">
            <div className="flex justify-center mb-6">
              <CheckCircle className="h-24 w-24 text-green-500" />
            </div>

            <h2 className="text-3xl font-bold text-gray-800 mb-4">
              Selamat! Assessment Berhasil Diselesaikan
            </h2>

            <p className="text-lg text-gray-600 mb-8">
              Terima kasih telah menyelesaikan {assessmentTypeName} dengan sungguh-sungguh.
            </p>
          </div>
        </div>

        {/* Assessment Summary */}
        <div className="bg-white rounded-lg shadow-sm border p-8 mb-3">
          <h3 className="text-xl font-bold text-gray-800 mb-6 border-b pb-3">
            Ringkasan Assessment
          </h3>

          <div className="space-y-4">
            <div className="flex justify-between">
              <span className="font-medium text-gray-700">Kode Pelaut:</span>
              <span className="text-gray-600">{seamanCode}</span>
            </div>
            <div className="flex justify-between">
              <span className="font-medium text-gray-700">Email:</span>
              <span className="text-gray-600">{email}</span>
            </div>
            <div className="flex justify-between">
              <span className="font-medium text-gray-700">Assessment:</span>
              <span className="text-gray-600">{assessmentTypeName}</span>
            </div>
            <div className="flex justify-between">
              <span className="font-medium text-gray-700">Status:</span>
              <span className="text-green-600 font-medium">Selesai</span>
            </div>
          </div>
        </div>

        {/* Next Steps */}
        <div className="bg-white rounded-lg shadow-sm border p-8 mb-3">
          <div className="space-y-4 text-gray-700">
            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0 w-6 h-6 bg-blue-100 rounded-full flex items-center justify-center mt-0.5">
                <span className="text-blue-600 text-sm font-medium">1</span>
              </div>
              <p>
                Hasil assessment Anda akan diproses dan dianalisis oleh tim HR dalam waktu 3-5 hari
                kerja.
              </p>
            </div>

            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0 w-6 h-6 bg-blue-100 rounded-full flex items-center justify-center mt-0.5">
                <span className="text-blue-600 text-sm font-medium">2</span>
              </div>
              <p>
                Anda akan menerima notifikasi melalui email mengenai hasil assessment dan langkah
                selanjutnya.
              </p>
            </div>

            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0 w-6 h-6 bg-blue-100 rounded-full flex items-center justify-center mt-0.5">
                <span className="text-blue-600 text-sm font-medium">3</span>
              </div>
              <p>
                Jika ada pertanyaan, silakan hubungi tim recruitment di
                <span className="font-medium text-blue-600"> recruitment.crewspil@gmail.com</span>
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
