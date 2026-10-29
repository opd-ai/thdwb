//go:build android || ios || js || wasip1

package platform

// stubGPUProvider implements GPUProvider for platforms without native GPU
// bindings. It reports the software backend as the active GPU.
type stubGPUProvider struct{}

var _ GPUProvider = (*stubGPUProvider)(nil)

// GPUInfo returns information about the active GPU backend.
func (s *stubGPUProvider) GPUInfo() GPUInfo {
	return GPUInfo{
		Backend: GPUBackendSoftware,
		Vendor:  "software",
		Device:  "CPU software renderer",
	}
}

// Compile-time check that stubGPUProvider implements GPUProvider.
var _ = GPUProvider((*stubGPUProvider)(nil))

// gpuProvider returns the platform-specific GPU provider for mobile platforms.
func gpuProvider() GPUProvider {
	return &stubGPUProvider{}
}