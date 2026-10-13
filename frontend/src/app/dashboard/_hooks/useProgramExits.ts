import { useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";
import type { ApiResponse } from "@/types/global-types";
import { reportKeys } from "./useReports";

// One person who left an IDP program because of a promotion.
export interface ProgramExit {
  reportId: number;
  seafarerCode: string;
  nama: string;
  exitedProgram: string;
  fromPosition: string; // jabatan saat masih di program ("" bila tidak diketahui)
  toPosition: string; // jabatan tujuan kenaikan
  currentPosition: string; // jabatan sekarang
  vesselName: string;
  promotedAt: string | null; // tanggal naik jabatan di CITRIX
  exitedAt: string | null; // kapan sistem mengeluarkannya dari program
  // "completed" = semua training program selesai, "promoted" = naik jabatan sebelum selesai
  reason: "completed" | "promoted";
  promoted: boolean; // sudah naik jabatan keluar dari program
  completedAt: string | null; // kapan program diselesaikan, bila diketahui
}

export function useProgramExits() {
  return useQuery<ProgramExit[], Error>({
    // under reportKeys.all so a CITRIX sync refreshes this list too
    queryKey: [...reportKeys.all, "program-exits"],
    queryFn: async () => {
      const response = await api.get<ApiResponse<ProgramExit[]>>("/reports/program-exits");
      return response.data.data ?? [];
    },
    staleTime: 3 * 60 * 1000,
  });
}
