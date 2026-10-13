"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Loader2, AlertTriangle, Database, Trash2, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { authFetch } from "@/app/lib/auth-fetch";

interface DataStatistics {
  reports: number;
  idp_tracking: number;
  gap_competencies: number;
  assessment_results: number;
  user_answers: number;
  coaching_reports: number;
  mentoring_reports: number;
  training_schedules: number;
}

export default function AdminPanel() {
  const [statistics, setStatistics] = useState<DataStatistics | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    action: string;
    title: string;
    description: string;
  }>({ open: false, action: "", title: "", description: "" });

  const fetchStatistics = async () => {
    setLoading(true);
    try {
      const baseURL = process.env.NEXT_PUBLIC_API_ENDPOINT || "http://localhost:8080";
      const response = await authFetch(`${baseURL}/admin/statistics`);
      if (!response.ok) throw new Error("Failed to fetch statistics");
      const result = await response.json();
      setStatistics(result.data);
    } catch (error) {
      toast.error("Failed to load statistics");
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatistics();
  }, []);

  const handleDeleteAction = async (action: string) => {
    setActionLoading(action);
    try {
      const baseURL = process.env.NEXT_PUBLIC_API_ENDPOINT || "http://localhost:8080";
      const response = await authFetch(`${baseURL}/admin/${action}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error(`Failed to delete ${action}`);
      const result = await response.json();
      toast.success(result.data.message || "Delete operation completed successfully");
      await fetchStatistics();
    } catch (error) {
      toast.error(`Failed to delete ${action}`);
      console.error(error);
    } finally {
      setActionLoading(null);
      setDeleteDialog({ open: false, action: "", title: "", description: "" });
    }
  };

  const confirmDelete = (action: string, title: string, description: string) => {
    setDeleteDialog({ open: true, action, title, description });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8 space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Admin Panel</h1>
          <p className="text-muted-foreground mt-2">
            Manage system data and perform administrative operations
          </p>
        </div>
        <Button onClick={fetchStatistics} variant="outline" size="sm">
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <div className="bg-yellow-50 dark:bg-yellow-900/10 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-500 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-yellow-800 dark:text-yellow-200">
              Warning: Delete operations are permanent and cannot be undone. Use with caution.
            </p>
          </div>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Reports</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{statistics?.reports || 0}</div>
            <p className="text-xs text-muted-foreground mt-1">Total seafarer reports</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">IDP Tracking</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{statistics?.idp_tracking || 0}</div>
            <p className="text-xs text-muted-foreground mt-1">Training progress records</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Gap Competencies</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{statistics?.gap_competencies || 0}</div>
            <p className="text-xs text-muted-foreground mt-1">Competency gap records</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Assessment Results</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{statistics?.assessment_results || 0}</div>
            <p className="text-xs text-muted-foreground mt-1">Total assessment submissions</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">User Answers</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{statistics?.user_answers || 0}</div>
            <p className="text-xs text-muted-foreground mt-1">Individual question responses</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Coaching Reports</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{statistics?.coaching_reports || 0}</div>
            <p className="text-xs text-muted-foreground mt-1">Coaching session records</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Mentoring Reports</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{statistics?.mentoring_reports || 0}</div>
            <p className="text-xs text-muted-foreground mt-1">Mentoring session records</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Training Schedules</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{statistics?.training_schedules || 0}</div>
            <p className="text-xs text-muted-foreground mt-1">Planned training sessions</p>
          </CardContent>
        </Card>
      </div>

      <Card className="border-destructive">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
          <CardDescription>
            These operations will permanently delete data from the database
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between p-4 border rounded-lg">
            <div>
              <h3 className="font-semibold">Delete All Reports</h3>
              <p className="text-sm text-muted-foreground">
                This will cascade delete gap_competencies, user_answers, assessment_results, and
                related records
              </p>
            </div>
            <Button
              variant="destructive"
              onClick={() =>
                confirmDelete(
                  "reports",
                  "Delete All Reports",
                  "This will permanently delete all reports and cascade to gap_competencies, user_answers, assessment_results, and related records. This action cannot be undone."
                )
              }
              disabled={actionLoading !== null}
            >
              {actionLoading === "reports" ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : (
                <Trash2 className="h-4 w-4 mr-2" />
              )}
              Delete Reports
            </Button>
          </div>

          <div className="flex items-center justify-between p-4 border rounded-lg">
            <div>
              <h3 className="font-semibold">Delete All IDP Tracking</h3>
              <p className="text-sm text-muted-foreground">
                Remove all training progress records without affecting reports
              </p>
            </div>
            <Button
              variant="destructive"
              onClick={() =>
                confirmDelete(
                  "idp-tracking",
                  "Delete All IDP Tracking",
                  "This will permanently delete all IDP tracking records. Training progress history will be lost. This action cannot be undone."
                )
              }
              disabled={actionLoading !== null}
            >
              {actionLoading === "idp-tracking" ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : (
                <Trash2 className="h-4 w-4 mr-2" />
              )}
              Delete IDP Tracking
            </Button>
          </div>

          <div className="flex items-center justify-between p-4 border rounded-lg">
            <div>
              <h3 className="font-semibold">Delete All Assessment Results</h3>
              <p className="text-sm text-muted-foreground">
                Remove all assessment submissions and answers
              </p>
            </div>
            <Button
              variant="destructive"
              onClick={() =>
                confirmDelete(
                  "assessment-results",
                  "Delete All Assessment Results",
                  "This will permanently delete all assessment results and user answers. Assessment history will be lost. This action cannot be undone."
                )
              }
              disabled={actionLoading !== null}
            >
              {actionLoading === "assessment-results" ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : (
                <Trash2 className="h-4 w-4 mr-2" />
              )}
              Delete Assessment Results
            </Button>
          </div>
        </CardContent>
      </Card>

      <AlertDialog
        open={deleteDialog.open}
        onOpenChange={(open) => !open && setDeleteDialog({ ...deleteDialog, open })}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{deleteDialog.title}</AlertDialogTitle>
            <AlertDialogDescription>{deleteDialog.description}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => handleDeleteAction(deleteDialog.action)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {actionLoading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              Confirm Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
