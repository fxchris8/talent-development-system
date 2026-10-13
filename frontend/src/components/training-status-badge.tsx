// Status seseorang di program IDP-nya (kolom reports.training_status di backend).
//   Ongoing   - masih menjalani program
//   Completed - semua training program sudah dilakukan
//   Exited    - naik jabatan sebelum selesai, otomatis keluar dari program

interface TrainingStatusBadgeProps {
  status?: string | null;
  exitedProgram?: string | null;
}

const STYLES: Record<string, string> = {
  Ongoing: "bg-blue-100 text-blue-800",
  Completed: "bg-green-100 text-green-800",
  Exited: "bg-amber-100 text-amber-800",
};

export function trainingStatusLabel(status?: string | null, exitedProgram?: string | null) {
  switch (status) {
    case "Ongoing":
      return "Ongoing";
    case "Completed":
      return "Completed";
    case "Exited":
      return exitedProgram ? `Keluar dari ${exitedProgram}` : "Keluar";
    default:
      return "-";
  }
}

export function TrainingStatusBadge({ status, exitedProgram }: TrainingStatusBadgeProps) {
  const style = status ? STYLES[status] : undefined;
  if (!style) return <span>-</span>;

  return (
    <span
      className={`inline-block whitespace-nowrap rounded-full px-2 py-0.5 text-xs font-medium ${style}`}
      title={status === "Exited" ? "Naik jabatan, otomatis keluar dari program" : undefined}
    >
      {trainingStatusLabel(status, exitedProgram)}
    </span>
  );
}
