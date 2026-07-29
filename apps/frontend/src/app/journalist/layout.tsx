import { DashboardNav } from "@/components/DashboardNav";
import { RoleGate } from "@/components/RoleGate";

export default function JournalistLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <RoleGate role="journalist">
      <div className="app-shell">
        <DashboardNav role="journalist" />
        <main className="app-main">{children}</main>
      </div>
    </RoleGate>
  );
}
