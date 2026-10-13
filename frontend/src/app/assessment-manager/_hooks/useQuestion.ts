import api from "@/app/lib/api";
import { ApiReturn } from "@/app/types/api";
import { QuestionOptionResponse } from "@/types/assessment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { authFetch } from "@/app/lib/auth-fetch";

export type QuestionCreatePayload = {
  role: string;
  assessmentId: number;
  questionText: string;
  category?: string;
  isImage?: number;
  imageUrl?: string;
  options: {
    optionLetter: string;
    optionText: string;
    score: number;
    isImage?: number;
    imageUrl?: string | null;
  }[];
};

export type QuestionUpdatePayload = {
  role: string;
  assessmentId: number;
  questionText: string;
  category?: string;
  isImage?: number;
  imageUrl?: string;
  options: {
    optionId?: number;
    optionLetter: string;
    optionText: string;
    score: number;
    isImage?: number;
    imageUrl?: string | null;
    action?: string;
  }[];
};

// Hook for deleting a question with options
export function useDeleteQuestion() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (questionId: number) => {
      const response = await api.delete<ApiReturn<void>>(
        `/api/questions-with-options/${questionId}`
      );

      if (!response.data) {
        throw new Error("Failed to delete question");
      }

      return response.data;
    },
    onSuccess: () => {
      // Invalidate questions queries to refresh data
      queryClient.invalidateQueries({ queryKey: ["questions"] });
    },
  });
}

// Hook for updating a question with options
export function useUpdateQuestion() {
  const queryClient = useQueryClient();

  return useMutation<
    QuestionOptionResponse,
    Error,
    { questionId: number; data: QuestionUpdatePayload }
  >({
    mutationFn: async ({ questionId, data }) => {
      const response = await api.put<ApiReturn<QuestionOptionResponse>>(
        `/api/questions-with-options/${questionId}`,
        data
      );

      if (!response.data) {
        throw new Error("Failed to update question");
      }

      return response.data.data;
    },
    onSuccess: () => {
      // Invalidate questions queries to refresh data
      queryClient.invalidateQueries({ queryKey: ["questions"] });
    },
  });
}

export function useBulkDeleteQuestions() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, number[]>({
    mutationFn: async (questionIds: number[]) => {
      const response = await api.delete<ApiReturn<void>>(
        `/api/questions-with-options/bulk-delete`,
        { data: { questionIds } }
      );

      if (!response.data) {
        throw new Error("Failed to bulk delete questions");
      }
    },
    onSuccess: () => {
      // Invalidate questions queries to refresh data
      queryClient.invalidateQueries({ queryKey: ["questions"] });
    },
  });
}

export function useBulkUpdateAspect() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, { questionIds: number[]; aspectId: number | null }>({
    mutationFn: async ({ questionIds, aspectId }) => {
      const response = await api.put<ApiReturn<void>>(`/api/questions/bulk/aspect`, {
        questionIds,
        aspectId,
      });

      if (!response.data) {
        throw new Error("Failed to bulk update aspect");
      }
    },
    onSuccess: () => {
      // Invalidate questions queries to refresh data
      queryClient.invalidateQueries({ queryKey: ["questions"] });
    },
  });
}

// Hook for creating a new question with options
export function useCreateQuestion() {
  const queryClient = useQueryClient();

  return useMutation<QuestionOptionResponse, Error, QuestionCreatePayload>({
    mutationFn: async (data) => {
      const response = await api.post<ApiReturn<QuestionOptionResponse>>(
        `/api/questions-with-options`,
        data
      );

      if (!response.data) {
        throw new Error("Failed to create question");
      }

      return response.data.data;
    },
    onSuccess: () => {
      // Invalidate questions queries to refresh data
      queryClient.invalidateQueries({ queryKey: ["questions"] });
    },
  });
}

export const useGetQuestionsByAssessmentId = (assessmentId: number) => {
  return useQuery({
    queryKey: ["questions", "assessment", assessmentId],
    queryFn: async () => {
      if (assessmentId === 0) return null;

      const baseURL = process.env.NEXT_PUBLIC_API_ENDPOINT || "http://localhost:8080";
      const response = await authFetch(`${baseURL}/api/questions/assessment/${assessmentId}`, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        throw new Error("Failed to fetch questions by assessment ID");
      }

      return response.json();
    },
    enabled: assessmentId > 0,
  });
};
