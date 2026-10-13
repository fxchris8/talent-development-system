import React, { useState, useEffect, useMemo } from "react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  ChevronDown,
  ChevronRight,
  Save,
  RotateCcw,
  Edit,
  Download,
  PlayCircle,
  AlertTriangle,
  History,
} from "lucide-react";
import { toast } from "sonner";
import { useSwapSchedules, useToggleTrainingStarted } from "./_hooks/useTrainingPlan";
import type { TrainingPlanSummary } from "./_hooks/useTrainingPlan";
import CourseNameDialog from "./CourseNameDialog";
import {
  buildTimelineLayout,
  describeDelay,
  getWeekLabel,
  toCalendarDayIso,
  weekEnd,
} from "./_hooks/timelineLayout";

interface TrainingScheduleTimelineProps {
  summary: TrainingPlanSummary;
  program: string;
  competencyMapping?: {
    [key: string]: {
      name: string;
      training_topics: string[];
    };
  };
  onExportExcel?: () => void;
}

interface ScheduleItem {
  id: number;
  competencyCode: string;
  competencyName: string;
  trainingMaterial: string;
  category: string;
  materialType: 1 | 2;
  scheduledDate: Date;
  isStarted: boolean;
}

interface WeekSlot {
  weekLabel: string;
  weekNumber: 1 | 2 | 3 | 4;
  date: Date;
}

const formatMonth = (date: Date): string =>
  date.toLocaleDateString("id-ID", { month: "long", year: "numeric" });

export default function TrainingScheduleTimeline({
  summary,
  program,
  competencyMapping,
  onExportExcel,
}: TrainingScheduleTimelineProps) {
  const [collapsedMonths, setCollapsedMonths] = useState<Set<string>>(new Set());
  const [schedules, setSchedules] = useState<ScheduleItem[]>([]);
  const [originalSchedules, setOriginalSchedules] = useState<ScheduleItem[]>([]);
  const [draggedItem, setDraggedItem] = useState<ScheduleItem | null>(null);
  const [dragOverWeek, setDragOverWeek] = useState<string | null>(null);
  const [editMode, setEditMode] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [postponedCollapsed, setPostponedCollapsed] = useState(true);
  const [toggleLoading, setToggleLoading] = useState<number | null>(null);
  const [showCourseNameDialog, setShowCourseNameDialog] = useState(false);
  const [selectedSchedule, setSelectedSchedule] = useState<ScheduleItem | null>(null);
  const swapSchedules = useSwapSchedules();
  const toggleTrainingStarted = useToggleTrainingStarted();

  const toggleMonth = (monthKey: string) => {
    const newCollapsed = new Set(collapsedMonths);
    if (newCollapsed.has(monthKey)) {
      newCollapsed.delete(monthKey);
    } else {
      newCollapsed.add(monthKey);
    }
    setCollapsedMonths(newCollapsed);
  };

  const getCompetencyName = (competencyCode: string): string => {
    return competencyMapping?.[competencyCode]?.name || competencyCode;
  };

  const getTrainingMaterial = (competencyCode: string, materialType: 1 | 2): string => {
    const topics = competencyMapping?.[competencyCode]?.training_topics;
    if (!topics || topics.length === 0) return "Training Material";
    return topics[materialType - 1] || topics[0] || "Training Material";
  };

  const parseSchedules = (): ScheduleItem[] => {
    const schedulesArray: ScheduleItem[] = [];

    // Backend now sends ISO timestamps (2025-10-22T00:00:00Z) instead of "I-10" format
    const allEntries: Array<{ code: string; dateStr: string; type: 1 | 2 }> = [
      ...Object.entries(summary.trainingMateri1).map(([code, dateStr]) => ({
        code,
        dateStr,
        type: 1 as 1 | 2,
      })),
      ...Object.entries(summary.trainingMateri2).map(([code, dateStr]) => ({
        code,
        dateStr,
        type: 2 as 1 | 2,
      })),
    ];

    allEntries.forEach((entry) => {
      if (!entry.dateStr || entry.dateStr === "-") return;

      // Parse ISO timestamp directly from database
      const date = new Date(entry.dateStr);
      if (isNaN(date.getTime())) return; // Skip invalid dates

      const scheduleId = summary.scheduleIds?.[entry.code]?.[entry.type.toString()] || 0;
      const isStarted = summary.isStartedStatus?.[entry.code]?.[entry.type.toString()] || false;

      schedulesArray.push({
        id: scheduleId,
        competencyCode: entry.code,
        competencyName: getCompetencyName(entry.code),
        trainingMaterial: getTrainingMaterial(entry.code, entry.type),
        category: summary.category[entry.code] || "NM",
        materialType: entry.type,
        scheduledDate: date,
        isStarted: isStarted,
      });
    });

    return schedulesArray.sort((a, b) => a.scheduledDate.getTime() - b.scheduledDate.getTime());
  };

  useEffect(() => {
    const parsedSchedules = parseSchedules();
    setSchedules(parsedSchedules);
    setOriginalSchedules(
      parsedSchedules.map((s) => ({ ...s, scheduledDate: new Date(s.scheduledDate) }))
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [summary, competencyMapping]);

  // Timeline = bulan ini ke depan. Yang belum dimulai dan minggunya sudah lewat dikumpulkan
  // di blok "Tertunda" paling atas; bulan lampau yang sudah berjalan masuk "riwayat".
  const layout = useMemo(() => buildTimelineLayout(schedules, new Date()), [schedules]);
  const postponedIds = useMemo(
    () => new Set(layout.postponed.map((s) => s.id)),
    [layout.postponed]
  );

  const getWeekSlotsForMonth = (month: Date): WeekSlot[] => {
    return [
      { weekLabel: "I", weekNumber: 1, date: new Date(month.getFullYear(), month.getMonth(), 1) },
      { weekLabel: "II", weekNumber: 2, date: new Date(month.getFullYear(), month.getMonth(), 8) },
      {
        weekLabel: "III",
        weekNumber: 3,
        date: new Date(month.getFullYear(), month.getMonth(), 15),
      },
      { weekLabel: "IV", weekNumber: 4, date: new Date(month.getFullYear(), month.getMonth(), 22) },
    ];
  };

  // Jadwal pada satu minggu, tanpa yang sudah dipindah ke blok "Tertunda".
  const getSchedulesForWeek = (weekSlot: WeekSlot): ScheduleItem[] => {
    return schedules.filter(
      (schedule) =>
        !postponedIds.has(schedule.id) &&
        getWeekLabel(schedule.scheduledDate) === weekSlot.weekLabel &&
        schedule.scheduledDate.getMonth() === weekSlot.date.getMonth() &&
        schedule.scheduledDate.getFullYear() === weekSlot.date.getFullYear()
    );
  };

  const countForMonth = (month: Date): number =>
    schedules.filter(
      (s) =>
        !postponedIds.has(s.id) &&
        s.scheduledDate.getMonth() === month.getMonth() &&
        s.scheduledDate.getFullYear() === month.getFullYear()
    ).length;

  const getWeekKey = (month: Date, weekLabel: string): string => {
    return `${month.getFullYear()}-${month.getMonth()}-${weekLabel}`;
  };

  const handleDragStart = (e: React.DragEvent, item: ScheduleItem) => {
    setDraggedItem(item);
    e.dataTransfer.effectAllowed = "move";
  };

  const handleDragOverWeek = (e: React.DragEvent, weekKey: string) => {
    e.preventDefault();
    setDragOverWeek(weekKey);
  };

  const handleDragLeave = () => {
    setDragOverWeek(null);
  };

  const handleDropOnWeek = (e: React.DragEvent, weekSlot: WeekSlot) => {
    e.preventDefault();

    if (!draggedItem) {
      setDraggedItem(null);
      setDragOverWeek(null);
      return;
    }

    const newSchedules = schedules.map((s) =>
      s.id === draggedItem.id ? { ...s, scheduledDate: new Date(weekSlot.date) } : s
    );
    newSchedules.sort((a, b) => a.scheduledDate.getTime() - b.scheduledDate.getTime());

    setSchedules(newSchedules);
    setDraggedItem(null);
    setDragOverWeek(null);
  };

  const handleDragEnd = () => {
    setDraggedItem(null);
    setDragOverWeek(null);
  };

  const hasChanges = () => {
    return (
      JSON.stringify(schedules.map((s) => ({ id: s.id, date: s.scheduledDate.getTime() }))) !==
      JSON.stringify(originalSchedules.map((s) => ({ id: s.id, date: s.scheduledDate.getTime() })))
    );
  };

  const handleSave = async () => {
    const swaps = schedules
      .filter((schedule) => {
        const original = originalSchedules.find((o) => o.id === schedule.id);
        return original && original.scheduledDate.getTime() !== schedule.scheduledDate.getTime();
      })
      .map((schedule) => ({
        id: schedule.id,
        scheduledDate: toCalendarDayIso(schedule.scheduledDate),
      }));

    if (swaps.length === 0) {
      toast.info("No changes to save");
      return;
    }

    try {
      await swapSchedules.mutateAsync({ swaps, program });
      setOriginalSchedules(
        schedules.map((s) => ({ ...s, scheduledDate: new Date(s.scheduledDate) }))
      );
      setEditMode(false);
      toast.success("Schedules saved successfully!");
    } catch {
      toast.error("Failed to save schedules");
    }
  };

  const handleReset = () => {
    setSchedules(
      originalSchedules.map((s) => ({ ...s, scheduledDate: new Date(s.scheduledDate) }))
    );
    setEditMode(false);
    toast.info("Changes reset");
  };

  const handleToggleStarted = async (scheduleId: number, currentStatus: boolean) => {
    const schedule = schedules.find((s) => s.id === scheduleId);
    if (!schedule) return;

    if (!currentStatus) {
      setSelectedSchedule(schedule);
      setShowCourseNameDialog(true);
    } else {
      setToggleLoading(scheduleId);
      try {
        await toggleTrainingStarted.mutateAsync({
          scheduleId,
          isStarted: false,
          apolloCourseName: "",
        });

        setSchedules((prev) =>
          prev.map((s) => (s.id === scheduleId ? { ...s, isStarted: false } : s))
        );
        setOriginalSchedules((prev) =>
          prev.map((s) => (s.id === scheduleId ? { ...s, isStarted: false } : s))
        );

        toast.success("Training stopped!");
      } catch {
        toast.error("Failed to toggle training status");
      } finally {
        setToggleLoading(null);
      }
    }
  };

  const handleCourseNameSubmit = async (courseName: string) => {
    if (!selectedSchedule) return;

    setToggleLoading(selectedSchedule.id);
    try {
      await toggleTrainingStarted.mutateAsync({
        scheduleId: selectedSchedule.id,
        isStarted: true,
        apolloCourseName: courseName,
      });

      setSchedules((prev) =>
        prev.map((s) => (s.id === selectedSchedule.id ? { ...s, isStarted: true } : s))
      );
      setOriginalSchedules((prev) =>
        prev.map((s) => (s.id === selectedSchedule.id ? { ...s, isStarted: true } : s))
      );

      toast.success("Training started!");
      setShowCourseNameDialog(false);
      setSelectedSchedule(null);
    } catch {
      toast.error("Failed to start training");
    } finally {
      setToggleLoading(null);
    }
  };

  if (schedules.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        No training schedules available for {program}
      </div>
    );
  }

  const now = new Date();
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate());

  const renderSchedule = (schedule: ScheduleItem, postponed: boolean) => (
    <div
      key={schedule.id}
      draggable={editMode}
      onDragStart={(e) => editMode && handleDragStart(e, schedule)}
      onDragEnd={editMode ? handleDragEnd : undefined}
      className={`flex items-center gap-3 p-2 rounded-md transition-all ${
        editMode
          ? `cursor-move ${
              draggedItem?.id === schedule.id
                ? "opacity-50 bg-blue-200"
                : "bg-white hover:bg-gray-50 shadow-sm"
            }`
          : "bg-white shadow-sm"
      } ${postponed ? "border border-red-300" : ""}`}
    >
      {postponed && <AlertTriangle className="h-4 w-4 text-red-500 flex-shrink-0" />}
      <Badge
        variant={schedule.category === "M" ? "destructive" : "secondary"}
        className={`font-medium ${schedule.category !== "M" ? "bg-blue-500 text-white hover:bg-blue-600" : ""}`}
      >
        {schedule.category}
      </Badge>
      <div className="flex-1 min-w-0">
        <div className="font-medium text-sm">{schedule.trainingMaterial}</div>
        <div className="text-xs text-muted-foreground">
          {schedule.competencyCode} - {schedule.competencyName}
        </div>
        {postponed && (
          <div className="text-xs text-red-600 mt-0.5">
            Jadwal semula: Week {getWeekLabel(schedule.scheduledDate)},{" "}
            {formatMonth(schedule.scheduledDate)} · terlambat {describeDelay(schedule, now)}
          </div>
        )}
      </div>
      <div className="flex items-center gap-1.5 flex-shrink-0">
        <PlayCircle
          className={`h-3.5 w-3.5 ${schedule.isStarted ? "text-green-600" : "text-gray-400"}`}
        />
        <Switch
          checked={schedule.isStarted}
          onCheckedChange={() => handleToggleStarted(schedule.id, schedule.isStarted)}
          disabled={toggleLoading === schedule.id}
          className="scale-75"
        />
      </div>
    </div>
  );

  // `droppable` = minggu boleh jadi tujuan penjadwalan ulang (bulan riwayat tidak).
  const renderMonth = (month: Date, droppable: boolean) => {
    const monthKey = `${month.getFullYear()}-${month.getMonth()}`;
    const isCollapsed = collapsedMonths.has(monthKey);
    const total = countForMonth(month);

    return (
      <Card key={monthKey} className={`overflow-hidden ${droppable ? "" : "opacity-80"}`}>
        <CardContent className="p-0">
          <div
            className="bg-muted p-4 border-b cursor-pointer hover:bg-muted/80 transition-colors"
            onClick={() => toggleMonth(monthKey)}
          >
            <div className="flex items-center justify-between">
              <h4 className="font-medium">
                {formatMonth(month)}
                {!droppable && (
                  <span className="ml-2 text-xs font-normal text-muted-foreground">(riwayat)</span>
                )}
              </h4>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="text-xs">
                  {total} training{total !== 1 ? "s" : ""}
                </Badge>
                {isCollapsed ? (
                  <ChevronRight className="h-4 w-4" />
                ) : (
                  <ChevronDown className="h-4 w-4" />
                )}
              </div>
            </div>
          </div>

          {!isCollapsed && (
            <div className="p-4">
              <div className="space-y-3">
                {getWeekSlotsForMonth(month).map((weekSlot) => {
                  const weekSchedules = getSchedulesForWeek(weekSlot);
                  const weekKey = getWeekKey(month, weekSlot.weekLabel);
                  // Minggu yang sudah lewat bukan tujuan penjadwalan ulang: jadwal yang dijatuhkan
                  // di sana akan langsung dianggap tertunda lagi.
                  const weekPassed = weekEnd(weekSlot.date).getTime() <= startOfToday.getTime();
                  const canDrop = editMode && droppable && !weekPassed;
                  const isWeekDragOver = dragOverWeek === weekKey;

                  return (
                    <div
                      key={weekSlot.weekLabel}
                      onDragOver={(e) => canDrop && handleDragOverWeek(e, weekKey)}
                      onDragLeave={canDrop ? handleDragLeave : undefined}
                      onDrop={(e) => canDrop && handleDropOnWeek(e, weekSlot)}
                      className={`min-h-[60px] p-3 rounded-lg border-2 transition-all ${
                        canDrop
                          ? isWeekDragOver
                            ? "border-blue-400 bg-blue-50"
                            : "border-dashed border-gray-300 bg-gray-50/50"
                          : "border-transparent bg-muted/30"
                      }`}
                    >
                      <div className="flex items-start gap-3">
                        <div className="flex-shrink-0 w-12 text-center pt-1">
                          <div className="text-lg font-bold">{weekSlot.weekLabel}</div>
                          <div className="text-xs text-muted-foreground">Week</div>
                        </div>

                        <div className="flex-1 space-y-2">
                          {weekSchedules.length > 0 ? (
                            weekSchedules.map((schedule) => renderSchedule(schedule, false))
                          ) : (
                            <div className="text-xs text-muted-foreground italic py-2">
                              {canDrop
                                ? "Drop training here"
                                : weekPassed && droppable
                                  ? "Minggu sudah lewat"
                                  : "No training scheduled"}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          {!editMode && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setEditMode(true);
                // Rescheduling means dragging postponed items, so show them
                setPostponedCollapsed(false);
              }}
            >
              <Edit className="h-4 w-4 mr-2" />
              Edit Schedules
            </Button>
          )}
          {editMode && hasChanges() && (
            <>
              <Button variant="outline" size="sm" onClick={handleReset}>
                <RotateCcw className="h-4 w-4 mr-2" />
                Reset
              </Button>
              <Button size="sm" onClick={handleSave} disabled={swapSchedules.isPending}>
                <Save className="h-4 w-4 mr-2" />
                {swapSchedules.isPending ? "Saving..." : "Save Changes"}
              </Button>
            </>
          )}
          {editMode && !hasChanges() && (
            <Button variant="outline" size="sm" onClick={() => setEditMode(false)}>
              Cancel
            </Button>
          )}
          {layout.historyMonths.length > 0 && (
            <Button variant="ghost" size="sm" onClick={() => setShowHistory((v) => !v)}>
              <History className="h-4 w-4 mr-2" />
              {showHistory ? "Sembunyikan riwayat" : "Tampilkan riwayat"}
            </Button>
          )}
        </div>
        {onExportExcel && !editMode && (
          <Button onClick={onExportExcel} size="sm" className="gap-2">
            <Download className="h-4 w-4" />
            Export to Excel
          </Button>
        )}
      </div>

      {layout.postponed.length > 0 && (
        <Card className="overflow-hidden border-red-300 bg-red-50 py-0 gap-0">
          <CardContent className="p-0">
            <div
              className={`bg-red-50 p-4 cursor-pointer ${postponedCollapsed ? "" : "border-b border-red-200"}`}
              onClick={() => setPostponedCollapsed((v) => !v)}
            >
              <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="h-5 w-5 text-red-600 flex-shrink-0" />
                  <div>
                    <h4 className="font-semibold text-red-800">Tertunda / Belum Dilaksanakan</h4>
                    <p className="text-xs text-red-700">
                      Jadwalnya sudah lewat tetapi training belum dimulai.{" "}
                      {editMode
                        ? "Seret ke salah satu minggu di bawah untuk menjadwal ulang."
                        : "Mulai trainingnya, atau klik Edit Schedules untuk menjadwal ulang."}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <Badge variant="destructive" className="text-xs">
                    {layout.postponed.length} tertunda
                  </Badge>
                  {postponedCollapsed ? (
                    <ChevronRight className="h-4 w-4 text-red-700" />
                  ) : (
                    <ChevronDown className="h-4 w-4 text-red-700" />
                  )}
                </div>
              </div>
            </div>
            {!postponedCollapsed && (
              <div className="p-4 space-y-2 bg-red-50 max-h-[28rem] overflow-y-auto">
                {layout.postponed.map((schedule) => renderSchedule(schedule, true))}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {showHistory && layout.historyMonths.length > 0 && (
        <div className="space-y-4">{layout.historyMonths.map((m) => renderMonth(m, false))}</div>
      )}

      <div className="space-y-4">{layout.months.map((month) => renderMonth(month, true))}</div>

      <CourseNameDialog
        open={showCourseNameDialog}
        onOpenChange={(open) => {
          setShowCourseNameDialog(open);
          if (!open) setSelectedSchedule(null);
        }}
        onSubmit={handleCourseNameSubmit}
        isLoading={toggleLoading !== null}
        competencyCode={selectedSchedule?.competencyCode || ""}
        trainingMaterial={selectedSchedule?.trainingMaterial || ""}
      />
    </div>
  );
}
