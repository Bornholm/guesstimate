# Guesstimate CLI

A command-line tool for creating and managing 3-point estimations of tasks. This is an improved CLI port of the [Guesstimate web application](https://forge.cadoles.com/wpetit/guesstimate).

## Features

- **Interactive UI**: Terminal-based interactive editor using `tview` with vim-style navigation
- **One-shot commands**: Script-friendly commands for automation
- **YAML storage**: Estimations stored in YAML format for easy Git sharing
- **Markdown export**: Generate markdown reports for documentation
- **Statistical calculations**: Automatic calculation of weighted mean, standard deviation, and confidence intervals
- **Category repartition**: Visual breakdown of time distribution across task categories

## Installation

```bash
go install github.com/bornholm/guesstimate/cmd/guesstimate@latest
```

Or build from source:

```bash
git clone https://github.com/bornholm/guesstimate.git
cd guesstimate
go build -o guesstimate ./cmd/guesstimate
```

## Quick Start

```bash
# Initialize configuration
guesstimate config init

# Create a new estimation
guesstimate new "My Project"

# Edit interactively
guesstimate edit my-project.estimation.yml

# View results
guesstimate view my-project.estimation.yml
```

## Interactive Editor

The interactive editor provides a vim-like experience:

| Key        | Action                  |
| ---------- | ----------------------- |
| `:w`       | Save estimation         |
| `:q`       | Quit (warns if unsaved) |
| `:q!`      | Force quit              |
| `:wq`      | Save and quit           |
| `a`        | Add new task            |
| `e` or `i` | Edit selected task      |
| `d`        | Delete selected task    |
| `J`        | Move task down          |
| `K`        | Move task up            |
| `j/k/h/l`  | Navigate (vim-style)    |
| `?`        | Show help               |

## One-Shot Commands

For scripting and automation:

```bash
# Add a task
guesstimate task add my-project.estimation.yml "Feature A" -c development -o 2 -l 4 -p 6

# List tasks
guesstimate task list my-project.estimation.yml

# Show summary with category repartition
guesstimate summary my-project.estimation.yml

# Export to markdown
guesstimate view my-project.estimation.yml -o report.md
```

## Configuration

Configuration is stored in `.guesstimate/config.yml`:

```yaml
taskCategories:
  development:
    label: "Development"
    costPerTimeUnit: 500
  testing:
    label: "Testing"
    costPerTimeUnit: 400

timeUnit:
  label: "man-day"
  acronym: "md"

currency: "€"
roundUpEstimations: true
```

## Statistical Calculations

- **Weighted Mean**: `E = (O + 4*L + P) / 6`
- **Standard Deviation**: `SD = (P - O) / 6`
- **Confidence Intervals**: 68% (1×SD), 90% (1.645×SD), 99.7% (3×SD)

## License

[GPL-3.0](https://www.gnu.org/licenses/gpl-3.0.txt)
