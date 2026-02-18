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
	Currency   string                `json:"currency"`
	TimeUnit   string                `json:"timeUnit"`
	Max        CostDetail            `json:"max"`
	Min        CostDetail            `json:"min"`
	ByCategory map[string]CostDetail `json:"byCategory"`
}

// CostDetail represents detailed cost information
type CostDetail struct {
	Time float64 `json:"time"`
	Cost float64 `json:"cost"`
}

// Format formats an estimation as JSON
func (f *JSONFormatter) Format(estimation *model.Estimation) (string, error) {
	output := f.BuildOutput(estimation)
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

// BuildOutput builds the output structure
func (f *JSONFormatter) BuildOutput(estimation *model.Estimation) *Output {
	projectEst := stats.CalculateProjectEstimation(estimation)
	distribution := stats.CalculateCategoryDistribution(estimation, f.config)
	costs := stats.CalculateMinMaxCosts(estimation, f.config, stats.Confidence997)
	roundUp := f.config.RoundUpEstimations

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
				Optimistic:  task.Estimations.Optimistic,
				Likely:      task.Estimations.Likely,
				Pessimistic: task.Estimations.Pessimistic,
			},
			Calculated: TaskCalculatedOutput{
				WeightedMean:      roundFloat(task.WeightedMean(), roundUp),
				StandardDeviation: roundFloat(task.StandardDeviation(), roundUp),
			},
		})
	}

	// Build category distribution
	catDist := make([]CategoryDistributionOutput, 0, len(distribution))
	for _, dist := range distribution {
		catDist = append(catDist, CategoryDistributionOutput{
			CategoryID:    dist.CategoryID,
			CategoryLabel: dist.CategoryLabel,
			Time:          roundFloat(dist.Time, roundUp),
			Percentage:    dist.Percentage,
		})
	}

	// Build costs by category
	costsByCategory := make(map[string]CostDetail)
	for catID, catCost := range costs.Max.Details {
		costsByCategory[catID] = CostDetail{
			Time: roundFloat(catCost.Time, roundUp),
			Cost: roundFloat(catCost.Cost, false),
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
			WeightedMean:      roundFloat(projectEst.WeightedMean, roundUp),
			StandardDeviation: roundFloat(projectEst.StandardDeviation, roundUp),
			Confidence68: ConfidenceOutput{
				Level:     "68%",
				Mean:      roundFloat(projectEst.WeightedMean, roundUp),
				Deviation: roundFloat(projectEst.StandardDeviation, roundUp),
				Min:       roundFloat(projectEst.WeightedMean-projectEst.StandardDeviation, roundUp),
				Max:       roundFloat(projectEst.WeightedMean+projectEst.StandardDeviation, roundUp),
			},
			Confidence90: ConfidenceOutput{
				Level:     "90%",
				Mean:      roundFloat(projectEst.WeightedMean, roundUp),
				Deviation: roundFloat(projectEst.StandardDeviation*1.645, roundUp),
				Min:       roundFloat(projectEst.WeightedMean-projectEst.StandardDeviation*1.645, roundUp),
				Max:       roundFloat(projectEst.WeightedMean+projectEst.StandardDeviation*1.645, roundUp),
			},
			Confidence997: ConfidenceOutput{
				Level:     "99.7%",
				Mean:      roundFloat(projectEst.WeightedMean, roundUp),
				Deviation: roundFloat(projectEst.StandardDeviation*3, roundUp),
				Min:       roundFloat(projectEst.WeightedMean-projectEst.StandardDeviation*3, roundUp),
				Max:       roundFloat(projectEst.WeightedMean+projectEst.StandardDeviation*3, roundUp),
			},
		},
		CategoryDistribution: catDist,
		Costs: CostOutput{
			Currency:   f.config.Currency,
			TimeUnit:   f.config.TimeUnit.Acronym,
			Max:        CostDetail{Time: roundFloat(costs.Max.TotalTime, roundUp), Cost: roundFloat(costs.Max.TotalCost, false)},
			Min:        CostDetail{Time: roundFloat(costs.Min.TotalTime, roundUp), Cost: roundFloat(costs.Min.TotalCost, false)},
			ByCategory: costsByCategory,
		},
	}
}

// roundFloat rounds the value if roundUp is true, otherwise returns the value
func roundFloat(value float64, roundUp bool) float64 {
	if roundUp {
		return math.Ceil(value)
	}
	return value
}
