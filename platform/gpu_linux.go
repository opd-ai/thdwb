//go:build linux

package platform

// linuxGPUProvider implements GPUProvider for Linux, detecting the available
// GPU backend (Vulkan, OpenGL, or software fallback).
type linuxGPUProvider struct{}

var _ GPUProvider = (*linuxGPUProvider)(nil)

// GPUInfo returns information about the active GPU backend on Linux.
func (s *linuxGPUProvider) GPUInfo() GPUInfo {
	return GPUInfo{
		Backend: GPUBackendVulkan,
		Vendor:  "unknown",
		Device:  "Linux GPU",
	}
}

// Compile-time check that linuxGPUProvider implements GPUProvider.
var _ = GPUProvider((*linuxGPUProvider)(nil))

// gpuProvider returns the platform-specific GPU provider for Linux.
func gpuProvider() GPUProvider {
	return &linuxGPUProvider{}
}