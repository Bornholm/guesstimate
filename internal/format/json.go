package format

import (
	"encoding/json"
	"math"

	"github.com/bornholm/guesstimate/internal/model"
	"github.com/bornholm/guesstimate/internal/stats"
)

// JSONFormatter formats estimations as JSON with calculated values
type JSONFormatter struct {
	config *model.Config
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(config *model.Config) *JSONFormatter {
	return &JSONFormatter{config: config}
}

// Output represents the complete estimation output with calculated values
type Output struct {
	// Project information
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`

	// Tasks
	Tasks []TaskOutput `json:"tasks"`

	// Calculated statistics
	Statistics StatisticsOutput `json:"statistics"`

	// Category distribution
	CategoryDistribution []CategoryDistributionOutput `json:"categoryDistribution"`

	// Cost estimation
	Costs CostOutput `json:"costs"`
}

// TaskOutput represents a task with calculated values
type TaskOutput struct {
	ID            string               `json:"id"`
	Label         string               `json:"label"`
	Description   string               `json:"description,omitempty"`
	Category      string               `json:"category"`
	CategoryLabel string               `json:"categoryLabel"`
	Estimations   EstimationOutput     `json:"estimations"`
	Calculated    TaskCalculatedOutput `json:"calculated"`
}

// EstimationOutput represents the three-point estimates
type EstimationOutput struct {
	Optimistic  float64 `json:"optimistic"`
	Likely      float64 `json:"likely"`
	Pessimistic float64 `json:"pessimistic"`
}

// TaskCalculatedOutput represents calculated values for a task
type TaskCalculatedOutput struct {
	WeightedMean      float64 `json:"weightedMean"`
	StandardDeviation float64 `json:"standardDeviation"`
}

// StatisticsOutput represents project-level statistics
type StatisticsOutput struct {
	TaskCount         int              `json:"taskCount"`
	WeightedMean      float64          `json:"weightedMean"`
	StandardDeviation float64          `json:"standardDeviation"`
	Confidence68      ConfidenceOutput `json:"confidence68"`
	Confidence90      ConfidenceOutput `json:"confidence90"`
	Confidence997     ConfidenceOutput `json:"confidence997"`
}

// ConfidenceOutput represents a confidence interval
type ConfidenceOutput struct {
	Level     string  `json:"level"`
	Mean      float64 `json:"mean"`
	Deviation float64 `json:"deviation"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
}

// CategoryDistributionOutput represents category distribution
type CategoryDistributionOutput struct {
	CategoryID    string  `json:"categoryId"`
	CategoryLabel string  `json:"categoryLabel"`
	Time          float64 `json:"time"`
	Percentage    float64 `json:"percentage"`
}

// CostOutput represents cost estimation
type CostOutput struct {
	Currency   string                `json:"currency,omitempty"`
	TimeUnit   string                `json:"timeUnit,omitempty"`
	Max        CostDetail            `json:"max,omitempty"`
	Min        CostDetail            `json:"min,omitempty"`
	ByCategory map[string]CostDetail `json:"byCategory,omitempty"`
}

// CostDetail represents detailed cost information
type CostDetail struct {
	Time float64 `json:"time"`
	Cost float64 `json:"cost"`
}

// Format formats an estimation as JSON
func (f *JSONFormatter) Format(estimation *model.Estimation) (string, error) {
	return f.FormatWithOptions(estimation, FormatOptions{})
}

// FormatWithFactor formats an estimation as JSON with a time factor applied
func (f *JSONFormatter) FormatWithFactor(estimation *model.Estimation, timeFactor float64) (string, error) {
	return f.FormatWithOptions(estimation, FormatOptions{TimeFactor: timeFactor})
}

// FormatWithOptions formats an estimation as JSON with the given options
func (f *JSONFormatter) FormatWithOptions(estimation *model.Estimation, opts FormatOptions) (string, error) {
	output := f.BuildOutput(estimation, opts)
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

// FormatOptions represents options for formatting
type FormatOptions struct {
	TimeFactor float64
	NoCosts    bool
}

// BuildOutput builds the output structure
func (f *JSONFormatter) BuildOutput(estimation *model.Estimation, opts FormatOptions) *Output {
	timeFactor := opts.TimeFactor
	if timeFactor == 0 {
		timeFactor = 1.0
	}
	projectEst := stats.CalculateProjectEstimation(estimation)
	distribution := stats.CalculateCategoryDistribution(estimation, f.config)
	costs := stats.CalculateMinMaxCosts(estimation, f.config, stats.Confidence997)
	roundUp := f.config.RoundUpEstimations

	// Apply time factor to weighted mean and standard deviation
	weightedMean := projectEst.WeightedMean * timeFactor
	standardDeviation := projectEst.StandardDeviation * timeFactor

	// Build tasks output
	tasks := make([]TaskOutput, 0, len(estimation.Tasks))
	for _, task := range estimation.GetOrderedTasks() {
		cat := f.config.GetTaskCategory(task.Category)
		tasks = append(tasks, TaskOutput{
			ID:            string(task.ID),
			Label:         task.Label,
			Description:   task.Description,
			Category:      task.Category,
			CategoryLabel: cat.Label,
			Estimations: EstimationOutput{
				Optimistic:  task.Estimations.Optimistic * timeFactor,
				Likely:      task.Estimations.Likely * timeFactor,
				Pessimistic: task.Estimations.Pessimistic * timeFactor,
			},
			Calculated: TaskCalculatedOutput{
				WeightedMean:      roundFloat(task.WeightedMean()*timeFactor, roundUp),
				StandardDeviation: roundFloat(task.StandardDeviation()*timeFactor, roundUp),
			},
		})
	}

	// Build category distribution
	catDist := make([]CategoryDistributionOutput, 0, len(distribution))
	for _, dist := range distribution {
		catDist = append(catDist, CategoryDistributionOutput{
			CategoryID:    dist.CategoryID,
			CategoryLabel: dist.CategoryLabel,
			Time:          roundFloat(dist.Time*timeFactor, roundUp),
			Percentage:    dist.Percentage,
		})
	}

	var costOutput CostOutput
	if !opts.NoCosts {
		// Build costs by category
		costsByCategory := make(map[string]CostDetail)
		for catID, catCost := range costs.Max.Details {
			costsByCategory[catID] = CostDetail{
				Time: roundFloat(catCost.Time*timeFactor, roundUp),
				Cost: roundFloat(catCost.Cost*timeFactor, false),
			}
		}
		costOutput = CostOutput{
			Currency:   f.config.Currency,
			TimeUnit:   f.config.TimeUnit.Acronym,
			Max:        CostDetail{Time: roundFloat(costs.Max.TotalTime*timeFactor, roundUp), Cost: roundFloat(costs.Max.TotalCost*timeFactor, false)},
			Min:        CostDetail{Time: roundFloat(costs.Min.TotalTime*timeFactor, roundUp), Cost: roundFloat(costs.Min.TotalCost*timeFactor, false)},
			ByCategory: costsByCategory,
		}
	}

	return &Output{
		ID:          string(estimation.ID),
		Label:       estimation.Label,
		Description: estimation.Description,
		CreatedAt:   estimation.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   estimation.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		Tasks:       tasks,
		Statistics: StatisticsOutput{
			TaskCount:         len(estimation.Tasks),
			WeightedMean:      roundFloat(weightedMean, roundUp),
			StandardDeviation: roundFloat(standardDeviation, roundUp),
			Confidence68: ConfidenceOutput{
				Level:     "68%",
				Mean:      roundFloat(weightedMean, roundUp),
				Deviation: roundFloat(standardDeviation, roundUp),
				Min:       roundFloat(weightedMean-standardDeviation, roundUp),
				Max:       roundFloat(weightedMean+standardDeviation, roundUp),
			},
			Confidence90: ConfidenceOutput{
				Level:     "90%",
				Mean:      roundFloat(weightedMean, roundUp),
				Deviation: roundFloat(standardDeviation*1.645, roundUp),
				Min:       roundFloat(weightedMean-standardDeviation*1.645, roundUp),
				Max:       roundFloat(weightedMean+standardDeviation*1.645, roundUp),
			},
			Confidence997: ConfidenceOutput{
				Level:     "99.7%",
				Mean:      roundFloat(weightedMean, roundUp),
				Deviation: roundFloat(standardDeviation*3, roundUp),
				Min:       roundFloat(weightedMean-standardDeviation*3, roundUp),
				Max:       roundFloat(weightedMean+standardDeviation*3, roundUp),
			},
		},
		CategoryDistribution: catDist,
		Costs:                costOutput,
	}
}

// roundFloat rounds the value if roundUp is true, otherwise returns the value
func roundFloat(value float64, roundUp bool) float64 {
	if roundUp {
		return math.Ceil(value)
	}
	return value
}

// SynthesisOutput represents the combined output of multiple estimations
type SynthesisOutput struct {
	// Optional label for the synthesis
	Label string `json:"label,omitempty"`

	// Source estimations
	Sources []SynthesisSource `json:"sources"`

	// Total number of tasks across all estimations
	TotalTasks int `json:"totalTasks"`

	// Calculated statistics
	Statistics StatisticsOutput `json:"statistics"`

	// Category distribution
	CategoryDistribution []CategoryDistributionOutput `json:"categoryDistribution"`

	// Cost estimation
	Costs CostOutput `json:"costs"`

	// Optional: all tasks from all estimations
	Tasks []TaskOutput `json:"tasks,omitempty"`
}

// SynthesisSource represents a source estimation in the synthesis
type SynthesisSource struct {
	File  string `json:"file"`
	Label string `json:"label"`
	Tasks int    `json:"tasks"`
}

// SynthesisInput represents input for building synthesis output
type SynthesisInput struct {
	Label        string
	Sources      []SynthesisSource
	Estimations  []*model.Estimation
	IncludeTasks bool
	TimeFactor   float64
	NoCosts      bool
}

// BuildSynthesisOutput builds the synthesis output structure
func (f *JSONFormatter) BuildSynthesisOutput(input *SynthesisInput) *SynthesisOutput {
	projectEst := stats.CalculateSynthesisEstimation(input.Estimations)
	distribution := stats.CalculateSynthesisCategoryDistribution(input.Estimations, f.config)
	costs := stats.CalculateSynthesisMinMaxCosts(input.Estimations, f.config, stats.Confidence997)
	roundUp := f.config.RoundUpEstimations

	// Apply time factor (default to 1.0 if not set)
	timeFactor := input.TimeFactor
	if timeFactor == 0 {
		timeFactor = 1.0
	}

	// Apply time factor to weighted mean and standard deviation
	weightedMean := projectEst.WeightedMean * timeFactor
	standardDeviation := projectEst.StandardDeviation * timeFactor

	// Count total tasks
	totalTasks := 0
	for _, est := range input.Estimations {
		totalTasks += len(est.Tasks)
	}

	// Build tasks output if requested
	var tasks []TaskOutput
	if input.IncludeTasks {
		tasks = make([]TaskOutput, 0, totalTasks)
		for _, est := range input.Estimations {
			for _, task := range est.GetOrderedTasks() {
				cat := f.config.GetTaskCategory(task.Category)
				tasks = append(tasks, TaskOutput{
					ID:            string(task.ID),
					Label:         task.Label,
					Description:   task.Description,
					Category:      task.Category,
					CategoryLabel: cat.Label,
					Estimations: EstimationOutput{
						Optimistic:  task.Estimations.Optimistic * timeFactor,
						Likely:      task.Estimations.Likely * timeFactor,
						Pessimistic: task.Estimations.Pessimistic * timeFactor,
					},
					Calculated: TaskCalculatedOutput{
						WeightedMean:      roundFloat(task.WeightedMean()*timeFactor, roundUp),
						StandardDeviation: roundFloat(task.StandardDeviation()*timeFactor, roundUp),
					},
				})
			}
		}
	}

	// Build category distribution
	catDist := make([]CategoryDistributionOutput, 0, len(distribution))
	for _, dist := range distribution {
		catDist = append(catDist, CategoryDistributionOutput{
			CategoryID:    dist.CategoryID,
			CategoryLabel: dist.CategoryLabel,
			Time:          roundFloat(dist.Time*timeFactor, roundUp),
			Percentage:    dist.Percentage,
		})
	}

	var costOutput CostOutput
	if !input.NoCosts {
		// Build costs by category
		costsByCategory := make(map[string]CostDetail)
		for catID, catCost := range costs.Max.Details {
			costsByCategory[catID] = CostDetail{
				Time: roundFloat(catCost.Time*timeFactor, roundUp),
				Cost: roundFloat(catCost.Cost*timeFactor, false),
			}
		}
		costOutput = CostOutput{
			Currency:   f.config.Currency,
			TimeUnit:   f.config.TimeUnit.Acronym,
			Max:        CostDetail{Time: roundFloat(costs.Max.TotalTime*timeFactor, roundUp), Cost: roundFloat(costs.Max.TotalCost*timeFactor, false)},
			Min:        CostDetail{Time: roundFloat(costs.Min.TotalTime*timeFactor, roundUp), Cost: roundFloat(costs.Min.TotalCost*timeFactor, false)},
			ByCategory: costsByCategory,
		}
	}

	return &SynthesisOutput{
		Label:      input.Label,
		Sources:    input.Sources,
		TotalTasks: totalTasks,
		Tasks:      tasks,
		Statistics: StatisticsOutput{
			TaskCount:         totalTasks,
			WeightedMean:      roundFloat(weightedMean, roundUp),
			StandardDeviation: roundFloat(standardDeviation, roundUp),
			Confidence68: ConfidenceOutput{
				Level:     "68%",
				Mean:      roundFloat(weightedMean, roundUp),
				Deviation: roundFloat(standardDeviation, roundUp),
				Min:       roundFloat(weightedMean-standardDeviation, roundUp),
				Max:       roundFloat(weightedMean+standardDeviation, roundUp),
			},
			Confidence90: ConfidenceOutput{
				Level:     "90%",
				Mean:      roundFloat(weightedMean, roundUp),
				Deviation: roundFloat(standardDeviation*1.645, roundUp),
				Min:       roundFloat(weightedMean-standardDeviation*1.645, roundUp),
				Max:       roundFloat(weightedMean+standardDeviation*1.645, roundUp),
			},
			Confidence997: ConfidenceOutput{
				Level:     "99.7%",
				Mean:      roundFloat(weightedMean, roundUp),
				Deviation: roundFloat(standardDeviation*3, roundUp),
				Min:       roundFloat(weightedMean-standardDeviation*3, roundUp),
				Max:       roundFloat(weightedMean+standardDeviation*3, roundUp),
			},
		},
		CategoryDistribution: catDist,
		Costs:                costOutput,
	}
}

// FormatSynthesis formats a synthesis of multiple estimations as JSON
func (f *JSONFormatter) FormatSynthesis(input *SynthesisInput) (string, error) {
	output := f.BuildSynthesisOutput(input)
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}
