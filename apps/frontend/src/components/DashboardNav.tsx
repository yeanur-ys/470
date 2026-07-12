const links = ["journalist", "auditor", "admin"];

export function DashboardNav() {
  return (
    <nav aria-label="Dashboard roles">
      <ul>
        {links.map((link) => (
          <li key={link}>{link}</li>
        ))}
      </ul>
    </nav>
  );
}
