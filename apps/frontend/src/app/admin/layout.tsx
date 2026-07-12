import { DashboardNav } from "@/components/DashboardNav";
import { RoleGate } from "@/components/RoleGate";

export default function AdminLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <RoleGate role="admin">
      <div className="app-shell">
        <DashboardNav role="admin" />
        <main className="app-main">{children}</main>
      </div>
    </RoleGate>
  );
}
