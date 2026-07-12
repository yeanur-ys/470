import { DashboardNav } from "@/components/DashboardNav";
import { RoleGate } from "@/components/RoleGate";

export default function AuditorLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <RoleGate role="auditor">
      <section style={{ maxWidth: 720, margin: "0 auto", padding: "1rem" }}>
        <h1>Auditor Workspace</h1>
        <DashboardNav />
        {children}
      </section>
    </RoleGate>
  );
}
