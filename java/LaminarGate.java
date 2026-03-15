import com.sun.jna.Library;
import com.sun.jna.Native;
import com.sun.jna.Platform;

public class LaminarGate {

    public interface FFILibrary extends Library {
        // Assume ffi.so or ffi.dll is in the java library path
        FFILibrary INSTANCE = (FFILibrary) Native.load("ffi", FFILibrary.class);

        long CreateAdaptiveLimiter(double targetLatencyMs, double maxRate, double initialRate, double kp, double kd, double intervalMs, double emaAlpha);
        byte AdaptiveLimiterAllow(long id);
        void AdaptiveLimiterRecordLatency(long id, double latencyMs);
        double AdaptiveLimiterCurrentRate(long id);
        void AdaptiveLimiterStop(long id);
    }

    public static class AdaptiveLimiter implements AutoCloseable {
        private final long id;
        private boolean stopped = false;

        public AdaptiveLimiter(double targetLatencyMs, double maxRate, double initialRate, double kp, double kd, double intervalMs, double emaAlpha) {
            this.id = FFILibrary.INSTANCE.CreateAdaptiveLimiter(targetLatencyMs, maxRate, initialRate, kp, kd, intervalMs, emaAlpha);
        }

        public AdaptiveLimiter() {
            this(10.0, 1000.0, 500.0, 0.5, 0.1, 100.0, 0.2);
        }

        public boolean allow() {
            return FFILibrary.INSTANCE.AdaptiveLimiterAllow(this.id) != 0;
        }

        public void recordLatency(double latencyMs) {
            FFILibrary.INSTANCE.AdaptiveLimiterRecordLatency(this.id, latencyMs);
        }

        public double currentRate() {
            return FFILibrary.INSTANCE.AdaptiveLimiterCurrentRate(this.id);
        }

        public void stop() {
            if (!stopped) {
                FFILibrary.INSTANCE.AdaptiveLimiterStop(this.id);
                stopped = true;
            }
        }

        @Override
        public void close() {
            stop();
        }

        @Override
        protected void finalize() throws Throwable {
            try {
                stop();
            } finally {
                super.finalize();
            }
        }
    }
}
