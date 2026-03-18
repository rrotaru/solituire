package config

import "testing"

func TestDefaultConfig_Sane(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DrawCount != 1 {
		t.Errorf("DrawCount: want 1, got %d", cfg.DrawCount)
	}
	if cfg.ThemeName != "classic" {
		t.Errorf("ThemeName: want \"classic\", got %q", cfg.ThemeName)
	}
	if cfg.AutoMoveEnabled != false {
		t.Errorf("AutoMoveEnabled: want false, got true")
	}
	if cfg.Seed != 0 {
		t.Errorf("Seed: want 0, got %d", cfg.Seed)
	}
}

func TestDefaultConfig_ValidatesClean(t *testing.T) {
	if err := DefaultConfig().Validate(); err != nil {
		t.Errorf("DefaultConfig().Validate() returned unexpected error: %v", err)
	}
}

func TestValidate_DrawCount(t *testing.T) {
	tests := []struct {
		drawCount int
		wantErr   bool
	}{
		{1, false},
		{3, false},
		{0, true},
		{2, true},
		{4, true},
		{-1, true},
	}
	for _, tt := range tests {
		cfg := DefaultConfig()
		cfg.DrawCount = tt.drawCount
		err := cfg.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("DrawCount=%d: wantErr=%v, got err=%v", tt.drawCount, tt.wantErr, err)
		}
	}
}

func TestValidate_EmptyThemeName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ThemeName = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty ThemeName, got nil")
	}
}

func TestValidate_NonEmptyThemeName(t *testing.T) {
	for _, name := range []string{"classic", "dracula", "nord", "anything"} {
		cfg := DefaultConfig()
		cfg.ThemeName = name
		if err := cfg.Validate(); err != nil {
			t.Errorf("ThemeName=%q: unexpected error: %v", name, err)
		}
	}
}

func TestValidate_SeedZeroIsValid(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Seed = 0
	if err := cfg.Validate(); err != nil {
		t.Errorf("Seed=0 should be valid (means random), got error: %v", err)
	}
}
