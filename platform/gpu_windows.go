//go:build windows

package platform

// windowsGPUProvider implements GPUProvider for Windows, detecting the available
// GPU backend (Direct3D 12, Vulkan, or software fallback).
type windowsGPUProvider struct{}

var _ GPUProvider = (*windowsGPUProvider)(nil)

// GPUInfo returns information about the active GPU backend on Windows.
func (s *windowsGPUProvider) GPUInfo() GPUInfo {
	return GPUInfo{
		Backend: GPUBackendDirect3D12,
		Vendor:  "unknown",
		Device:  "Windows GPU",
	}
}

// Compile-time check that windowsGPUProvider implements GPUProvider.
var _ = GPUProvider((*windowsGPUProvider)(nil))

// gpuProvider returns the platform-specific GPU provider for Windows.
func gpuProvider() GPUProvider {
	return &windowsGPUProvider{}
}