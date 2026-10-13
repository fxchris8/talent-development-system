"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import Image from "next/image";

const roleOptions = [
  {
    value: "nahkoda",
    label: "Nakhoda",
    description: "Assessment untuk posisi Nakhoda/Captain",
  },
  {
    value: "mualim_1",
    label: "Mualim I",
    description: "Assessment untuk posisi Mualim I/Chief Officer",
  },
  {
    value: "mualim_2",
    label: "Mualim II",
    description: "Assessment untuk posisi Mualim II/Second Officer",
  },
  {
    value: "mualim_3",
    label: "Mualim III",
    description: "Assessment untuk posisi Mualim III/Third Officer",
  },
  {
    value: "kkm",
    label: "KKM",
    description: "Assessment untuk posisi Kepala Kamar Mesin/Chief Engineer",
  },
  {
    value: "masinis_2",
    label: "Masinis II",
    description: "Assessment untuk posisi Masinis II/Second Engineer",
  },
  {
    value: "masinis_3",
    label: "Masinis III",
    description: "Assessment untuk posisi Masinis III/Third Engineer",
  },
  {
    value: "masinis_4",
    label: "Masinis IV",
    description: "Assessment untuk posisi Masinis IV/Fourth Engineer",
  },
];

export default function CESLandingPage() {
  const router = useRouter();

  const handleSelectRole = (role: string) => {
    router.push(`/crew-evaluation-system/${role}`);
  };

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-6xl mx-auto px-6">
        {/* Header Section */}
        <div className="bg-white rounded-lg shadow-sm border p-8 mb-6">
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
              <p className="text-gray-600">Pilih role untuk memulai assessment</p>
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

        {/* Role Selection Grid */}
        <div className="bg-white rounded-lg shadow-sm border p-8">
          <h2 className="text-2xl font-bold text-gray-800 mb-6">Pilih Role Assessment</h2>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {roleOptions.map((role) => (
              <div
                key={role.value}
                className="border border-gray-200 rounded-lg p-6 hover:border-gray-400 hover:shadow-md transition-all cursor-pointer group"
                onClick={() => handleSelectRole(role.value)}
              >
                <div className="flex-1">
                  <h3 className="font-bold text-lg text-gray-800 mb-1">{role.label}</h3>
                  <p className="text-sm text-gray-600">{role.description}</p>
                </div>
                <Button
                  className="w-full mt-4 bg-gray-800 hover:bg-gray-700"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleSelectRole(role.value);
                  }}
                >
                  Mulai Assessment
                </Button>
              </div>
            ))}
          </div>

          <div className="mt-8 p-4 bg-blue-50 border border-blue-200 rounded-lg">
            <p className="text-sm text-blue-800">
              <strong>Catatan:</strong> Pastikan Anda memilih role yang sesuai dengan posisi Anda.
              Assessment akan menggunakan soal-soal yang spesifik untuk role yang dipilih.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
