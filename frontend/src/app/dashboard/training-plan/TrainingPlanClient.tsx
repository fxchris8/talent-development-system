"use client";

import { useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { Loader2, Users, TrendingUp, Calendar, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { SkeletonCard } from "@/components/skeleton-card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import {
  useGetTrainingPlan,
  useGetCompetencyMapping,
  useGetPrograms,
  useGenerateSchedules,
  useGetOverdueCount,
  type TrainingPlanParticipant,
} from "./_hooks/useTrainingPlan";
import TrainingScheduleTimeline from "./TrainingScheduleTimeline";
import CompetencyMappingCMS from "./CompetencyMappingCMS";
import TrainingMaterialTab from "./TrainingMaterialTab";
import { authFetch } from "@/app/lib/auth-fetch";

export default function TrainingPlanClient() {
  const [selectedProgram, setSelectedProgram] = useState("SDP");
  const [selectedParticipant, setSelectedParticipant] = useState<TrainingPlanParticipant | null>(
    null
  );
  const [showGenerateDialog, setShowGenerateDialog] = useState(false);
  const [startDate, setStartDate] = useState("");

  // React Query hooks
  const { data: programs, isLoading: programsLoading } = useGetPrograms();
  const {
    data: trainingPlan,
    isLoading: planLoading,
    error: planError,
  } = useGetTrainingPlan(selectedProgram);
  const { data: competencyMapping } = useGetCompetencyMapping(selectedProgram);
  const { data: overdueCount } = useGetOverdueCount(selectedProgram);

  // Fetch overdue counts for all programs (for dropdown badges)
  const { data: sdpOverdueCount } = useGetOverdueCount("SDP");
  const { data: mdpOverdueCount } = useGetOverdueCount("MDP");
  const { data: fdpOverdueCount } = useGetOverdueCount("FDP");

  const generateSchedules = useGenerateSchedules();

  // Get overdue count for a specific program code
  const getOverdueCountForProgram = (programCode: string): number => {
    switch (programCode) {
      case "SDP":
        return sdpOverdueCount || 0;
      case "MDP":
        return mdpOverdueCount || 0;
      case "FDP":
        return fdpOverdueCount || 0;
      default:
        return 0;
    }
  };

  // Handle program change
  const handleProgramChange = (program: string) => {
    setSelectedProgram(program);
  };

  // Handle schedule generation
  const handleGenerateSchedules = async () => {
    setShowGenerateDialog(true);
  };

  const confirmGenerate = async () => {
    setShowGenerateDialog(false);
    try {
      await generateSchedules.mutateAsync({
        program: selectedProgram,
        startDate: startDate || undefined,
      });
      toast.success("Training schedules generated successfully!");
      setStartDate(""); // Reset date after generation
    } catch (error) {
      toast.error("Failed to generate training schedules");
      console.error("Schedule generation error:", error);
    }
  };

  const handleExportExcel = async () => {
    if (!trainingPlan?.summary) {
      toast.error("No training plan data available to export");
      return;
    }

    try {
      const response = await authFetch(
        `${process.env.NEXT_PUBLIC_API_ENDPOINT}/api/training-plan/export-excel?program=${selectedProgram}`
      );

      if (!response.ok) {
        throw new Error("Failed to export Excel");
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `Training_Plan_${selectedProgram}_${new Date().toISOString().split("T")[0]}.xlsx`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      toast.success("Excel exported successfully!");
    } catch {
      toast.error("Failed to export Excel");
    }
  };

  // Get competency badge color based on category
  const getCompetencyBadgeColor = (category: string) => {
    // Support both full label and short code
    return category === "M" || category === "Mandatory" ? "destructive" : "secondary";
  };

  // Get readiness badge color
  const getReadinessBadgeColor = (readiness: string) => {
    switch (readiness.toLowerCase()) {
      case "ready now":
        return "default";
      case "ready 1-2 years":
        return "secondary";
      case "ready 3+ years":
        return "outline";
      default:
        return "outline";
    }
  };

  // Show error notification if training plan fetch fails
  if (planError) {
    toast.error("Failed to fetch training plan data");
  }

  // Loading state
  if (programsLoading || planLoading || !trainingPlan) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="space-y-2">
            <div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
            <div className="h-4 w-96 bg-gray-200 rounded animate-pulse" />
          </div>
          <div className="h-10 w-32 bg-gray-200 rounded animate-pulse" />
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <SkeletonCard key={i} />
          ))}
        </div>
        <div className="space-y-4">
          <div className="h-10 w-full bg-gray-200 rounded animate-pulse" />
          <div className="h-64 w-full bg-gray-200 rounded animate-pulse" />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Training Plan</h1>
          <p className="text-muted-foreground">
            Manage and generate training schedules based on competency gaps
          </p>
        </div>
        <div className="flex items-center gap-4">
          <Select value={selectedProgram} onValueChange={handleProgramChange}>
            <SelectTrigger className="w-[320px]">
              <SelectValue placeholder="Select Program" />
            </SelectTrigger>
            <SelectContent className="w-[320px]">
              {programs?.map((program) => {
                const programOverdueCount = getOverdueCountForProgram(program.code);
                return (
                  <SelectItem key={program.code} value={program.code} className="cursor-pointer">
                    <div className="flex items-center justify-between w-full">
                      <span className="flex-1 pr-3">{program.name}</span>
                      {programOverdueCount > 0 && (
                        <Badge
                          variant="destructive"
                          className="h-4 w-4 rounded-full p-0 flex items-center justify-center text-[10px] font-medium flex-shrink-0"
                        >
                          {programOverdueCount}
                        </Badge>
                      )}
                    </div>
                  </SelectItem>
                );
              })}
            </SelectContent>
          </Select>
          <Button
            onClick={handleGenerateSchedules}
            disabled={generateSchedules.isPending}
            className="gap-2"
          >
            {generateSchedules.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Calendar className="h-4 w-4" />
            )}
            Generate Schedules
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      {trainingPlan && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Participants</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{trainingPlan?.participants?.length || 0}</div>
              <p className="text-xs text-muted-foreground">Active seafarers in {selectedProgram}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Mandatory Training</CardTitle>
              <Badge variant="destructive" className="h-4 w-4 rounded-full p-0" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {trainingPlan?.summary?.category && competencyMapping
                  ? Object.entries(trainingPlan.summary.category).filter(
                      ([code, cat]) => cat === "M" && competencyMapping[code]
                    ).length // Filter hanya competencies di program ini
                  : 0}
              </div>
              <p className="text-xs text-muted-foreground">
                Critical competencies in {selectedProgram}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Non Mandatory Training</CardTitle>
              <Badge className="h-4 w-4 rounded-full p-0 bg-blue-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {trainingPlan?.summary?.category && competencyMapping
                  ? Object.entries(trainingPlan.summary.category).filter(
                      ([code, cat]) => cat === "NM" && competencyMapping[code]
                    ).length // Filter hanya competencies di program ini
                  : 0}
              </div>
              <p className="text-xs text-muted-foreground">
                Non-critical competencies in {selectedProgram}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Avg Gap Score</CardTitle>
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {trainingPlan?.participants && trainingPlan.participants.length > 0
                  ? (
                      trainingPlan.participants.reduce((sum, p) => sum + p.total, 0) /
                      trainingPlan.participants.length
                    ).toFixed(2)
                  : "0.00"}
              </div>
              <p className="text-xs text-muted-foreground">Average competency gaps</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Mandatory Deadline</CardTitle>
              <Calendar className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {trainingPlan?.minDeadlineMonths || 0}{" "}
                {trainingPlan?.minDeadlineMonths === 1 ? "month" : "months"}
              </div>
              <p className="text-xs text-muted-foreground">All mandatory training must complete</p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Main Content Tabs */}
      <Tabs defaultValue="participants" className="space-y-4">
        <TabsList>
          <TabsTrigger value="participants">Participants</TabsTrigger>
          <TabsTrigger value="summary">Gap Summary</TabsTrigger>
          <TabsTrigger value="materials">Training Materials</TabsTrigger>
          <TabsTrigger value="mapping">Competency Mapping</TabsTrigger>
          <TabsTrigger value="schedules" className="relative">
            Training Plan
            {!!overdueCount && overdueCount > 0 && (
              <Badge
                variant="destructive"
                className="ml-2 h-4 w-4 rounded-full p-0 flex items-center justify-center text-[10px] font-medium"
              >
                {overdueCount}
              </Badge>
            )}
          </TabsTrigger>
        </TabsList>

        {/* Participants Tab */}
        <TabsContent value="participants" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Training Plan Participants</CardTitle>
              <CardDescription>
                Seafarers requiring training based on competency gap analysis
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Seaman Code</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Position</TableHead>
                      <TableHead>Vessel</TableHead>
                      <TableHead>Total Gaps</TableHead>
                      <TableHead>Readiness</TableHead>
                      <TableHead>Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {trainingPlan?.participants?.map((participant) => (
                      <TableRow key={participant.no}>
                        <TableCell className="font-medium">{participant.seamanCode}</TableCell>
                        <TableCell>{participant.name}</TableCell>
                        <TableCell>{participant.position || "-"}</TableCell>
                        <TableCell>{participant.vesselName}</TableCell>
                        <TableCell>
                          <Badge variant="outline">{participant.total}</Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant={getReadinessBadgeColor(participant.readiness)}>
                            {participant.readiness}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Dialog>
                            <DialogTrigger asChild>
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => setSelectedParticipant(participant)}
                              >
                                View Details
                              </Button>
                            </DialogTrigger>
                            <DialogContent className="max-w-2xl">
                              <DialogHeader>
                                <DialogTitle>{participant.name}</DialogTitle>
                                <DialogDescription>{participant.seamanCode}</DialogDescription>
                              </DialogHeader>
                              {selectedParticipant && (
                                <div className="space-y-4">
                                  <div className="grid grid-cols-2 gap-4">
                                    <div>
                                      <p className="text-sm font-medium">Vessel</p>
                                      <p className="text-sm text-muted-foreground">
                                        {selectedParticipant.vesselName}
                                      </p>
                                    </div>
                                    <div>
                                      <p className="text-sm font-medium">Total Gaps</p>
                                      <p className="text-sm text-muted-foreground">
                                        {selectedParticipant.total}
                                      </p>
                                    </div>
                                  </div>
                                  <div className="space-y-2">
                                    <p className="text-sm font-medium">Competency Gaps</p>
                                    <div className="grid grid-cols-3 gap-2">
                                      {Object.entries(selectedParticipant.gaps)
                                        .filter(
                                          ([, value]) =>
                                            value === "1" || value === "X" || value === 1
                                        )
                                        .map(([key]) => (
                                          <Badge key={key} variant="secondary" className="text-xs">
                                            {key.toUpperCase()}
                                          </Badge>
                                        ))}
                                    </div>
                                  </div>
                                </div>
                              )}
                            </DialogContent>
                          </Dialog>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Summary Tab */}
        <TabsContent value="summary" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Competency Gap Summary</CardTitle>
              <CardDescription>
                Overview of competency gaps for {selectedProgram} program competencies
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {trainingPlan?.summary?.total &&
                  competencyMapping &&
                  Object.entries(trainingPlan.summary.total)
                    .filter(([competencyCode]) => competencyMapping[competencyCode]) // Filter hanya competencies di program ini
                    .sort((a, b) => {
                      // Sort by percentage (descending)
                      const percentA = trainingPlan.summary.percentageGap?.[a[0]] || 0;
                      const percentB = trainingPlan.summary.percentageGap?.[b[0]] || 0;
                      return percentB - percentA;
                    })
                    .map(([competencyCode, totalParticipants]) => {
                      // Get participants with this gap
                      const participantsWithGap = trainingPlan.participants.filter(
                        (p) =>
                          p.gaps[competencyCode] === "1" ||
                          p.gaps[competencyCode] === "X" ||
                          p.gaps[competencyCode] === 1
                      );

                      return (
                        <div key={competencyCode} className="space-y-2">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              <span className="font-medium">{competencyCode}</span>
                              <span className="text-sm text-muted-foreground">
                                - {competencyMapping[competencyCode]?.name || competencyCode}
                              </span>
                              <Badge
                                variant={getCompetencyBadgeColor(
                                  trainingPlan.summary.category?.[competencyCode] || ""
                                )}
                                className={
                                  trainingPlan.summary.category?.[competencyCode] === "NM"
                                    ? "bg-blue-500 text-white hover:bg-blue-600"
                                    : ""
                                }
                              >
                                {trainingPlan.summary.category?.[competencyCode] === "M"
                                  ? "Mandatory"
                                  : "Non-Mandatory"}
                              </Badge>
                            </div>
                            <div className="flex items-center gap-2">
                              <Dialog>
                                <DialogTrigger asChild>
                                  <Button
                                    variant="link"
                                    className="text-sm text-muted-foreground hover:text-primary p-0"
                                  >
                                    {totalParticipants} participants
                                  </Button>
                                </DialogTrigger>
                                <DialogContent className="max-w-3xl">
                                  <DialogHeader>
                                    <DialogTitle>
                                      {competencyCode} -{" "}
                                      {competencyMapping[competencyCode]?.name || competencyCode}
                                    </DialogTitle>
                                    <DialogDescription>
                                      Participants with this competency gap
                                    </DialogDescription>
                                  </DialogHeader>
                                  <div className="max-h-[400px] overflow-y-auto">
                                    <Table>
                                      <TableHeader>
                                        <TableRow>
                                          <TableHead>Seaman Code</TableHead>
                                          <TableHead>Name</TableHead>
                                          <TableHead>Position</TableHead>
                                          <TableHead>Vessel</TableHead>
                                        </TableRow>
                                      </TableHeader>
                                      <TableBody>
                                        {participantsWithGap.map((participant) => (
                                          <TableRow key={participant.no}>
                                            <TableCell className="font-medium">
                                              {participant.seamanCode}
                                            </TableCell>
                                            <TableCell>{participant.name}</TableCell>
                                            <TableCell>{participant.position || "-"}</TableCell>
                                            <TableCell>{participant.vesselName}</TableCell>
                                          </TableRow>
                                        ))}
                                      </TableBody>
                                    </Table>
                                  </div>
                                </DialogContent>
                              </Dialog>
                              <span className="text-sm font-medium">
                                {trainingPlan.summary.percentageGap?.[competencyCode]?.toFixed(1) ||
                                  0}
                                %
                              </span>
                            </div>
                          </div>
                          <Progress
                            value={trainingPlan.summary.percentageGap?.[competencyCode] || 0}
                            className="h-2"
                          />
                        </div>
                      );
                    })}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Competency Mapping Tab */}
        <TabsContent value="mapping" className="space-y-4">
          <Card>
            <CardContent>
              <CompetencyMappingCMS program={selectedProgram} trainingPlan={trainingPlan} />
            </CardContent>
          </Card>
        </TabsContent>

        {/* Training Materials Tab */}
        <TabsContent value="materials" className="space-y-4">
          <TrainingMaterialTab />
        </TabsContent>

        {/* Schedules Tab */}
        <TabsContent value="schedules" className="space-y-4">
          <Card>
            <CardHeader className="space-y-1 pb-4">
              <CardTitle>Training Plan Timeline</CardTitle>
              <CardDescription>
                Visual timeline of training schedules for {selectedProgram}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {trainingPlan?.summary &&
              trainingPlan.summary.trainingMateri1 &&
              trainingPlan.summary.trainingMateri2 &&
              (Object.keys(trainingPlan.summary.trainingMateri1).length > 0 ||
                Object.keys(trainingPlan.summary.trainingMateri2).length > 0) ? (
                <TrainingScheduleTimeline
                  summary={trainingPlan.summary}
                  program={selectedProgram}
                  competencyMapping={competencyMapping}
                  onExportExcel={handleExportExcel}
                />
              ) : (
                <div className="text-center py-8">
                  <Calendar className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                  <h3 className="text-lg font-medium text-muted-foreground mb-2">
                    No Schedules Generated
                  </h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    Click &quot;Generate Schedules&quot; to create training schedules for this
                    program.
                  </p>
                  <Button
                    onClick={handleGenerateSchedules}
                    disabled={generateSchedules.isPending}
                    className="gap-2"
                  >
                    {generateSchedules.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Calendar className="h-4 w-4" />
                    )}
                    Generate Schedules
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Generate Schedules Warning Dialog */}
      <AlertDialog open={showGenerateDialog} onOpenChange={setShowGenerateDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2 text-destructive">
              <AlertTriangle className="h-5 w-5" />
              Critical Warning
            </AlertDialogTitle>
            <AlertDialogDescription asChild>
              <div className="space-y-4">
                <p className="font-semibold text-foreground">
                  Generating schedules will DELETE all existing schedules for this program!
                </p>
                <p>
                  This action cannot be undone. All previously generated training schedules for the{" "}
                  <span className="font-medium">{selectedProgram}</span> program will be permanently
                  removed.
                </p>
                <div className="space-y-2 pt-2">
                  <Label htmlFor="start-date">Start Date (Optional)</Label>
                  <Input
                    id="start-date"
                    type="date"
                    value={startDate}
                    onChange={(e) => setStartDate(e.target.value)}
                    placeholder="YYYY-MM-DD"
                    className="w-full"
                  />
                  <p className="text-xs text-muted-foreground">
                    Leave empty to use default start date (October 1, 2025)
                  </p>
                </div>
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmGenerate}
              className="bg-destructive hover:bg-destructive/90"
            >
              Generate Schedules
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
