using System;
using System.Runtime.InteropServices;

namespace LaminarGate
{
    public class AdaptiveLimiter : IDisposable
    {
        private const string LibPath = "../ffi/ffi.so"; // Path to shared library

        [DllImport(LibPath, CallingConvention = CallingConvention.Cdecl)]
        private static extern long CreateAdaptiveLimiter(double targetLatencyMs, double maxRate, double initialRate, double kp, double kd, double intervalMs, double emaAlpha);

        [DllImport(LibPath, CallingConvention = CallingConvention.Cdecl)]
        [return: MarshalAs(UnmanagedType.I1)]
        private static extern bool AdaptiveLimiterAllow(long id);

        [DllImport(LibPath, CallingConvention = CallingConvention.Cdecl)]
        private static extern void AdaptiveLimiterRecordLatency(long id, double latencyMs);

        [DllImport(LibPath, CallingConvention = CallingConvention.Cdecl)]
        private static extern double AdaptiveLimiterCurrentRate(long id);

        [DllImport(LibPath, CallingConvention = CallingConvention.Cdecl)]
        private static extern void AdaptiveLimiterStop(long id);

        private readonly long _id;
        private bool _disposed = false;

        public AdaptiveLimiter(double targetLatencyMs = 10.0, double maxRate = 1000.0, double initialRate = 500.0, double kp = 0.5, double kd = 0.1, double intervalMs = 100.0, double emaAlpha = 0.2)
        {
            _id = CreateAdaptiveLimiter(targetLatencyMs, maxRate, initialRate, kp, kd, intervalMs, emaAlpha);
        }

        public bool Allow()
        {
            return AdaptiveLimiterAllow(_id);
        }

        public void RecordLatency(double latencyMs)
        {
            AdaptiveLimiterRecordLatency(_id, latencyMs);
        }

        public double CurrentRate()
        {
            return AdaptiveLimiterCurrentRate(_id);
        }

        public void Stop()
        {
            if (!_disposed)
            {
                AdaptiveLimiterStop(_id);
                _disposed = true;
            }
        }

        public void Dispose()
        {
            Stop();
            GC.SuppressFinalize(this);
        }

        ~AdaptiveLimiter()
        {
            Stop();
        }
    }
}
