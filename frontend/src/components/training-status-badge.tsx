// Development status: where someone stands in their IDP program (reports.training_status).
//   Ongoing   - masih menjalani program            -> "Ongoing MDP"
//   Completed - semua training program sudah selesai -> "Completed MDP"
//   Exited    - sudah naik jabatan, keluar dari program -> "Completed MDP" (detail di tooltip
//               dan di Riwayat Keluar Program: lulus atau naik jabatan sebelum selesai)

interface TrainingStatusBadgeProps {
  status?: string | null;
  program?: string | null; // current IDP program
  exitedProgram?: string | null; // program left, for Exited
}

const STYLES: Record<string, string> = {
  Ongoing: "bg-blue-100 text-blue-800",
  Completed: "bg-green-100 text-green-800",
  Exited: "bg-green-100 text-green-800",
};

const HINTS: Record<string, string> = {
  Ongoing: "Masih menjalani program",
  Completed: "Semua training program sudah selesai",
  Exited: "Sudah naik jabatan dan keluar dari program. Lihat Riwayat Keluar Program di Dashboard.",
};

function statusLabel(status: string, program: string): string {
  const suffix = program ? ` ${program}` : "";
  return status === "Ongoing" ? `Ongoing${suffix}` : `Completed${suffix}`;
}

export function TrainingStatusBadge({ status, program, exitedProgram }: TrainingStatusBadgeProps) {
  const style = status ? STYLES[status] : undefined;
  if (!status || !style) return <span>-</span>;

  const shownProgram = ((status === "Exited" ? exitedProgram : program) || "").trim();

  return (
    <span
      className={`inline-block whitespace-nowrap rounded-full px-2 py-0.5 text-xs font-medium ${style}`}
      title={HINTS[status]}
    >
      {statusLabel(status, shownProgram)}
    </span>
  );
}
