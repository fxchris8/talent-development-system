import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";
import type {
  IReport,
  IPaginationData,
  IPaginationRequest,
  ApiResponse,
} from "@/types/global-types";

interface IdpCountData {
  mdp: number;
  fdp: number;
  sdp: number;
  total?: number;
}

// Query key factory for reports queries
export const reportKeys = {
  all: ["reports"] as const,
  list: () => [...reportKeys.all, "list"] as const,
  paginated: (request: IPaginationRequest) => [...reportKeys.list(), request] as const,
  idpCount: (batchId?: number | null) =>
    [...reportKeys.all, "idp-count", batchId ?? "all"] as const,
};

// React Query hook for fetching IDP (Integrated Development Programs) count
export function useGetIdpCount(batchId?: number | null, query?: string) {
  const response = useQuery<IdpCountData, Error>({
    queryKey: [...reportKeys.idpCount(batchId), query ?? ""],
    queryFn: async () => {
      // Add required Page and PageSize parameters matching DashboardRequest structure
      const params = new URLSearchParams({
        page: "next", // Must be "next" or "prev" according to backend validation
        page_size: "1000", // Use a large page size since we just need the count
        anchor_id: "0", // Optional but good to include
        in_program: "true", // the dashboard only counts people currently in a program
      });
      if (batchId !== undefined && batchId !== null) {
        params.set("batch_id", String(batchId));
      }
      if (query) {
        params.set("query", query);
      }
      const response = await api.get<ApiResponse<IdpCountData>>(
        `/reports/idp-count?${params.toString()}`
      );

      if (!response.data) {
        throw new Error("Failed to fetch IDP count data");
      }

      return response.data.data;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes (formerly cacheTime)
    retry: (failureCount, error: unknown) => {
      // Don't retry on 404 or authentication errors
      if (error && typeof error === "object" && "response" in error) {
        const axiosError = error as { response: { status: number } };
        if (axiosError.response?.status === 404 || axiosError.response?.status === 401) {
          return false;
        }
      }
      return failureCount < 3;
    },
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000), // Exponential backoff
  });

  return response;
}

// React Query hook for fetching paginated reports
export function useGetReports(paginationRequest: IPaginationRequest) {
  const response = useQuery<IPaginationData<IReport>, Error>({
    queryKey: reportKeys.paginated(paginationRequest),
    queryFn: async () => {
      try {
        const params: Record<string, string> = {
          anchor_id: (paginationRequest.anchorId ?? 0).toString(),
          page: paginationRequest.page,
          page_size: String(paginationRequest.pageSize),
          filter: paginationRequest.filter,
          in_program: "true", // people who left their program are shown in the exit history
        };
        if (paginationRequest.batchId !== undefined && paginationRequest.batchId !== null) {
          params.batch_id = String(paginationRequest.batchId);
        }
        if (paginationRequest.query) {
          params.query = paginationRequest.query;
        }
        const response = await api.get<ApiResponse<IPaginationData<IReport>>>(`/reports`, {
          params,
        });

        if (!response.data) {
          console.error("Response data is empty");
          throw new Error("Failed to fetch reports data");
        }

        return response.data.data;
      } catch (error) {
        console.error("Error fetching reports:", error);
        throw error;
      }
    },
    enabled: true, // Always enabled since paginationRequest is always provided
    // Keep the current rows on screen while the next page / filter loads
    placeholderData: keepPreviousData,
    staleTime: 3 * 60 * 1000, // 3 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes (formerly cacheTime)
  });

  return response;
}
