package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sst/ion/pkg/project/path"
)

type Runtime interface {
	Match(runtime string) bool
	Build(ctx context.Context, input *BuildInput) (*BuildOutput, error)
	Run(ctx context.Context, input *RunInput) (Worker, error)
	ShouldRebuild(functionID string, path string) bool
}

type Worker interface {
	Stop()
	Logs() io.ReadCloser
}

type BuildInput struct {
	CfgPath    string
	Dev        bool                       `json:"dev"`
	FunctionID string                     `json:"functionID"`
	Handler    string                     `json:"handler"`
	Runtime    string                     `json:"runtime"`
	Properties json.RawMessage            `json:"properties"`
	Links      map[string]json.RawMessage `json:"links"`
	CopyFiles  []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"copyFiles"`
}

func (input *BuildInput) Out() string {
	return filepath.Join(path.ResolveWorkingDir(input.CfgPath), "artifacts", input.FunctionID)
}

type BuildOutput struct {
	Out     string   `json:"out"`
	Handler string   `json:"handler"`
	Errors  []string `json:"errors"`
}

type RunInput struct {
	CfgPath    string
	Runtime    string
	Server     string
	FunctionID string
	WorkerID   string
	Build      *BuildOutput
	Env        []string
}

type Collection struct {
	runtimes []Runtime
	cfgPath  string
	targets  map[string]*BuildInput
}

func NewCollection(platform string, runtimes ...Runtime) *Collection {
	return &Collection{
		runtimes: runtimes,
		cfgPath:  platform,
		targets:  map[string]*BuildInput{},
	}
}

func (c *Collection) Runtime(input string) (Runtime, bool) {
	for _, runtime := range c.runtimes {
		if runtime.Match(input) {
			return runtime, true
		}
	}
	return nil, false
}

func (c *Collection) Build(ctx context.Context, input *BuildInput) (*BuildOutput, error) {
	slog.Info("building function", "runtime", input.Runtime, "functionID", input.FunctionID)
	defer slog.Info("function built", "runtime", input.Runtime, "functionID", input.FunctionID)
	runtime, ok := c.Runtime(input.Runtime)
	if !ok {
		return nil, fmt.Errorf("Runtime not found: %v", input.Runtime)
	}
	out := input.Out()
	if err := os.RemoveAll(out); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return nil, err
	}
	result, err := runtime.Build(ctx, input)
	if err != nil {
		return nil, err
	}
	result.Out = out

	if len(input.CopyFiles) > 0 {
		for _, item := range input.CopyFiles {
			from, err := filepath.Abs(item.From)
			if err != nil {
				return nil, err
			}
			var dest string
			if item.To != "" {
				dest = filepath.Join(out, item.To)
			} else {
				dest = filepath.Join(out, item.From)
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return nil, err
			}
			if input.Dev {
				if err := os.Symlink(from, dest); err != nil {
					return nil, err
				}
			}
			if !input.Dev {
				sourceFile, err := os.Open(from)
				if err != nil {
					return nil, err
				}
				defer sourceFile.Close()
				destFile, err := os.Create(dest)
				if err != nil {
					return nil, err
				}
				defer destFile.Close()
				_, err = io.Copy(destFile, sourceFile)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return result, nil
}

func (c *Collection) Run(ctx context.Context, input *RunInput) (Worker, error) {
	slog.Info("running function", "runtime", input.Runtime, "functionID", input.FunctionID)
	runtime, ok := c.Runtime(input.Runtime)
	input.Env = append(input.Env, "SST_LIVE=true")
	input.Env = append(input.Env, "SST_DEV=true")
	if !ok {
		return nil, fmt.Errorf("runtime not found")
	}
	return runtime.Run(ctx, input)
}

func (c *Collection) ShouldRebuild(runtime string, functionID string, file string) bool {
	slog.Info("checking if function should be rebuilt", "runtime", runtime, "functionID", functionID, "file", file, "runtime", runtime)
	r, ok := c.Runtime(runtime)
	if !ok {
		return false
	}
	result := r.ShouldRebuild(functionID, file)
	slog.Info("should rebuild", "result", result, "functionID", functionID)
	return result
}

func (c *Collection) AddTarget(input *BuildInput) {
	input.CfgPath = c.cfgPath
	c.targets[input.FunctionID] = input
}
