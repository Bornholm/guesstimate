package stats

import (
	"math"

	"github.com/bornholm/guesstimate/internal/model"
)

// EstimationResult contains the calculated estimation results
type EstimationResult struct {
	WeightedMean      float64 `json:"weightedMean"`
	StandardDeviation float64 `json:"standardDeviation"`
}

// ConfidenceLevel represents a confidence level with its multiplier
type ConfidenceLevel struct {
	Name       string
	Multiplier float64
}

// Standard confidence levels
var (
	Confidence68  = ConfidenceLevel{Name: "68%", Multiplier: 1}
	Confidence90  = ConfidenceLevel{Name: "90%", Multiplier: 1.645}
	Confidence997 = ConfidenceLevel{Name: "99.7%", Multiplier: 3}
)

// CalculateEstimation calculates the weighted mean and standard deviation for a task
func CalculateEstimation(task *model.Task) EstimationResult {
	return EstimationResult{
		WeightedMean:      task.WeightedMean(),
		StandardDeviation: task.StandardDeviation(),
	}
}

// CalculateProjectEstimation calculates the weighted mean and standard deviation for an entire project
func CalculateProjectEstimation(estimation *model.Estimation) EstimationResult {
	var totalMean float64
	var totalVariance float64

	for _, task := range estimation.Tasks {
		totalMean += task.WeightedMean()
		totalVariance += math.Pow(task.StandardDeviation(), 2)
	}

	return EstimationResult{
		WeightedMean:      totalMean,
		StandardDeviation: math.Sqrt(totalVariance),
	}
}

// CalculateCategoryEstimation calculates the weighted mean for a specific category
func CalculateCategoryEstimation(estimation *model.Estimation, categoryID string) EstimationResult {
	var totalMean float64
	var totalVariance float64

	for _, task := range estimation.Tasks {
		if task.Category == categoryID {
			totalMean += task.WeightedMean()
			totalVariance += math.Pow(task.StandardDeviation(), 2)
		}
	}

	return EstimationResult{
		WeightedMean:      totalMean,
		StandardDeviation: math.Sqrt(totalVariance),
	}
}

// CategoryDistribution represents the distribution of time across categories
type CategoryDistribution struct {
	CategoryID    string
	CategoryLabel string
	Time          float64
	Percentage    float64
}

// CalculateCategoryDistribution calculates the distribution of time across categories
func CalculateCategoryDistribution(estimation *model.Estimation, config *model.Config) []CategoryDistribution {
	projectEst := CalculateProjectEstimation(estimation)
	if projectEst.WeightedMean == 0 {
		return nil
	}

	distributions := make([]CategoryDistribution, 0)
	seenCategories := make(map[string]bool)

	// First, process configured categories
	for catID, cat := range config.TaskCategories {
		catEst := CalculateCategoryEstimation(estimation, catID)
		percentage := 0.0
		if projectEst.WeightedMean > 0 {
			percentage = (catEst.WeightedMean / projectEst.WeightedMean) * 100
		}

		distributions = append(distributions, CategoryDistribution{
			CategoryID:    catID,
			CategoryLabel: cat.Label,
			Time:          catEst.WeightedMean,
			Percentage:    percentage,
		})
		seenCategories[catID] = true
	}

	// Then, add any categories from tasks that are not in the config
	for _, task := range estimation.Tasks {
		if !seenCategories[task.Category] {
			catEst := CalculateCategoryEstimation(estimation, task.Category)
			percentage := 0.0
			if projectEst.WeightedMean > 0 {
				percentage = (catEst.WeightedMean / projectEst.WeightedMean) * 100
			}
			cat := config.GetTaskCategory(task.Category)
			distributions = append(distributions, CategoryDistribution{
				CategoryID:    task.Category,
				CategoryLabel: cat.Label,
				Time:          catEst.WeightedMean,
				Percentage:    percentage,
			})
			seenCategories[task.Category] = true
		}
	}

	return distributions
}

// CostEstimation represents cost estimation results
type CostEstimation struct {
	TotalTime float64
	TotalCost float64
	Details   map[string]CategoryCost
}

// CategoryCost represents cost details for a category
type CategoryCost struct {
	Time        float64
	Cost        float64
	CostPerUnit float64
}

// MinMaxCost represents minimum and maximum cost estimates
type MinMaxCost struct {
	Min CostEstimation
	Max CostEstimation
}

// CalculateMinMaxCosts calculates the min and max cost estimates for a given confidence level
func CalculateMinMaxCosts(estimation *model.Estimation, config *model.Config, confidence ConfidenceLevel) MinMaxCost {
	projectEst := CalculateProjectEstimation(estimation)
	distribution := CalculateCategoryDistribution(estimation, config)

	minCost := CostEstimation{
		Details: make(map[string]CategoryCost),
	}
	maxCost := CostEstimation{
		Details: make(map[string]CategoryCost),
	}

	// Calculate min estimate (E - SD * multiplier)
	minTime := math.Max(0, projectEst.WeightedMean-projectEst.StandardDeviation*confidence.Multiplier)
	// Calculate max estimate (E + SD * multiplier)
	maxTime := projectEst.WeightedMean + projectEst.StandardDeviation*confidence.Multiplier

	for _, dist := range distribution {
		cat := config.GetTaskCategory(dist.CategoryID)

		// Min time for this category
		minCatTime := (dist.Percentage / 100) * minTime
		minCatCost := minCatTime * cat.CostPerTimeUnit
		minCost.Details[dist.CategoryID] = CategoryCost{
			Time:        minCatTime,
			Cost:        minCatCost,
			CostPerUnit: cat.CostPerTimeUnit,
		}
		minCost.TotalTime += minCatTime
		minCost.TotalCost += minCatCost

		// Max time for this category
		maxCatTime := (dist.Percentage / 100) * maxTime
		maxCatCost := maxCatTime * cat.CostPerTimeUnit
		maxCost.Details[dist.CategoryID] = CategoryCost{
			Time:        maxCatTime,
			Cost:        maxCatCost,
			CostPerUnit: cat.CostPerTimeUnit,
		}
		maxCost.TotalTime += maxCatTime
		maxCost.TotalCost += maxCatCost
	}

	return MinMaxCost{
		Min: minCost,
		Max: maxCost,
	}
}

// FormatEstimation formats an estimation value with optional rounding
func FormatEstimation(value float64, roundUp bool) float64 {
	if roundUp {
		return math.Ceil(value)
	}
	return value
}
