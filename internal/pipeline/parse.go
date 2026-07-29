package pipeline

import (
	"fmt"
	"log/slog"

	"github.com/vunas/blaster/internal/ai"
	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/endpoint"
	"github.com/vunas/blaster/internal/filesystem"
	"github.com/vunas/blaster/internal/openapi"
)

type StageError struct {
	Stage string
	Err   error
}

func (e *StageError) Error() string {
	return fmt.Sprintf("pipeline stage [%s] failed: %v", e.Stage, e.Err)
}

func (e *StageError) Unwrap() error {
	return e.Err
}

func newStageErr(stage string, err error) error {
	if err == nil {
		return nil
	}
	return &StageError{Stage: stage, Err: err}
}

// RunParse orchestrates the strict Parse pipeline.
func RunParse(cfg config.ParseConfig, fs filesystem.FS, log *slog.Logger) error {
	log.Info("Starting Blaster Parse Pipeline", "spec", cfg.SpecFile)

	log.Debug("Reading API specification file")
	rawBytes, err := fs.Read(cfg.SpecFile)
	if err != nil {
		return newStageErr("read_spec_file", err)
	}

	apiParser := openapi.NewParser()
	apiModel, err := apiParser.Parse(rawBytes)
	if err != nil {
		return newStageErr("openapi_parse", err)
	}

	log.Debug("Reading existing endpoints for state reconciliation")
	reader := endpoint.NewReader(cfg.EndpointDir, fs)
	oldEndpoints := reader.ReadAll()

	log.Info("Compiling endpoints (Deterministic phases)...")
	compiler := endpoint.NewCompiler(
		endpoint.NewSynchronizer(),
		endpoint.NewDiscoverer(cfg.EndpointConfig),
		endpoint.NewResolver(cfg.EndpointConfig),
		log,
	)

	compiledEndpoints := compiler.Compile(apiModel, oldEndpoints)

	pendingCount := countPending(compiledEndpoints)
	if pendingCount > 0 {
		log.Warn("Found pending candidates requiring resolution", "count", pendingCount)

		if cfg.AIConfig.IsActive() {
			log.Info("Invoking AI analysis for pending candidates...")
			aiResolver := ai.NewResolver(cfg.AIConfig, log)
			if err := aiResolver.Resolve(compiledEndpoints); err != nil {
				log.Warn("AI resolution encountered an issue, maintaining pending states", "error", err)
			}
		} else {
			log.Info("AI is inactive. Pending candidates will remain '?' in YAML.")
		}
	}

	log.Info("Writing compiled endpoints to disk...")
	writer := endpoint.NewWriter(cfg.EndpointDir, fs)
	if err := writer.WriteAll(compiledEndpoints); err != nil {
		return newStageErr("endpoint_write", err)
	}

	log.Info("Parse pipeline completed successfully!")
	return nil
}

func countPending(endpoints []*endpoint.Endpoint) int {
	count := 0
	for _, ep := range endpoints {
		for _, id := range ep.Identities {
			if id.Status == endpoint.StatusPending {
				count++
			}
		}
		for _, rel := range ep.Relatives {
			if rel.Status == endpoint.StatusPending {
				count++
			}
		}
	}
	return count
}
