import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../lib/api";
import { reportKeys } from "./useReports";

// CITRIX pull + report refresh can take a while, so allow a longer timeout than the 20s default.
const SYNC_TIMEOUT_MS = 5 * 60 * 1000;

export function useCitrixSync() {
  const queryClient = useQueryClient();

  return useMutation<void, Error>({
    mutationFn: async () => {
      try {
        await api.post("/api/citrix/sync", null, { timeout: SYNC_TIMEOUT_MS });
      } catch (error: unknown) {
        const message =
          (error as { response?: { data?: { message?: string } } })?.response?.data?.message ??
          (error as Error)?.message ??
          "Sinkronisasi CITRIX gagal";
        throw new Error(message);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: reportKeys.all });
    },
  });
}
