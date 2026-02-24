package format

import (
	"github.com/bornholm/guesstimate/internal/model"
	"gopkg.in/yaml.v3"
)

// YAMLFormatter formats estimations as YAML with calculated values
type YAMLFormatter struct {
	config *model.Config
}

// NewYAMLFormatter creates a new YAML formatter
func NewYAMLFormatter(config *model.Config) *YAMLFormatter {
	return &YAMLFormatter{config: config}
}

// Format formats an estimation as YAML
func (f *YAMLFormatter) Format(estimation *model.Estimation) (string, error) {
	return f.FormatWithOptions(estimation, FormatOptions{})
}

// FormatWithFactor formats an estimation as YAML with a time factor applied
func (f *YAMLFormatter) FormatWithFactor(estimation *model.Estimation, timeFactor float64) (string, error) {
	return f.FormatWithOptions(estimation, FormatOptions{TimeFactor: timeFactor})
}

// FormatWithOptions formats an estimation as YAML with the given options
func (f *YAMLFormatter) FormatWithOptions(estimation *model.Estimation, opts FormatOptions) (string, error) {
	// Use the same output structure as JSON formatter
	jsonFormatter := NewJSONFormatter(f.config)
	output := jsonFormatter.BuildOutput(estimation, opts)

	data, err := yaml.Marshal(output)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatSynthesis formats a synthesis of multiple estimations as YAML
func (f *YAMLFormatter) FormatSynthesis(input *SynthesisInput) (string, error) {
	// Use the same output structure as JSON formatter
	jsonFormatter := NewJSONFormatter(f.config)
	output := jsonFormatter.BuildSynthesisOutput(input)

	data, err := yaml.Marshal(output)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
