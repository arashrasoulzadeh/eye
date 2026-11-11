export default function DashboardLoading() {
  return (
    <div className="space-y-5" aria-label="Loading admin page">
      <div className="h-7 w-48 animate-pulse rounded bg-surface-2" />
      <div className="h-4 w-96 max-w-full animate-pulse rounded bg-surface-2" />
      <div className="grid gap-4 md:grid-cols-3">
        {[1, 2, 3].map((item) => <div key={item} className="h-24 animate-pulse rounded-xl border border-border bg-surface-2" />)}
      </div>
      <div className="h-72 animate-pulse rounded-xl border border-border bg-surface-2" />
    </div>
  );
}
