package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tzone85/project-x/internal/git"
	"github.com/tzone85/project-x/internal/graph"
	"github.com/tzone85/project-x/internal/monitor"
	"github.com/tzone85/project-x/internal/pipeline"
	"github.com/tzone85/project-x/internal/planner"
	"github.com/tzone85/project-x/internal/runtime"
	"github.com/tzone85/project-x/internal/state"
)

func newResumeCmd() *cobra.Command {
	var godmode bool

	cmd := &cobra.Command{
		Use:   "resume <req-id>",
		Short: "Dispatch and monitor agents for a planned requirement",
		Long:  "Builds the dependency DAG, dispatches the next wave of agents, and monitors through the full pipeline (review, QA, rebase, merge).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResume(cmd.Context(), args[0], godmode)
		},
	}

	cmd.Flags().BoolVar(&godmode, "godmode", false, "enable autonomous operation (skip permission prompts)")
	return cmd
}

func runResume(ctx context.Context, reqID string, godmode bool) error {
	// 1. Load requirement and validate it exists
	req, err := app.projStore.GetRequirement(reqID)
	if err != nil {
		return fmt.Errorf("requirement %s not found: %w", reqID, err)
	}
	if req.Status == "archived" {
		return fmt.Errorf("requirement %s is archived", reqID)
	}

	fmt.Printf("Resuming requirement: %s (%s)\n", reqID, req.Title)

	// 2. Load stories and build DAG
	stories, err := app.projStore.ListStories(state.StoryFilter{ReqID: reqID})
	if err != nil {
		return fmt.Errorf("list stories: %w", err)
	}
	if len(stories) == 0 {
		return fmt.Errorf("no stories found for requirement %s — run 'px plan' first", reqID)
	}

	deps, err := app.projStore.ListStoryDeps(reqID)
	if err != nil {
		return fmt.Errorf("list story deps: %w", err)
	}

	dag := graph.NewDAG()
	storyMap := make(map[string]planner.PlannedStory, len(stories))
	completed := make(map[string]bool)

	for _, s := range stories {
		dag.AddNode(s.ID)
		storyMap[s.ID] = planner.PlannedStory{
			ID: s.ID, Title: s.Title, Description: s.Description,
			AcceptanceCriteria: s.AcceptanceCriteria, Complexity: s.Complexity,
			OwnedFiles: s.OwnedFiles, WaveHint: s.WaveHint,
		}
		if s.Status == "merged" || s.Status == "pr_submitted" {
			completed[s.ID] = true
		}
	}
	for _, d := range deps {
		dag.AddEdge(d.DependsOnID, d.StoryID)
	}

	if len(completed) == len(stories) {
		fmt.Println("All stories are already complete!")
		return nil
	}

	// 3. Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived shutdown signal, finishing in-flight work...")
		cancel()
	}()

	// 4. Determine repo directory
	repoDir := req.RepoPath
	if repoDir == "" {
		repoDir, _ = os.Getwd()
	}

	// 5. Set up runtime registry and router
	runner := git.ExecRunner{}
	reg := runtime.NewRegistry()
	reg.Register("claude-code", runtime.NewClaudeCodeRuntime(godmode))
	reg.Register("codex", runtime.NewCodexRuntime())
	reg.Register("gemini", runtime.NewGeminiRuntime())

	router := runtime.NewRouter(reg, app.config)

	// 6. Set up the pipeline stages
	llmClient := buildLLMClient()
	pipelineStages := []pipeline.Stage{
		pipeline.NewAutoCommitStage(runner),
		pipeline.NewDiffCheckStage(runner),
		pipeline.NewReviewStage(runner, llmClient),
		pipeline.NewQAStage(runner),
		pipeline.NewRebaseStage(runner, llmClient, 10),
		pipeline.NewMergeStage(runner, app.config.Merge.AutoMerge),
		pipeline.NewCleanupStage(runner),
	}
	pipelineRunner := pipeline.NewRunner(pipelineStages, app.config.Pipeline, app.eventStore)

	// 7. Set up dispatcher, executor, watchdog, poller
	dispatcher := monitor.NewDispatcher(app.config.Routing)
	executor := monitor.NewExecutor(runner, router, app.config, app.eventStore, app.projector)
	watchdog := monitor.NewWatchdog(monitor.WatchdogConfig{
		StuckThresholdS: app.config.Monitor.StuckThresholdS,
	}, app.eventStore)

	poller := monitor.NewPoller(
		monitor.PollerConfig{PollIntervalMs: app.config.Monitor.PollIntervalMs},
		runner, watchdog, pipelineRunner, app.eventStore, reg, app.projector,
	)

	// 8. Wave loop: dispatch → monitor → repeat until all done
	waveNumber := 0
	for {
		waveNumber++
		assignments, err := dispatcher.DispatchWave(dag, completed, reqID, storyMap, waveNumber)
		if err != nil {
			return fmt.Errorf("dispatch wave %d: %w", waveNumber, err)
		}
		if len(assignments) == 0 {
			// Check if all stories are done
			allDone := len(completed) == len(stories)
			if allDone {
				break
			}
			fmt.Println("No stories ready for dispatch (dependencies not met). Waiting...")
			break
		}

		fmt.Printf("\nWave %d: dispatching %d stories\n", waveNumber, len(assignments))
		for _, a := range assignments {
			fmt.Printf("  %s → %s (branch: %s)\n", a.StoryID, a.Role, a.Branch)
		}

		// Emit assignment events
		for _, a := range assignments {
			evt := state.NewEvent(state.EventStoryAssigned, a.AgentID, a.StoryID, map[string]any{
				"agent_id": a.AgentID,
				"wave":     waveNumber,
			})
			app.eventStore.Append(evt)
			app.projector.Send(evt)
		}

		// Spawn agents
		results := executor.SpawnAll(repoDir, assignments, storyMap)
		var activeAgents []monitor.ActiveAgent
		for _, r := range results {
			if r.Error != nil {
				fmt.Printf("  ERROR spawning %s: %v\n", r.Assignment.StoryID, r.Error)
				continue
			}
			activeAgents = append(activeAgents, monitor.ActiveAgent{
				Assignment:   r.Assignment,
				WorktreePath: r.WorktreePath,
				RuntimeName:  r.RuntimeName,
			})
		}

		if len(activeAgents) == 0 {
			fmt.Println("No agents spawned successfully")
			break
		}

		// Monitor until all agents complete
		if err := poller.Run(ctx, activeAgents, repoDir); err != nil {
			return fmt.Errorf("poller: %w", err)
		}

		// Update completed set
		refreshedStories, _ := app.projStore.ListStories(state.StoryFilter{ReqID: reqID})
		for _, s := range refreshedStories {
			if s.Status == "merged" || s.Status == "pr_submitted" {
				completed[s.ID] = true
			}
		}

		// Check for context cancellation
		if ctx.Err() != nil {
			fmt.Println("Shutdown complete. Resume later with: px resume", reqID)
			return nil
		}
	}

	// 9. Mark requirement complete if all stories done
	if len(completed) == len(stories) {
		compEvt := state.NewEvent(state.EventReqCompleted, "monitor", "", map[string]any{"id": reqID})
		app.eventStore.Append(compEvt)
		app.projector.Send(compEvt)
		fmt.Printf("\nAll %d stories complete! Requirement %s is done.\n", len(stories), reqID)
	}

	return nil
}
