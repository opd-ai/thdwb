//go:build darwin

package platform

// darwinGPUProvider implements GPUProvider for macOS, detecting the available
// GPU backend (Metal or software fallback).
type darwinGPUProvider struct{}

var _ GPUProvider = (*darwinGPUProvider)(nil)

// GPUInfo returns information about the active GPU backend on macOS.
func (s *darwinGPUProvider) GPUInfo() GPUInfo {
	return GPUInfo{
		Backend: GPUBackendMetal,
		Vendor:  "Apple",
		Device:  "macOS GPU",
	}
}

// Compile-time check that darwinGPUProvider implements GPUProvider.
var _ = GPUProvider((*darwinGPUProvider)(nil))

// gpuProvider returns the platform-specific GPU provider for macOS.
func gpuProvider() GPUProvider {
	return &darwinGPUProvider{}
}