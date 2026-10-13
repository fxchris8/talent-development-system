"use client";

import Image from "next/image";

interface HeaderSectionProps {
  title: string;
  logo1?: string;
  logo2?: string;
}

export default function HeaderSection({
  title,
  logo1 = "/images/logo1.png",
  logo2 = "/images/logo2.png",
}: HeaderSectionProps) {
  return (
    <div className="bg-white rounded-lg shadow-sm border p-8 mb-3">
      <div className="flex justify-between items-center mb-6">
        <Image width={64} height={64} src={logo1} alt="Logo Kiri" className="h-16 w-auto" />
        <div className="text-center">
          <h1 className="text-3xl font-bold uppercase text-gray-800 mb-2">{title}</h1>
        </div>
        <Image width={64} height={64} src={logo2} alt="Logo Kanan" className="h-16 w-auto" />
      </div>
    </div>
  );
}
