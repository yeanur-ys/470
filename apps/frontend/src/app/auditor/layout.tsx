import { DashboardNav } from "@/components/DashboardNav";
import { RoleGate } from "@/components/RoleGate";

export default function AuditorLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <RoleGate role="auditor">
      <div className="app-shell">
        <DashboardNav role="auditor" />
        <main className="app-main">{children}</main>
      </div>
    </RoleGate>
  );
}
