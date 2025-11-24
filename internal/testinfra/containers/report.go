package containers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Report struct {
	jsonMode    bool
	startTime   time.Time
	phases      []PhaseResult
	currentStep string
	steps       []StepResult
}

type PhaseResult struct {
	Name      string        `json:"name"`
	StartTime time.Time     `json:"start_time"`
	Duration  time.Duration `json:"duration"`
	Steps     []StepResult  `json:"steps"`
}

type StepResult struct {
	Description string        `json:"description"`
	Success     bool          `json:"success"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
}

type FinalReport struct {
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	Phases    []PhaseResult `json:"phases"`
	Timestamp time.Time     `json:"timestamp"`
}

func NewReport(jsonMode bool) *Report {
	return &Report{
		jsonMode:  jsonMode,
		startTime: time.Now(),
		phases:    make([]PhaseResult, 0),
		steps:     make([]StepResult, 0),
	}
}

func (r *Report) Phase(name string) {
	if len(r.steps) > 0 {
		// Save current phase before starting new one
		if len(r.phases) > 0 {
			r.phases[len(r.phases)-1].Steps = r.steps
			r.phases[len(r.phases)-1].Duration = time.Since(r.phases[len(r.phases)-1].StartTime)
		}
		r.steps = make([]StepResult, 0)
	}

	r.phases = append(r.phases, PhaseResult{
		Name:      name,
		StartTime: time.Now(),
		Steps:     make([]StepResult, 0),
	})

	if !r.jsonMode {
		separator := strings.Repeat("─", 60)
		fmt.Printf("\n%s\n", separator)
		fmt.Printf("▶ %s\n", name)
		fmt.Printf("%s\n", separator)
	}
}

func (r *Report) Step(description string) {
	r.currentStep = description
	// Silent - only show results, not intermediate steps
}

func (r *Report) StepSuccess(description string) {
	r.steps = append(r.steps, StepResult{
		Description: description,
		Success:     true,
		Duration:    0,
	})
	if !r.jsonMode {
		fmt.Printf("  ✓ %s\n", description)
	}
}

func (r *Report) StepFail(description string, err error) {
	r.steps = append(r.steps, StepResult{
		Description: description,
		Success:     false,
		Error:       err.Error(),
	})
	if !r.jsonMode {
		fmt.Printf("  ❌ %s: %v\n", description, err)
	}
}

func (r *Report) Success(message string) {
	if !r.jsonMode {
		fmt.Printf("\n✅ %s\n", message)
	}
}

func (r *Report) Fail(format string, args ...interface{}) {
	if !r.jsonMode {
		fmt.Printf("\n❌ "+format+"\n", args...)
	}
}

func (r *Report) Info(format string, args ...interface{}) {
	if !r.jsonMode {
		fmt.Printf("    · "+format+"\n", args...)
	}
}

func (r *Report) Print() {
	// Save last phase
	if len(r.phases) > 0 && len(r.steps) > 0 {
		r.phases[len(r.phases)-1].Steps = r.steps
		r.phases[len(r.phases)-1].Duration = time.Since(r.phases[len(r.phases)-1].StartTime)
	}

	if r.jsonMode {
		finalReport := FinalReport{
			Success:   true,
			Duration:  time.Since(r.startTime),
			Phases:    r.phases,
			Timestamp: time.Now(),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(finalReport)
	} else {
		fmt.Printf("\n📊 Total Duration: %v\n", time.Since(r.startTime))
	}
}
