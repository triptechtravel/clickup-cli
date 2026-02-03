package iostreams

import "fmt"

// ANSI color codes
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	gray   = "\033[90m"
	white  = "\033[97m"
)

// ColorScheme provides color formatting when color output is enabled.
type ColorScheme struct {
	enabled bool
}

func NewColorScheme(enabled bool) *ColorScheme {
	return &ColorScheme{enabled: enabled}
}

func (c *ColorScheme) Bold(t string) string {
	if !c.enabled {
		return t
	}
	return fmt.Sprintf("%s%s%s", bold, t, reset)
}

func (c *ColorScheme) Red(t string) string {
	if !c.enabled {
		return t
	}
	return fmt.Sprintf("%s%s%s", red, t, reset)
}

func (c *ColorScheme) Green(t string) string {
	if !c.enabled {
		return t
	}
	return fmt.Sprintf("%s%s%s", green, t, reset)
}

func (c *ColorScheme) Yellow(t string) string {
	if !c.enabled {
		return t
	}
	return fmt.Sprintf("%s%s%s", yellow, t, reset)
}

func (c *ColorScheme) Blue(t string) string {
	if !c.enabled {
		return t
	}
	return fmt.Sprintf("%s%s%s", blue, t, reset)
}

func (c *ColorScheme) Cyan(t string) string {
	if !c.enabled {
		return t
	}
	return fmt.Sprintf("%s%s%s", cyan, t, reset)
}

func (c *ColorScheme) Gray(t string) string {
	if !c.enabled {
		return t
	}
	return fmt.Sprintf("%s%s%s", gray, t, reset)
}

func (c *ColorScheme) White(t string) string {
	if !c.enabled {
		return t
	}
	return fmt.Sprintf("%s%s%s", white, t, reset)
}

// StatusColor returns a color function appropriate for a ClickUp status.
func (c *ColorScheme) StatusColor(status string) func(string) string {
	switch status {
	case "to do", "open":
		return c.Gray
	case "in progress", "in review":
		return c.Blue
	case "done", "complete", "closed":
		return c.Green
	default:
		return c.Yellow
	}
}

// PriorityColor returns a color function appropriate for a ClickUp priority.
func (c *ColorScheme) PriorityColor(priority int) func(string) string {
	switch priority {
	case 1: // Urgent
		return c.Red
	case 2: // High
		return c.Yellow
	case 3: // Normal
		return c.Blue
	case 4: // Low
		return c.Gray
	default:
		return c.White
	}
}

func (c *ColorScheme) Enabled() bool {
	return c.enabled
}
