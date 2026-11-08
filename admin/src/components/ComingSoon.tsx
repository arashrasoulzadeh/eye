import { Card } from "./Card";

export function ComingSoon({ title, description }: { title: string; description: string }) {
  return (
    <div>
      <h1 className="text-xl font-semibold text-fg">{title}</h1>
      <Card className="mt-6 flex flex-col items-center gap-2 px-6 py-16 text-center">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-surface-2 text-fg-muted">◔</div>
        <p className="text-sm font-medium text-fg">Not implemented yet</p>
        <p className="max-w-sm text-sm text-fg-muted">{description}</p>
      </Card>
    </div>
  );
}
