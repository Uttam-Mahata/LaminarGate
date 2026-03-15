import ctypes
import os

# Load the shared library
lib_path = os.path.join(os.path.dirname(__file__), '..', 'ffi', 'ffi.so')
lib = ctypes.CDLL(lib_path)

# Define argument and return types
lib.CreateAdaptiveLimiter.argtypes = [ctypes.c_double, ctypes.c_double, ctypes.c_double, ctypes.c_double, ctypes.c_double, ctypes.c_double, ctypes.c_double]
lib.CreateAdaptiveLimiter.restype = ctypes.c_int64

lib.AdaptiveLimiterAllow.argtypes = [ctypes.c_int64]
lib.AdaptiveLimiterAllow.restype = ctypes.c_bool

lib.AdaptiveLimiterRecordLatency.argtypes = [ctypes.c_int64, ctypes.c_double]
lib.AdaptiveLimiterRecordLatency.restype = None

lib.AdaptiveLimiterCurrentRate.argtypes = [ctypes.c_int64]
lib.AdaptiveLimiterCurrentRate.restype = ctypes.c_double

lib.AdaptiveLimiterStop.argtypes = [ctypes.c_int64]
lib.AdaptiveLimiterStop.restype = None

class AdaptiveLimiter:
    def __init__(self, target_latency_ms=10.0, max_rate=1000.0, initial_rate=500.0, kp=0.5, kd=0.1, interval_ms=100.0, ema_alpha=0.2):
        self._id = lib.CreateAdaptiveLimiter(target_latency_ms, max_rate, initial_rate, kp, kd, interval_ms, ema_alpha)

    def allow(self):
        return lib.AdaptiveLimiterAllow(self._id)

    def record_latency(self, latency_ms):
        lib.AdaptiveLimiterRecordLatency(self._id, latency_ms)

    def current_rate(self):
        return lib.AdaptiveLimiterCurrentRate(self._id)

    def stop(self):
        lib.AdaptiveLimiterStop(self._id)

    def __del__(self):
        self.stop()
