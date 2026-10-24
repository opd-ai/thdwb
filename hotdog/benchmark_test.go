package hotdog

import (
	"testing"
)

func BenchmarkNewWindowContext(b *testing.B) {
	for i := 0; i < b.N; i++ {
		wc := NewWindowContext(nil, nil, nil)
		if wc == nil {
			b.Fatal("NewWindowContext returned nil")
		}
	}
}

func BenchmarkSetOrigin(b *testing.B) {
	wc := NewWindowContext(nil, nil, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wc.SetOrigin("https://example.com")
	}
}

func BenchmarkGetOrigin(b *testing.B) {
	wc := NewWindowContext(nil, nil, nil)
	wc.SetOrigin("https://example.com:8080/path")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wc.GetOrigin()
	}
}

func BenchmarkRegisterEvent(b *testing.B) {
	wc := NewWindowContext(nil, nil, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wc.RegisterEvent("test", func(interface{}) {})
	}
}

func BenchmarkEmitEvent(b *testing.B) {
	wc := NewWindowContext(nil, nil, nil)
	wc.RegisterEvent("test", func(interface{}) {})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wc.EmitEvent("test", nil)
	}
}

func BenchmarkSetGetInputState(b *testing.B) {
	wc := NewWindowContext(nil, nil, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wc.SetInputState("key", "value")
		wc.GetInputState("key")
	}
}

func BenchmarkSetGetLayoutCache(b *testing.B) {
	wc := NewWindowContext(nil, nil, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wc.SetLayoutCache("key", "value")
		wc.GetLayoutCache("key")
	}
}

func BenchmarkIsActive(b *testing.B) {
	wc := NewWindowContext(nil, nil, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wc.IsActive()
	}
}

func BenchmarkSetActive(b *testing.B) {
	wc := NewWindowContext(nil, nil, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wc.SetActive(true)
		wc.SetActive(false)
	}
}

