package format

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bornholm/guesstimate/internal/i18n"
	"github.com/bornholm/guesstimate/internal/model"
	"github.com/bornholm/guesstimate/internal/stats"
)

// MarkdownFormatter formats estimations as markdown
type MarkdownFormatter struct {
	config *model.Config
}

// NewMarkdownFormatter creates a new markdown formatter
func NewMarkdownFormatter(config *model.Config) *MarkdownFormatter {
	// Initialize i18n with config language
	i18n.InitLanguage(config.Language)
	return &MarkdownFormatter{config: config}
}

// Format formats an estimation as markdown
func (f *MarkdownFormatter) Format(estimation *model.Estimation) string {
	return f.FormatWithFactor(estimation, 1.0)
}

// FormatWithFactor formats an estimation as markdown with a time factor applied
func (f *MarkdownFormatter) FormatWithFactor(estimation *model.Estimation, timeFactor float64) string {
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("# %s\n\n", estimation.Label))

	// Description
	if estimation.Description != "" {
		sb.WriteString(fmt.Sprintf("> %s\n\n", estimation.Description))
	}

	// Categories
	sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeyCategories)))
	sb.WriteString(fmt.Sprintf("| %s | %s |\n", i18n.T(i18n.KeyCategory), i18n.T(i18n.KeyCostPerTimeUnit)))
	sb.WriteString("|----------|------------------|\n")
	for catID, cat := range f.config.TaskCategories {
		sb.WriteString(fmt.Sprintf("| %s | %s %s/%s |\n", cat.Label, formatFloat(cat.CostPerTimeUnit, false), f.config.Currency, f.config.TimeUnit.Acronym))
		_ = catID // avoid unused variable warning
	}
	sb.WriteString("\n")

	// Summary
	sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeySummary)))
	sb.WriteString(fmt.Sprintf("| %s | %s |\n", i18n.T(i18n.KeyConfidence), i18n.T(i18n.KeyEstimation)))
	sb.WriteString("|------------|------------|\n")

	projectEst := stats.CalculateProjectEstimation(estimation)
	roundUp := f.config.RoundUpEstimations

	// Apply time factor
	weightedMean := projectEst.WeightedMean * timeFactor
	standardDeviation := projectEst.StandardDeviation * timeFactor

	for _, cl := range []stats.ConfidenceLevel{stats.Confidence997, stats.Confidence90, stats.Confidence68} {
		e := weightedMean
		sd := standardDeviation * cl.Multiplier

		eStr := formatFloat(e, roundUp)
		sdStr := formatFloat(sd, roundUp)

		sb.WriteString(fmt.Sprintf("| >= %s | %s ± %s %s |\n", cl.Name, eStr, sdStr, f.config.TimeUnit.Acronym))
	}
	sb.WriteString("\n")

	// Financial Preview
	sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeyFinancialPreview)))
	costs := stats.CalculateMinMaxCosts(estimation, f.config, stats.Confidence997)

	sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", i18n.T(i18n.KeyType), i18n.T(i18n.KeyTime), i18n.T(i18n.KeyCost)))
	sb.WriteString("|------|------|------|\n")
	sb.WriteString(fmt.Sprintf("| %s | %s %s | %s %s |\n",
		i18n.T(i18n.KeyMaximum),
		formatFloat(costs.Max.TotalTime*timeFactor, roundUp), f.config.TimeUnit.Acronym,
		formatFloat(costs.Max.TotalCost*timeFactor, false), f.config.Currency))
	sb.WriteString(fmt.Sprintf("| %s | %s %s | %s %s |\n",
		i18n.T(i18n.KeyMinimum),
		formatFloat(costs.Min.TotalTime*timeFactor, roundUp), f.config.TimeUnit.Acronym,
		formatFloat(costs.Min.TotalCost*timeFactor, false), f.config.Currency))
	sb.WriteString("\n")

	// Cost by Category
	sb.WriteString(fmt.Sprintf("### %s\n\n", i18n.T(i18n.KeyCostByCategory)))
	sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", i18n.T(i18n.KeyCategory), i18n.T(i18n.KeyTime), i18n.T(i18n.KeyCost)))
	sb.WriteString("|----------|------|------|\n")

	for catID, catCost := range costs.Max.Details {
		cat := f.config.GetTaskCategory(catID)
		sb.WriteString(fmt.Sprintf("| %s | %s %s | %s %s |\n",
			cat.Label,
			formatFloat(catCost.Time*timeFactor, roundUp), f.config.TimeUnit.Acronym,
			formatFloat(catCost.Cost*timeFactor, false), f.config.Currency))
	}
	sb.WriteString("\n")

	// Tasks
	sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeyTasks)))
	sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n",
		i18n.T(i18n.KeyTask), i18n.T(i18n.KeyCategory), i18n.T(i18n.KeyOptimistic),
		i18n.T(i18n.KeyLikely), i18n.T(i18n.KeyPessimistic), i18n.T(i18n.KeyMean), i18n.T(i18n.KeySD)))
	sb.WriteString("|------|----------|------------|--------|-------------|------|----|\n")

	for _, task := range estimation.GetOrderedTasks() {
		cat := f.config.GetTaskCategory(task.Category)
		mean := task.WeightedMean() * timeFactor
		sd := task.StandardDeviation() * timeFactor

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n",
			task.Label,
			cat.Label,
			formatFloat(task.Estimations.Optimistic*timeFactor, false),
			formatFloat(task.Estimations.Likely*timeFactor, false),
			formatFloat(task.Estimations.Pessimistic*timeFactor, false),
			formatFloat(mean, roundUp),
			formatFloat(sd, roundUp),
		))
	}
	sb.WriteString("\n")

	// Category Distribution
	sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeyCategoryDistribution)))
	sb.WriteString(fmt.Sprintf("| %s | %s |\n", i18n.T(i18n.KeyCategory), i18n.T(i18n.KeyPercentage)))
	sb.WriteString("|----------|------------|\n")

	distribution := stats.CalculateCategoryDistribution(estimation, f.config)
	for _, dist := range distribution {
		sb.WriteString(fmt.Sprintf("| %s | %.0f%% |\n", dist.CategoryLabel, dist.Percentage))
	}
	sb.WriteString("\n")

	// Footer
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("*%s*\n", fmt.Sprintf(i18n.T(i18n.KeyGeneratedBy), time.Now().Format("2006-01-02 15:04:05"))))

	return sb.String()
}

func formatFloat(value float64, roundUp bool) string {
	if roundUp {
		return fmt.Sprintf("%.0f", math.Ceil(value))
	}
	return fmt.Sprintf("%.2f", value)
}

// FormatSynthesis formats a synthesis of multiple estimations as markdown
func (f *MarkdownFormatter) FormatSynthesis(input *SynthesisInput) string {
	var sb strings.Builder

	// Apply time factor (default to 1.0 if not set)
	timeFactor := input.TimeFactor
	if timeFactor == 0 {
		timeFactor = 1.0
	}

	// Title
	if input.Label != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", input.Label))
	} else {
		sb.WriteString(fmt.Sprintf("# %s\n\n", i18n.T(i18n.KeyEstimationSynthesis)))
	}

	// Categories
	sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeyCategories)))
	sb.WriteString(fmt.Sprintf("| %s | %s |\n", i18n.T(i18n.KeyCategory), i18n.T(i18n.KeyCostPerTimeUnit)))
	sb.WriteString("|----------|------------------|\n")
	for catID, cat := range f.config.TaskCategories {
		sb.WriteString(fmt.Sprintf("| %s | %s %s/%s |\n", cat.Label, formatFloat(cat.CostPerTimeUnit, false), f.config.Currency, f.config.TimeUnit.Acronym))
		_ = catID // avoid unused variable warning
	}
	sb.WriteString("\n")

	// Sources
	sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeySources)))
	sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", i18n.T(i18n.KeyFile), i18n.T(i18n.KeyLabel), i18n.T(i18n.KeyTasks)))
	sb.WriteString("|------|-------|-------|\n")
	for _, source := range input.Sources {
		sb.WriteString(fmt.Sprintf("| %s | %s | %d |\n", source.File, source.Label, source.Tasks))
	}
	sb.WriteString("\n")

	// Summary
	sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeySummary)))
	sb.WriteString(fmt.Sprintf("| %s | %s |\n", i18n.T(i18n.KeyConfidence), i18n.T(i18n.KeyEstimation)))
	sb.WriteString("|------------|------------|\n")

	projectEst := stats.CalculateSynthesisEstimation(input.Estimations)
	roundUp := f.config.RoundUpEstimations

	// Apply time factor
	weightedMean := projectEst.WeightedMean * timeFactor
	standardDeviation := projectEst.StandardDeviation * timeFactor

	for _, cl := range []stats.ConfidenceLevel{stats.Confidence997, stats.Confidence90, stats.Confidence68} {
		e := weightedMean
		sd := standardDeviation * cl.Multiplier

		eStr := formatFloat(e, roundUp)
		sdStr := formatFloat(sd, roundUp)

		sb.WriteString(fmt.Sprintf("| >= %s | %s ± %s %s |\n", cl.Name, eStr, sdStr, f.config.TimeUnit.Acronym))
	}
	sb.WriteString("\n")

	// Financial Preview
	sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeyFinancialPreview)))
	costs := stats.CalculateSynthesisMinMaxCosts(input.Estimations, f.config, stats.Confidence997)

	sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", i18n.T(i18n.KeyType), i18n.T(i18n.KeyTime), i18n.T(i18n.KeyCost)))
	sb.WriteString("|------|------|------|\n")
	sb.WriteString(fmt.Sprintf("| %s | %s %s | %s %s |\n",
		i18n.T(i18n.KeyMaximum),
		formatFloat(costs.Max.TotalTime*timeFactor, roundUp), f.config.TimeUnit.Acronym,
		formatFloat(costs.Max.TotalCost*timeFactor, false), f.config.Currency))
	sb.WriteString(fmt.Sprintf("| %s | %s %s | %s %s |\n",
		i18n.T(i18n.KeyMinimum),
		formatFloat(costs.Min.TotalTime*timeFactor, roundUp), f.config.TimeUnit.Acronym,
		formatFloat(costs.Min.TotalCost*timeFactor, false), f.config.Currency))
	sb.WriteString("\n")

	// Cost by Category
	sb.WriteString(fmt.Sprintf("### %s\n\n", i18n.T(i18n.KeyCostByCategory)))
	sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", i18n.T(i18n.KeyCategory), i18n.T(i18n.KeyTime), i18n.T(i18n.KeyCost)))
	sb.WriteString("|----------|------|------|\n")

	for catID, catCost := range costs.Max.Details {
		cat := f.config.GetTaskCategory(catID)
		sb.WriteString(fmt.Sprintf("| %s | %s %s | %s %s |\n",
			cat.Label,
			formatFloat(catCost.Time*timeFactor, roundUp), f.config.TimeUnit.Acronym,
			formatFloat(catCost.Cost*timeFactor, false), f.config.Currency))
	}
	sb.WriteString("\n")

	// Tasks (if requested)
	if input.IncludeTasks {
		sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeyTasks)))
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			i18n.T(i18n.KeySource), i18n.T(i18n.KeyTask), i18n.T(i18n.KeyCategory),
			i18n.T(i18n.KeyOptimistic), i18n.T(i18n.KeyLikely), i18n.T(i18n.KeyPessimistic),
			i18n.T(i18n.KeyMean), i18n.T(i18n.KeySD)))
		sb.WriteString("|--------|------|----------|------------|--------|-------------|------|----|\n")

		for i, est := range input.Estimations {
			sourceLabel := input.Sources[i].Label
			if sourceLabel == "" {
				sourceLabel = input.Sources[i].File
			}
			for _, task := range est.GetOrderedTasks() {
				cat := f.config.GetTaskCategory(task.Category)
				mean := task.WeightedMean() * timeFactor
				sd := task.StandardDeviation() * timeFactor

				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n",
					sourceLabel,
					task.Label,
					cat.Label,
					formatFloat(task.Estimations.Optimistic*timeFactor, false),
					formatFloat(task.Estimations.Likely*timeFactor, false),
					formatFloat(task.Estimations.Pessimistic*timeFactor, false),
					formatFloat(mean, roundUp),
					formatFloat(sd, roundUp),
				))
			}
		}
		sb.WriteString("\n")
	}

	// Category Distribution
	sb.WriteString(fmt.Sprintf("## %s\n\n", i18n.T(i18n.KeyCategoryDistribution)))
	sb.WriteString(fmt.Sprintf("| %s | %s |\n", i18n.T(i18n.KeyCategory), i18n.T(i18n.KeyPercentage)))
	sb.WriteString("|----------|------------|\n")

	distribution := stats.CalculateSynthesisCategoryDistribution(input.Estimations, f.config)
	for _, dist := range distribution {
		sb.WriteString(fmt.Sprintf("| %s | %.0f%% |\n", dist.CategoryLabel, dist.Percentage))
	}
	sb.WriteString("\n")

	// Footer
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("*%s*\n", fmt.Sprintf(i18n.T(i18n.KeyGeneratedBy), time.Now().Format("2006-01-02 15:04:05"))))

	return sb.String()
}
