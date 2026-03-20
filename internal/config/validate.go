package config

import (
	"errors"
	"fmt"
	"strings"
)

// validLogLevels lists accepted log levels.
var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

// validRoutingStrategies lists accepted routing strategies.
var validRoutingStrategies = map[string]bool{
	"cost_optimized": true,
	"performance":    true,
}

// validOnExhaustPolicies lists accepted on_exhaust values.
var validOnExhaustPolicies = map[string]bool{
	"escalate":          true,
	"pause_requirement": true,
}

// validSessionPolicies lists accepted session recovery policies.
var validSessionPolicies = map[string]bool{
	"redispatch": true,
	"restart":    true,
	"ignore":     true,
}

// Validate checks that the config values are within acceptable ranges.
func (c Config) Validate() error {
	var errs []string

	if e := c.Budget.validate(); e != nil {
		errs = append(errs, e...)
	}
	if e := c.Sessions.validate(); e != nil {
		errs = append(errs, e...)
	}
	if e := c.Pipeline.validate(); e != nil {
		errs = append(errs, e...)
	}
	if e := c.Planning.validate(); e != nil {
		errs = append(errs, e...)
	}
	if e := c.Routing.validate(); e != nil {
		errs = append(errs, e...)
	}
	if e := c.Pricing.validate(); e != nil {
		errs = append(errs, e...)
	}
	if e := c.Workspace.validate(); e != nil {
		errs = append(errs, e...)
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (b BudgetConfig) validate() []string {
	var errs []string
	if b.MaxCostPerStoryUSD <= 0 {
		errs = append(errs, "budget.max_cost_per_story_usd must be > 0")
	}
	if b.MaxCostPerRequirementUSD <= 0 {
		errs = append(errs, "budget.max_cost_per_requirement_usd must be > 0")
	}
	if b.MaxCostPerDayUSD <= 0 {
		errs = append(errs, "budget.max_cost_per_day_usd must be > 0")
	}
	if b.WarningThresholdPct < 1 || b.WarningThresholdPct > 100 {
		errs = append(errs, "budget.warning_threshold_pct must be between 1 and 100")
	}
	return errs
}

func (s SessionsConfig) validate() []string {
	var errs []string
	if s.StaleThresholdS <= 0 {
		errs = append(errs, "sessions.stale_threshold_s must be > 0")
	}
	if !validSessionPolicies[s.OnDead] {
		errs = append(errs, fmt.Sprintf("sessions.on_dead must be one of: %s", joinKeys(validSessionPolicies)))
	}
	if !validSessionPolicies[s.OnStale] {
		errs = append(errs, fmt.Sprintf("sessions.on_stale must be one of: %s", joinKeys(validSessionPolicies)))
	}
	if s.MaxRecoveryAttempts < 0 {
		errs = append(errs, "sessions.max_recovery_attempts must be >= 0")
	}
	return errs
}

func (p PipelineConfig) validate() []string {
	var errs []string
	for name, stage := range p.Stages {
		if stage.MaxRetries < 0 {
			errs = append(errs, fmt.Sprintf("pipeline.stages.%s.max_retries must be >= 0", name))
		}
		if !validOnExhaustPolicies[stage.OnExhaust] {
			errs = append(errs, fmt.Sprintf("pipeline.stages.%s.on_exhaust must be one of: %s", name, joinKeys(validOnExhaustPolicies)))
		}
	}
	return errs
}

func (p PlanningConfig) validate() []string {
	var errs []string
	if p.MaxStoryComplexity <= 0 {
		errs = append(errs, "planning.max_story_complexity must be > 0")
	}
	if p.MaxStoriesPerRequirement <= 0 {
		errs = append(errs, "planning.max_stories_per_requirement must be > 0")
	}
	return errs
}

func (r RoutingConfig) validate() []string {
	var errs []string
	if !validRoutingStrategies[r.Strategy] {
		errs = append(errs, fmt.Sprintf("routing.strategy must be one of: %s", joinKeys(validRoutingStrategies)))
	}
	return errs
}

func (p PricingMap) validate() []string {
	var errs []string
	for model, pricing := range p {
		if pricing.InputPer1M < 0 {
			errs = append(errs, fmt.Sprintf("pricing.%s.input_per_1m must be >= 0", model))
		}
		if pricing.OutputPer1M < 0 {
			errs = append(errs, fmt.Sprintf("pricing.%s.output_per_1m must be >= 0", model))
		}
	}
	return errs
}

func (w WorkspaceConfig) validate() []string {
	var errs []string
	if !validLogLevels[w.LogLevel] {
		errs = append(errs, fmt.Sprintf("workspace.log_level must be one of: %s", joinKeys(validLogLevels)))
	}
	return errs
}

// joinKeys returns sorted keys from a map as a comma-separated string.
func joinKeys(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}
