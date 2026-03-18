package theme

// ThemeRegistry holds the set of available themes and allows lookup and cycling.
type ThemeRegistry struct {
	themes []Theme
}

// NewRegistry returns a ThemeRegistry pre-loaded with all built-in themes.
// Order: Classic, Dracula, Solarized Dark, Solarized Light, Nord.
func NewRegistry() *ThemeRegistry {
	return &ThemeRegistry{
		themes: []Theme{Classic, Dracula, SolarizedDark, SolarizedLight, Nord},
	}
}

// List returns the names of all registered themes in registration order.
func (r *ThemeRegistry) List() []string {
	names := make([]string, len(r.themes))
	for i, t := range r.themes {
		names[i] = t.Name
	}
	return names
}

// Get returns the theme with the given name.
// If no theme with that name is found, Classic is returned as the default.
func (r *ThemeRegistry) Get(name string) Theme {
	for _, t := range r.themes {
		if t.Name == name {
			return t
		}
	}
	return Classic
}

// Next returns the theme that follows the current theme in the cycle.
// If current is the last theme (or not found), it wraps to the first theme.
func (r *ThemeRegistry) Next(current string) Theme {
	for i, t := range r.themes {
		if t.Name == current {
			return r.themes[(i+1)%len(r.themes)]
		}
	}
	return r.themes[0]
}
