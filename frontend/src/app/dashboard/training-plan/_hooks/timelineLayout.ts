// Pure layout rules of the training schedule timeline (no React), so they can be tested.

export interface TimelineSchedule {
  id: number;
  scheduledDate: Date;
  isStarted: boolean;
}

export interface TimelineLayout<T extends TimelineSchedule> {
  // Not started and its week is already over: shown together at the top, oldest first.
  postponed: T[];
  // Months of the timeline: the current month up to the last scheduled month.
  months: Date[];
  // Months before the current one that contain trainings that did start (the "history").
  historyMonths: Date[];
}

const startOfDay = (date: Date): Date =>
  new Date(date.getFullYear(), date.getMonth(), date.getDate());

const startOfMonth = (date: Date): Date => new Date(date.getFullYear(), date.getMonth(), 1);

// A month has four week slots starting on day 1, 8, 15 and 22.
export const getWeekLabel = (date: Date): "I" | "II" | "III" | "IV" => {
  const day = date.getDate();
  if (day <= 7) return "I";
  if (day <= 14) return "II";
  if (day <= 21) return "III";
  return "IV";
};

// First day after the week slot the date belongs to.
export const weekEnd = (date: Date): Date => {
  const day = date.getDate();
  if (day <= 7) return new Date(date.getFullYear(), date.getMonth(), 8);
  if (day <= 14) return new Date(date.getFullYear(), date.getMonth(), 15);
  if (day <= 21) return new Date(date.getFullYear(), date.getMonth(), 22);
  return new Date(date.getFullYear(), date.getMonth() + 1, 1);
};

// A training is postponed once its whole week has passed without it being started.
export const isPostponed = (schedule: TimelineSchedule, now: Date): boolean =>
  !schedule.isStarted && weekEnd(schedule.scheduledDate).getTime() <= startOfDay(now).getTime();

export function buildTimelineLayout<T extends TimelineSchedule>(
  schedules: T[],
  now: Date
): TimelineLayout<T> {
  const currentMonth = startOfMonth(now);

  const postponed = schedules
    .filter((s) => isPostponed(s, now))
    .sort((a, b) => a.scheduledDate.getTime() - b.scheduledDate.getTime());
  const postponedIds = new Set(postponed.map((s) => s.id));

  // The timeline always starts at the current month, even when it has no training.
  let lastMonth = currentMonth;
  for (const s of schedules) {
    const month = startOfMonth(s.scheduledDate);
    if (month.getTime() > lastMonth.getTime()) lastMonth = month;
  }
  const months: Date[] = [];
  for (
    let m = new Date(currentMonth);
    m.getTime() <= lastMonth.getTime();
    m = new Date(m.getFullYear(), m.getMonth() + 1, 1)
  ) {
    months.push(m);
  }

  const historyKeys = new Set<number>();
  for (const s of schedules) {
    const month = startOfMonth(s.scheduledDate);
    if (month.getTime() < currentMonth.getTime() && !postponedIds.has(s.id)) {
      historyKeys.add(month.getTime());
    }
  }
  const historyMonths = [...historyKeys].sort((a, b) => a - b).map((t) => new Date(t));

  return { postponed, months, historyMonths };
}

// "2 minggu", "3 bulan" ... how long a postponed training has been waiting.
export function describeDelay(schedule: TimelineSchedule, now: Date): string {
  const days = Math.max(
    1,
    Math.round(
      (startOfDay(now).getTime() - weekEnd(schedule.scheduledDate).getTime()) / 86_400_000
    ) + 1
  );
  if (days < 7) return `${days} hari`;
  if (days < 30) return `${Math.floor(days / 7)} minggu`;
  if (days < 365) return `${Math.floor(days / 30)} bulan`;
  return `${Math.floor(days / 365)} tahun`;
}

// The backend stores a calendar day. Sending local midnight through toISOString() lands on
// the previous day for timezones ahead of UTC (WIB), so send the calendar day at UTC midnight.
export const toCalendarDayIso = (date: Date): string => {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}T00:00:00Z`;
};
