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
	// Use the same output structure as JSON formatter
	jsonFormatter := NewJSONFormatter(f.config)
	output := jsonFormatter.BuildOutput(estimation)

	data, err := yaml.Marshal(output)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
