package model

import (
	"math"

	"github.com/google/uuid"
)

// TaskID is a unique identifier for a task
type TaskID string

// Task represents a single task with 3-point estimation
type Task struct {
	ID          TaskID      `yaml:"id"`
	Label       string      `yaml:"label"`
	Description string      `yaml:"description,omitempty"`
	Category    string      `yaml:"category"`
	Estimations Estimations `yaml:"estimations"`
}

// Estimations contains the 3-point estimation values
type Estimations struct {
	Optimistic  float64 `yaml:"optimistic"`
	Likely      float64 `yaml:"likely"`
	Pessimistic float64 `yaml:"pessimistic"`
}

// NewTask creates a new task with the given label and category
func NewTask(label, category string) *Task {
	return &Task{
		ID:          TaskID(generateID()),
		Label:       label,
		Description: "",
		Category:    category,
		Estimations: Estimations{
			Optimistic:  0,
			Likely:      0,
			Pessimistic: 0,
		},
	}
}

// WeightedMean calculates the weighted mean (expected value) using the 3-point estimation formula
// E = (O + 4*L + P) / 6
func (t *Task) WeightedMean() float64 {
	return (t.Estimations.Optimistic + 4*t.Estimations.Likely + t.Estimations.Pessimistic) / 6
}

// StandardDeviation calculates the standard deviation using the 3-point estimation formula
// SD = (P - O) / 6
func (t *Task) StandardDeviation() float64 {
	return (t.Estimations.Pessimistic - t.Estimations.Optimistic) / 6
}

// Validate checks if the task estimations are valid (optimistic <= likely <= pessimistic)
func (t *Task) Validate() []string {
	var errors []string

	if t.Estimations.Optimistic < 0 {
		errors = append(errors, "optimistic estimate must be >= 0")
	}
	if t.Estimations.Likely < 0 {
		errors = append(errors, "likely estimate must be >= 0")
	}
	if t.Estimations.Pessimistic < 0 {
		errors = append(errors, "pessimistic estimate must be >= 0")
	}

	if t.Estimations.Likely < t.Estimations.Optimistic {
		errors = append(errors, "likely estimate should be >= optimistic estimate")
	}
	if t.Estimations.Pessimistic < t.Estimations.Likely {
		errors = append(errors, "pessimistic estimate should be >= likely estimate")
	}

	return errors
}

// SetEstimations sets all three estimates and ensures coherency using the given multiplier.
// The multiplier determines the percentage difference between adjacent estimates.
// Missing values (0) are auto-filled, and constraints are enforced by propagating forward:
// optimistic → likely → pessimistic. This ensures user input is always respected.
// Computed values are rounded up to the nearest integer.
func (t *Task) SetEstimations(optimistic, likely, pessimistic float64, multiplier float64) {
	o := optimistic
	l := likely
	p := pessimistic

	// Auto-fill missing values (0) based on what's provided
	if o > 0 && l == 0 && p == 0 {
		// Only optimistic is set
		l = math.Ceil(o * (1 + multiplier))
		p = math.Ceil(l * (1 + multiplier))
	} else if l > 0 && o == 0 && p == 0 {
		// Only likely is set
		o = math.Floor(l * (1 - multiplier))
		if o < 0 {
			o = 0
		}
		p = math.Ceil(l * (1 + multiplier))
	} else if p > 0 && o == 0 && l == 0 {
		// Only pessimistic is set
		l = math.Floor(p * (1 - multiplier))
		o = math.Floor(l * (1 - multiplier))
		if o < 0 {
			o = 0
		}
	} else if o > 0 && l > 0 && p == 0 {
		// Optimistic and likely set, pessimistic missing
		p = math.Ceil(l * (1 + multiplier))
	} else if o > 0 && p > 0 && l == 0 {
		// Optimistic and pessimistic set, likely missing
		l = math.Ceil((o + p) / 2)
		if l < o {
			l = o
		}
		if l > p {
			l = p
		}
	} else if l > 0 && p > 0 && o == 0 {
		// Likely and pessimistic set, optimistic missing
		o = math.Floor(l * (1 - multiplier))
		if o < 0 {
			o = 0
		}
	}

	// Enforce constraints by propagating forward (respect user input)
	// Only update values that violate the ordering constraint
	if l > 0 && l <= o {
		l = math.Ceil(o * (1 + multiplier))
	}
	if p > 0 && p <= l {
		p = math.Ceil(l * (1 + multiplier))
	}

	t.Estimations.Optimistic = o
	t.Estimations.Likely = l
	t.Estimations.Pessimistic = p
}

func generateID() string {
	return uuid.New().String()[:8]
}
