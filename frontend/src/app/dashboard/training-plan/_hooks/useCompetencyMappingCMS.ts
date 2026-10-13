import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { BASE_URL } from "@/app/lib/api";
import { authFetch } from "@/app/lib/auth-fetch";

const API_BASE = BASE_URL;

export interface Training {
  no: number;
  competencyTypeId: number;
  lvl: number;
  deskripsi_perilaku: string;
  tools_training: string;
  kode: string;
  topik_training: string;
  competencyType?: {
    id: number;
    code: string;
    name: string;
    description?: string;
  };
}

export interface CompetencyMappingItem {
  id: number;
  competencyTypeId: number;
  program: string;
  trainingMaterial1Id: number | null;
  trainingMaterial2Id: number | null;
  createdAt: string;
  updatedAt: string;
  competencyType?: {
    id: number;
    code: string;
    name: string;
    description?: string;
  };
  trainingMaterial1?: Training;
  trainingMaterial2?: Training;
}

export interface CompetencyMappingFormData {
  competencyTypeId: number;
  program: string;
  trainingMaterial1Id: number | null;
  trainingMaterial2Id: number | null;
}

// Get all trainings for dropdowns
export function useGetAllTrainings() {
  return useQuery({
    queryKey: ["trainings", "all"],
    queryFn: async () => {
      const response = await authFetch(`${API_BASE}/api/trainings`);
      if (!response.ok) {
        throw new Error("Failed to fetch trainings");
      }
      const data = await response.json();
      return data.data as Training[];
    },
  });
}

// Get competency mappings by program
export function useGetCompetencyMappings(program: string) {
  return useQuery({
    queryKey: ["competency-mappings", program],
    queryFn: async () => {
      const response = await authFetch(`${API_BASE}/api/competency-mappings?program=${program}`);
      if (!response.ok) {
        throw new Error("Failed to fetch competency mappings");
      }
      const data = await response.json();
      return data.data as CompetencyMappingItem[];
    },
    enabled: !!program,
  });
}

// Create competency mapping
export function useCreateCompetencyMapping() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: CompetencyMappingFormData) => {
      const response = await authFetch(`${API_BASE}/api/competency-mappings`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        throw new Error("Failed to create competency mapping");
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["competency-mappings"] });
      toast.success("Competency mapping created successfully");
    },
    onError: (error: Error) => {
      toast.error(error.message);
    },
  });
}

// Update competency mapping
export function useUpdateCompetencyMapping() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, data }: { id: number; data: CompetencyMappingFormData }) => {
      const response = await authFetch(`${API_BASE}/api/competency-mappings/${id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        throw new Error("Failed to update competency mapping");
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["competency-mappings"] });
      queryClient.invalidateQueries({ queryKey: ["training-plan"] });
      toast.success("Competency mapping updated successfully");
    },
    onError: (error: Error) => {
      toast.error(error.message);
    },
  });
}

// Delete competency mapping
export function useDeleteCompetencyMapping() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: number) => {
      const response = await authFetch(`${API_BASE}/api/competency-mappings/${id}`, {
        method: "DELETE",
      });

      if (!response.ok) {
        throw new Error("Failed to delete competency mapping");
      }

      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["competency-mappings"] });
      toast.success("Competency mapping deleted successfully");
    },
    onError: (error: Error) => {
      toast.error(error.message);
    },
  });
}
