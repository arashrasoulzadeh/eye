export type Event = {
    event_id: string;
    session_id: string;
    event_name: string;
    client_ts: number;
    payload: Record<string, unknown>;
    dims: Record<string, unknown>;
};
export type NodeClientOptions = {
    endpoint: string;
    apiKey: string;
    tenantId: string;
    queueFile?: string;
    maxQueueSize?: number;
    fetch?: typeof globalThis.fetch;
};
export type Diagnostics = {
    tenantId: string;
    sessionId: string | null;
    queuedEvents: number;
    droppedEvents: number;
    lastFlushAt: number | null;
    lastError: string | null;
};
export declare class EyeNodeClient {
    private readonly opts;
    private state;
    private loaded;
    private lock;
    constructor(options: NodeClientOptions);
    private withLock;
    private load;
    private persist;
    ensureSession(): Promise<string>;
    track(eventName: string, payload?: Record<string, unknown>, dims?: Record<string, unknown>): Promise<void>;
    flush(): Promise<number>;
    diagnostics(): Promise<Diagnostics>;
}
export default EyeNodeClient;
