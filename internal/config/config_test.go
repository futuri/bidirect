package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.InitialSize != 400 {
		t.Errorf("InitialSize = %d, want 400", cfg.InitialSize)
	}
	if !cfg.KeepAspect {
		t.Error("KeepAspect should be true")
	}
	if cfg.MinSize != 150 {
		t.Errorf("MinSize = %d, want 150", cfg.MinSize)
	}
	if cfg.AlphaThreshold != 10 {
		t.Errorf("AlphaThreshold = %d, want 10", cfg.AlphaThreshold)
	}
	if cfg.BorderGrabSize != 8 {
		t.Errorf("BorderGrabSize = %d, want 8", cfg.BorderGrabSize)
	}
	if cfg.FPS != 60 {
		t.Errorf("FPS = %d, want 60", cfg.FPS)
	}
	if !cfg.AnimationEnabled {
		t.Error("AnimationEnabled should be true")
	}
	if cfg.WindowTitle != "Heart" {
		t.Errorf("WindowTitle = %q, want Heart", cfg.WindowTitle)
	}
}
