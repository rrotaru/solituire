package theme_test

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"solituire/theme"
)

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
}

func TestRegistry_ReturnsFiveThemes(t *testing.T) {
	r := theme.NewRegistry()
	names := r.List()
	if len(names) != 5 {
		t.Fatalf("expected 5 themes, got %d: %v", len(names), names)
	}
}

func TestRegistry_GetByName(t *testing.T) {
	r := theme.NewRegistry()

	t.Run("existing theme", func(t *testing.T) {
		got := r.Get("Classic")
		if got.Name != "Classic" {
			t.Errorf("expected Classic, got %q", got.Name)
		}
	})

	t.Run("all registered names resolve", func(t *testing.T) {
		for _, name := range r.List() {
			got := r.Get(name)
			if got.Name != name {
				t.Errorf("Get(%q) returned theme with Name=%q", name, got.Name)
			}
		}
	})

	t.Run("lowercase names resolve (config contract)", func(t *testing.T) {
		cases := map[string]string{
			"classic":         "Classic",
			"dracula":         "Dracula",
			"solarized dark":  "Solarized Dark",
			"solarized light": "Solarized Light",
			"nord":            "Nord",
		}
		for input, want := range cases {
			got := r.Get(input)
			if got.Name != want {
				t.Errorf("Get(%q): want %q, got %q", input, want, got.Name)
			}
		}
	})

	t.Run("unknown name returns Classic", func(t *testing.T) {
		got := r.Get("DoesNotExist")
		if got.Name != "Classic" {
			t.Errorf("expected Classic fallback, got %q", got.Name)
		}
	})
}

func TestRegistry_NextCycles(t *testing.T) {
	r := theme.NewRegistry()
	names := r.List()

	t.Run("advances through each theme", func(t *testing.T) {
		for i, name := range names {
			want := names[(i+1)%len(names)]
			got := r.Next(name)
			if got.Name != want {
				t.Errorf("Next(%q): want %q, got %q", name, want, got.Name)
			}
		}
	})

	t.Run("last theme wraps to first", func(t *testing.T) {
		last := names[len(names)-1]
		got := r.Next(last)
		if got.Name != names[0] {
			t.Errorf("Next(%q) should wrap to %q, got %q", last, names[0], got.Name)
		}
	})

	t.Run("unknown name returns first theme", func(t *testing.T) {
		got := r.Next("DoesNotExist")
		if got.Name != names[0] {
			t.Errorf("Next(unknown) should return first theme %q, got %q", names[0], got.Name)
		}
	})

	t.Run("lowercase current advances correctly", func(t *testing.T) {
		got := r.Next("classic")
		if got.Name != "Dracula" {
			t.Errorf("Next(%q): want %q, got %q", "classic", "Dracula", got.Name)
		}
	})
}

func TestSolarizedLight_BlackSuitContrastsCardBackground(t *testing.T) {
	r := theme.NewRegistry()
	th := r.Get("Solarized Light")
	if string(th.BlackSuit) == string(th.CardBackground) {
		t.Errorf("SolarizedLight: BlackSuit (%s) must differ from CardBackground (%s) to ensure readability",
			th.BlackSuit, th.CardBackground)
	}
}

func TestAllThemesHaveNonZeroFields(t *testing.T) {
	r := theme.NewRegistry()
	for _, name := range r.List() {
		th := r.Get(name)
		t.Run(name, func(t *testing.T) {
			v := reflect.ValueOf(th)
			typ := v.Type()
			colorType := reflect.TypeOf(lipgloss.Color(""))
			for i := 0; i < v.NumField(); i++ {
				field := v.Field(i)
				fieldName := typ.Field(i).Name
				if fieldName == "Name" {
					continue
				}
				if field.Type() == colorType {
					if field.String() == "" {
						t.Errorf("theme %q: field %q is empty", name, fieldName)
					}
				}
			}
		})
	}
}
