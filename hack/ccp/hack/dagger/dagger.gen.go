// Code generated by dagger. DO NOT EDIT.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"dagger/ccp/internal/dagger"
	"dagger/ccp/internal/telemetry"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var dag = dagger.Connect()

func Tracer() trace.Tracer {
	return otel.Tracer("dagger.io/sdk.go")
}

// used for local MarshalJSON implementations
var marshalCtx = context.Background()

// called by main()
func setMarshalContext(ctx context.Context) {
	marshalCtx = ctx
	dagger.SetMarshalContext(ctx)
}

type DaggerObject = dagger.DaggerObject

type ExecError = dagger.ExecError

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// convertSlice converts a slice of one type to a slice of another type using a
// converter function
func convertSlice[I any, O any](in []I, f func(I) O) []O {
	out := make([]O, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

func (r CCP) MarshalJSON() ([]byte, error) {
	var concrete struct {
		Source *dagger.Directory
	}
	concrete.Source = r.Source
	return json.Marshal(&concrete)
}

func (r *CCP) UnmarshalJSON(bs []byte) error {
	var concrete struct {
		Source *dagger.Directory
	}
	err := json.Unmarshal(bs, &concrete)
	if err != nil {
		return err
	}
	r.Source = concrete.Source
	return nil
}

func (r Build) MarshalJSON() ([]byte, error) {
	var concrete struct {
		Source *dagger.Directory
	}
	concrete.Source = r.Source
	return json.Marshal(&concrete)
}

func (r *Build) UnmarshalJSON(bs []byte) error {
	var concrete struct {
		Source *dagger.Directory
	}
	err := json.Unmarshal(bs, &concrete)
	if err != nil {
		return err
	}
	r.Source = concrete.Source
	return nil
}

func (r Lint) MarshalJSON() ([]byte, error) {
	var concrete struct {
		Source *dagger.Directory
	}
	concrete.Source = r.Source
	return json.Marshal(&concrete)
}

func (r *Lint) UnmarshalJSON(bs []byte) error {
	var concrete struct {
		Source *dagger.Directory
	}
	err := json.Unmarshal(bs, &concrete)
	if err != nil {
		return err
	}
	r.Source = concrete.Source
	return nil
}

func main() {
	ctx := context.Background()

	// Direct slog to the new stderr. This is only for dev time debugging, and
	// runtime errors/warnings.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))

	if err := dispatch(ctx); err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
}

func dispatch(ctx context.Context) error {
	ctx = telemetry.InitEmbedded(ctx, resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("dagger-go-sdk"),
		// TODO version?
	))
	defer telemetry.Close()

	// A lot of the "work" actually happens when we're marshalling the return
	// value, which entails getting object IDs, which happens in MarshalJSON,
	// which has no ctx argument, so we use this lovely global variable.
	setMarshalContext(ctx)

	fnCall := dag.CurrentFunctionCall()
	parentName, err := fnCall.ParentName(ctx)
	if err != nil {
		return fmt.Errorf("get parent name: %w", err)
	}
	fnName, err := fnCall.Name(ctx)
	if err != nil {
		return fmt.Errorf("get fn name: %w", err)
	}
	parentJson, err := fnCall.Parent(ctx)
	if err != nil {
		return fmt.Errorf("get fn parent: %w", err)
	}
	fnArgs, err := fnCall.InputArgs(ctx)
	if err != nil {
		return fmt.Errorf("get fn args: %w", err)
	}

	inputArgs := map[string][]byte{}
	for _, fnArg := range fnArgs {
		argName, err := fnArg.Name(ctx)
		if err != nil {
			return fmt.Errorf("get fn arg name: %w", err)
		}
		argValue, err := fnArg.Value(ctx)
		if err != nil {
			return fmt.Errorf("get fn arg value: %w", err)
		}
		inputArgs[argName] = []byte(argValue)
	}

	result, err := invoke(ctx, []byte(parentJson), parentName, fnName, inputArgs)
	if err != nil {
		return fmt.Errorf("invoke: %w", err)
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err = fnCall.ReturnValue(ctx, dagger.JSON(resultBytes)); err != nil {
		return fmt.Errorf("store return value: %w", err)
	}
	return nil
}
func invoke(ctx context.Context, parentJSON []byte, parentName string, fnName string, inputArgs map[string][]byte) (_ any, err error) {
	_ = inputArgs
	switch parentName {
	case "Build":
		switch fnName {
		case "WorkerImage":
			var parent Build
			err = json.Unmarshal(parentJSON, &parent)
			if err != nil {
				panic(fmt.Errorf("%s: %w", "failed to unmarshal parent object", err))
			}
			return (*Build).WorkerImage(&parent), nil
		case "CCPImage":
			var parent Build
			err = json.Unmarshal(parentJSON, &parent)
			if err != nil {
				panic(fmt.Errorf("%s: %w", "failed to unmarshal parent object", err))
			}
			return (*Build).CCPImage(&parent), nil
		case "MySQLContainer":
			var parent Build
			err = json.Unmarshal(parentJSON, &parent)
			if err != nil {
				panic(fmt.Errorf("%s: %w", "failed to unmarshal parent object", err))
			}
			return (*Build).MySQLContainer(&parent), nil
		default:
			return nil, fmt.Errorf("unknown function %s", fnName)
		}
	case "CCP":
		switch fnName {
		case "Build":
			var parent CCP
			err = json.Unmarshal(parentJSON, &parent)
			if err != nil {
				panic(fmt.Errorf("%s: %w", "failed to unmarshal parent object", err))
			}
			return (*CCP).Build(&parent), nil
		case "GenerateDumps":
			var parent CCP
			err = json.Unmarshal(parentJSON, &parent)
			if err != nil {
				panic(fmt.Errorf("%s: %w", "failed to unmarshal parent object", err))
			}
			return (*CCP).GenerateDumps(&parent, ctx)
		case "Etoe":
			var parent CCP
			err = json.Unmarshal(parentJSON, &parent)
			if err != nil {
				panic(fmt.Errorf("%s: %w", "failed to unmarshal parent object", err))
			}
			var test string
			if inputArgs["test"] != nil {
				err = json.Unmarshal([]byte(inputArgs["test"]), &test)
				if err != nil {
					panic(fmt.Errorf("%s: %w", "failed to unmarshal input arg test", err))
				}
			}
			var dbMode DatabaseExecutionMode
			if inputArgs["dbMode"] != nil {
				err = json.Unmarshal([]byte(inputArgs["dbMode"]), &dbMode)
				if err != nil {
					panic(fmt.Errorf("%s: %w", "failed to unmarshal input arg dbMode", err))
				}
			}
			return nil, (*CCP).Etoe(&parent, ctx, test, dbMode)
		case "Lint":
			var parent CCP
			err = json.Unmarshal(parentJSON, &parent)
			if err != nil {
				panic(fmt.Errorf("%s: %w", "failed to unmarshal parent object", err))
			}
			return (*CCP).Lint(&parent), nil
		case "":
			var parent CCP
			err = json.Unmarshal(parentJSON, &parent)
			if err != nil {
				panic(fmt.Errorf("%s: %w", "failed to unmarshal parent object", err))
			}
			var source *dagger.Directory
			if inputArgs["source"] != nil {
				err = json.Unmarshal([]byte(inputArgs["source"]), &source)
				if err != nil {
					panic(fmt.Errorf("%s: %w", "failed to unmarshal input arg source", err))
				}
			}
			var ref string
			if inputArgs["ref"] != nil {
				err = json.Unmarshal([]byte(inputArgs["ref"]), &ref)
				if err != nil {
					panic(fmt.Errorf("%s: %w", "failed to unmarshal input arg ref", err))
				}
			}
			return New(source, ref)
		default:
			return nil, fmt.Errorf("unknown function %s", fnName)
		}
	case "Lint":
		switch fnName {
		case "Go":
			var parent Lint
			err = json.Unmarshal(parentJSON, &parent)
			if err != nil {
				panic(fmt.Errorf("%s: %w", "failed to unmarshal parent object", err))
			}
			return (*Lint).Go(&parent), nil
		default:
			return nil, fmt.Errorf("unknown function %s", fnName)
		}
	case "":
		return dag.Module().
			WithObject(
				dag.TypeDef().WithObject("CCP").
					WithFunction(
						dag.Function("Build",
							dag.TypeDef().WithObject("Build"))).
					WithFunction(
						dag.Function("GenerateDumps",
							dag.TypeDef().WithObject("Directory"))).
					WithFunction(
						dag.Function("Etoe",
							dag.TypeDef().WithKind(dagger.VoidKind).WithOptional(true)).
							WithDescription("Run the e2e tests.\n\nThis function configures").
							WithArg("test", dag.TypeDef().WithKind(dagger.StringKind).WithOptional(true)).
							WithArg("dbMode", dag.TypeDef().WithEnum("DatabaseExecutionMode"), dagger.FunctionWithArgOpts{DefaultValue: dagger.JSON("\"USE_DUMPS\"")})).
					WithFunction(
						dag.Function("Lint",
							dag.TypeDef().WithObject("Lint"))).
					WithConstructor(
						dag.Function("New",
							dag.TypeDef().WithObject("CCP")).
							WithArg("source", dag.TypeDef().WithObject("Directory").WithOptional(true), dagger.FunctionWithArgOpts{Description: "Project source directory."}).
							WithArg("ref", dag.TypeDef().WithKind(dagger.StringKind).WithOptional(true), dagger.FunctionWithArgOpts{Description: "Checkout the repository (at the designated ref) and use it as the source\ndirectory instead of the local one."}))).
			WithObject(
				dag.TypeDef().WithObject("Build").
					WithFunction(
						dag.Function("WorkerImage",
							dag.TypeDef().WithObject("Container"))).
					WithFunction(
						dag.Function("CCPImage",
							dag.TypeDef().WithObject("Container"))).
					WithFunction(
						dag.Function("MySQLContainer",
							dag.TypeDef().WithObject("Container")))).
			WithEnum(
				dag.TypeDef().WithEnum("DatabaseExecutionMode", dagger.TypeDefWithEnumOpts{Description: "DatabaseExecutionMode defines the different modes in which the e2e tests can\noperate with the application databases."}).
					WithEnumValue("USE_DUMPS", dagger.TypeDefWithEnumValueOpts{Description: "UseDumps attempts to configure the MySQL service using the database dumps\npreviously generated."}).
					WithEnumValue("USE_CACHED", dagger.TypeDefWithEnumValueOpts{Description: "UseCached is the default mode that relies on whatever is the existing\nMySQL service state."}).
					WithEnumValue("FORCE_DROP", dagger.TypeDefWithEnumValueOpts{Description: "ForceDrop drops the existing databases forcing the application to\nrecreate them using Django migrations."})).
			WithObject(
				dag.TypeDef().WithObject("Lint").
					WithFunction(
						dag.Function("Go",
							dag.TypeDef().WithObject("Container"))).
					WithField("Source", dag.TypeDef().WithObject("Directory"))), nil
	default:
		return nil, fmt.Errorf("unknown object %s", parentName)
	}
}
