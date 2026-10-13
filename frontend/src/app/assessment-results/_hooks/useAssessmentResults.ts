import { useQuery } from "@tanstack/react-query";
import { api } from "../../lib/api";

// Types
export interface AspectScoreResult {
  aspectId: number;
  aspectName: string;
  rawScore: number;
  finalScore: number;
}

export interface CorevaReportData {
  rawScore: number;
  finalScore: number;
  konversiScore: number;
  scoreResult: AspectScoreResult[];
}

export interface AssessmentReportData {
  rawScore: number;
  finalScore: number;
  konversiScore: number;
}

export interface ValueAssessmentReport {
  seafarerCode: string;
  completedAt: string | null;
  totalScore: number;
  valueAssessment: number;
  valueAssessmentScore: number;
  interpretasi: string;
  coreva: CorevaReportData;
  aware: AssessmentReportData;
  grit: AssessmentReportData;
}

export interface AssessmentBySeafarerCode {
  totalFinalScore: number;
}

interface ApiResponse<T> {
  code: number;
  message: string;
  data: T | null;
}

// Query key factory for assessment queries
export const assessmentKeys = {
  all: ["assessments"] as const,
  report: (seafarerCode: string) => [...assessmentKeys.all, "report", seafarerCode] as const,
  bySeafarerCode: (seafarerCode: string) =>
    [...assessmentKeys.all, "seafarer", seafarerCode] as const,
};

/**
 * Hook untuk mendapatkan Value Assessment Report berdasarkan seafarer code
 * Endpoint: GET /assessment-results/report/:seafarerCode
 */
export function useGetAssessmentReport(seafarerCode: string) {
  return useQuery<ValueAssessmentReport | null, Error>({
    queryKey: assessmentKeys.report(seafarerCode),
    queryFn: async () => {
      console.log("Fetching assessment report for:", seafarerCode);

      try {
        const response = await api.get<ApiResponse<ValueAssessmentReport>>(
          `/assessment-results/report/${seafarerCode}`
        );

        console.log("Assessment report response:", response);

        // Check if response is successful
        if (response.data && response.data.code === 200) {
          if (response.data.data) {
            return response.data.data;
          } else {
            // Backend returned success but with null data
            return null;
          }
        }

        return null;
      } catch (error) {
        // 404 only means this seafarer has not completed the assessment yet
        const status = (error as { response?: { status?: number } })?.response?.status;
        if (status === 404) {
          return null;
        }
        console.error("Error fetching assessment report:", error);
        throw error;
      }
    },
    enabled: Boolean(seafarerCode), // Only run query if seafarerCode exists
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
}

/**
 * Hook untuk mendapatkan assessment results berdasarkan seafarer code
 * Endpoint: GET /assessment-results/seafarer/:seafarerCode
 * Digunakan untuk mendapatkan totalFinalScore saja
 */
export function useGetAssessmentBySeafarerCode(seafarerCode: string) {
  return useQuery<number | null, Error>({
    queryKey: assessmentKeys.bySeafarerCode(seafarerCode),
    queryFn: async () => {
      console.log("Fetching assessment by seafarer code:", seafarerCode);

      try {
        const response = await api.get<ApiResponse<AssessmentBySeafarerCode>>(
          `/assessment-results/seafarer/${seafarerCode}`
        );

        console.log("Assessment by seafarer code response:", response);

        // Check if response is successful
        if (response.data && response.data.code === 200) {
          if (response.data.data && response.data.data.totalFinalScore) {
            // Round to 1 decimal place
            return Math.round(response.data.data.totalFinalScore * 10) / 10;
          }
        }

        return null;
      } catch (error) {
        console.error("Error fetching assessment by seafarer code:", error);
        // Return null instead of throwing to prevent error state for missing assessments
        return null;
      }
    },
    enabled: Boolean(seafarerCode), // Only run query if seafarerCode exists
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    retry: false, // Don't retry for this query as missing data is expected
  });
}
