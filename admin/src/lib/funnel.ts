export function formatEventLabel(name: unknown, titles: Record<string, string> = {}) {
  const value = typeof name === "string" && name.trim() ? name : "Unknown event";
  return titles[value] || String(value).replace(/[_.-]+/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

export function funnelTransition(previousSessions: number, nextSessions: number) {
  return {
    dropped: previousSessions - nextSessions,
    rate: previousSessions > 0 ? nextSessions / previousSessions : null,
  };
}

export type PinnedFunnel = { name: string; steps: string[] };

export function previousPeriod(from: number, to: number) {
  return { from_ts: from - (to - from + 1), to_ts: from - 1 };
}

export function funnelSummary(stages: readonly { sessions: number }[]) {
  const entered = stages[0]?.sessions ?? 0;
  const completed = stages.at(-1)?.sessions ?? 0;
  const transitions = stages.slice(1).map((stage, index) =>
    funnelTransition(stages[index].sessions, stage.sessions),
  );
  let biggestDrop: { index: number; dropped: number; rate: number } | null = null;
  for (const [index, { rate, dropped }] of transitions.entries()) {
    if (rate !== null && rate < (biggestDrop?.rate ?? 1)) {
      biggestDrop = { index, dropped, rate };
    }
  }
  return { entered, completed, rate: entered ? completed / entered : null, transitions, biggestDrop };
}
