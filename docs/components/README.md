# Admin Panel Components

React components used in the Next.js admin dashboard.

## Core Components

### `Card` / `CardHeader` / `StatCard`

```tsx
import { Card, CardHeader, StatCard } from '@/components/Card';

<Card className="mt-6">
  <CardHeader title="Events by Hour" />
  <div className="p-5">
    <LineTrend data={eventsByHour} />
  </div>
</Card>

<StatCard label="Events" value={1234} />
```

**Props:**
- `Card`: `className?`, `children`
- `CardHeader`: `title: string`
- `StatCard`: `label: string`, `value: string|number`

### `DataTable`

```tsx
import { DataTable, Column } from '@/components/DataTable';

const columns: Column<Event>[] = [
  { key: 'name', header: 'Event', render: (e) => <span>{e.EventName}</span> },
  { key: 'time', header: 'Time', render: (e) => e.IngestTS ? new Date(e.IngestTS).toLocaleString() : '—', sortValue: (e) => e.IngestTS },
];

<DataTable 
  columns={columns} 
  rows={events} 
  rowKey={(e) => e.EventID}
  emptyMessage="No events"
/>
```

**Props:**
- `columns: Column<T>[]` - Column definitions
- `rows: T[]` - Data rows
- `rowKey: (row: T) => string` - Unique key function
- `emptyMessage?: string`
- `className?: string`

### `Chart` - `LineTrend` / `BarBreakdown`

```tsx
import { LineTrend, BarBreakdown, ChartPoint } from '@/components/Chart';

const data: ChartPoint[] = [
  { label: '00:00', value: 10 },
  { label: '01:00', value: 15 },
];

<LineTrend data={data} height={220} />
<BarBreakdown data={data} height={220} />
```

**Props:**
- `data: ChartPoint[]` - Array of `{ label: string; value: number }`
- `height?: number` - Default: 220

Handles empty/undefined data gracefully.

### `Badge`

```tsx
import { Badge } from '@/components/Badge';

<Badge tone="accent">client</Badge>
<Badge tone="warning">system</Badge>
<Badge tone="danger">error</Badge>
<Badge tone="success">ok</Badge>
<Badge tone="neutral">pending</Badge>
```

### `Button`

```tsx
import { Button } from '@/components/Button';

<Button onClick={handleClick} variant="primary">Save</Button>
<Button variant="secondary">Cancel</Button>
<Button variant="ghost">Delete</Button>
<Button disabled>Loading...</Button>
```

### `Input`

```tsx
import { Input } from '@/components/Input';

<Input 
  value={text} 
  onChange={setText} 
  placeholder="Search..."
  className="w-64"
/>
```

### `Combobox` (Searchable Select)

```tsx
import { Combobox } from '@/components/Combobox';

<Combobox
  options={[
    { value: 't1', label: 'Tenant 1' },
    { value: 't2', label: 'Tenant 2' },
  ]}
  value={selected}
  onChange={setSelected}
  placeholder="Select tenant"
/>
```

### `JsonView`

```tsx
import { JsonView } from '@/components/JsonView';

<JsonView value={event.Payload} />
<JsonView value={event.Dims} maxDepth={2} />
```

Collapsible JSON viewer with syntax highlighting.

### `ConfirmDialog`

```tsx
import { ConfirmDialog } from '@/components/ConfirmDialog';

<ConfirmDialog
  open={showDialog}
  onClose={() => setShowDialog(false)}
  onConfirm={handleDelete}
  title="Delete Tenant?"
  description="This action cannot be undone."
  confirmText="Delete"
  variant="danger"
/>
```

### `Toast` / `useToast`

```tsx
import { useToast, Toast } from '@/components/Toast';

function MyComponent() {
  const { push } = useToast();
  
  push('success', 'Saved!');
  push('error', 'Failed to save');
  push('info', 'Processing...');
  
  return <Toast />;
}
```

### `EntityLink`

```tsx
import { EntityLink } from '@/components/EntityLink';

<EntityLink kind="tenant" id="t1" />
<EntityLink kind="session" id="s1" />
<EntityLink kind="event" id="e1" />
```

Renders a link to the appropriate admin page.

### `LiveSession`

```tsx
import { LiveSession } from '@/components/LiveSession';

<LiveSession session={session} className="mt-6" />
```

Real-time session event count + last seen via SSE.
Shows connection status: connecting/live/disconnected.

### `LiveCounter`

```tsx
import { LiveCounter } from '@/components/LiveCounter';

<LiveCounter tenantId="t1" className="mt-6" />
```

Real-time tenant event count via SSE.

### `Sidebar`

```tsx
import { Sidebar } from '@/components/Sidebar';

<Sidebar 
  currentPath={pathname}
  navItems={[
    { href: '/dashboard', label: 'Dashboard', icon: HomeIcon },
    { href: '/tenants', label: 'Tenants', icon: UsersIcon },
  ]}
/>
```

## Layout Components

### `(dashboard)/layout.tsx`

Wraps all dashboard pages with sidebar and header.

### `globals.css`

CSS variables for theming:
```css
:root {
  --color-fg: #111827;
  --color-fg-muted: #6b7280;
  --color-accent: #6366f1;
  --color-border: #e5e7eb;
  --color-danger: #ef4444;
  --color-success: #22c55e;
  --color-warning: #f59e0b;
}
```

## Hooks

### `useToast()`

Returns `{ push: (type: 'success'|'error'|'info', message: string) => void }`

## Testing

```bash
cd admin
npm test  # vitest + React Testing Library
```

Component tests in `**/*.test.tsx`:
- `Card.test.tsx`
- `Button.test.tsx`
- `Combobox.test.tsx`
- `DataTable.test.tsx`
- `Chart.test.tsx`
- `Toast.test.tsx`
- `ConfirmDialog.test.tsx`
- `SessionDetailClient.test.tsx`
- `TenantDetailClient.test.tsx`
- `cohorts/page.test.tsx`
- `alerts/page.test.tsx`
- `settings/page.test.tsx`