package model

// DefaultAutoEstimationMultiplier is the default multiplier for auto-estimation (33%)
const DefaultAutoEstimationMultiplier = 0.33

// Config represents the application configuration stored in .guesstimate/config.yml
type Config struct {
	TaskCategories           map[string]TaskCategory `yaml:"taskCategories"`
	TimeUnit                 TimeUnit                `yaml:"timeUnit"`
	Currency                 string                  `yaml:"currency"`
	RoundUpEstimations       bool                    `yaml:"roundUpEstimations"`
	AutoEstimationMultiplier float64                 `yaml:"autoEstimationMultiplier,omitempty"`
}

// TaskCategory represents a category of tasks with associated cost
type TaskCategory struct {
	ID              string  `yaml:"-"`
	Label           string  `yaml:"label"`
	CostPerTimeUnit float64 `yaml:"costPerTimeUnit"`
}

// TimeUnit represents the time unit configuration
type TimeUnit struct {
	Label   string `yaml:"label"`
	Acronym string `yaml:"acronym"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		TaskCategories: map[string]TaskCategory{
			"development": {
				ID:              "development",
				Label:           "Development",
				CostPerTimeUnit: 500,
			},
			"project-management": {
				ID:              "project-management",
				Label:           "Project Management",
				CostPerTimeUnit: 500,
			},
			"testing": {
				ID:              "testing",
				Label:           "Testing",
				CostPerTimeUnit: 500,
			},
		},
		TimeUnit: TimeUnit{
			Label:   "man-day",
			Acronym: "md",
		},
		Currency:                 "€ H.T.",
		RoundUpEstimations:       true,
		AutoEstimationMultiplier: DefaultAutoEstimationMultiplier,
	}
}

// GetAutoEstimationMultiplier returns the configured multiplier or the default
func (c *Config) GetAutoEstimationMultiplier() float64 {
	if c.AutoEstimationMultiplier <= 0 {
		return DefaultAutoEstimationMultiplier
	}
	return c.AutoEstimationMultiplier
}

// GetTaskCategory returns a task category by ID, or a default one if not found
func (c *Config) GetTaskCategory(id string) TaskCategory {
	if cat, ok := c.TaskCategories[id]; ok {
		return cat
	}
	return TaskCategory{
		ID:              id,
		Label:           id,
		CostPerTimeUnit: 500,
	}
}

// GetFirstCategoryID returns the ID of the first task category
func (c *Config) GetFirstCategoryID() string {
	for id := range c.TaskCategories {
		return id
	}
	return ""
}
