const ffi = require('ffi-napi');
const path = require('path');

const libPath = path.join(__dirname, '..', 'ffi', 'ffi.so');

const lib = ffi.Library(libPath, {
    'CreateAdaptiveLimiter': ['int64', ['double', 'double', 'double', 'double', 'double', 'double', 'double']],
    'AdaptiveLimiterAllow': ['bool', ['int64']],
    'AdaptiveLimiterRecordLatency': ['void', ['int64', 'double']],
    'AdaptiveLimiterCurrentRate': ['double', ['int64']],
    'AdaptiveLimiterStop': ['void', ['int64']]
});

class AdaptiveLimiter {
    constructor(targetLatencyMs = 10.0, maxRate = 1000.0, initialRate = 500.0, kp = 0.5, kd = 0.1, intervalMs = 100.0, emaAlpha = 0.2) {
        this.id = lib.CreateAdaptiveLimiter(targetLatencyMs, maxRate, initialRate, kp, kd, intervalMs, emaAlpha);
    }

    allow() {
        return lib.AdaptiveLimiterAllow(this.id);
    }

    recordLatency(latencyMs) {
        lib.AdaptiveLimiterRecordLatency(this.id, latencyMs);
    }

    currentRate() {
        return lib.AdaptiveLimiterCurrentRate(this.id);
    }

    stop() {
        lib.AdaptiveLimiterStop(this.id);
    }
}

module.exports = { AdaptiveLimiter };
