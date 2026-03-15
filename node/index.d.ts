declare module "laminargate" {
    export class AdaptiveLimiter {
        constructor(targetLatencyMs?: number, maxRate?: number, initialRate?: number, kp?: number, kd?: number, intervalMs?: number, emaAlpha?: number);
        allow(): boolean;
        recordLatency(latencyMs: number): void;
        currentRate(): number;
        stop(): void;
    }
}
