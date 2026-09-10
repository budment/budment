package bundler

import (
	"fmt"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

func BundleInMemory(entryPoint string) ([]byte, error) {
	options := api.BuildOptions{
		EntryPoints: []string{entryPoint},
		Bundle:      true,
		Write:       false,
		Format:      api.FormatIIFE,
		GlobalName:  "__BUDMENT_EXPORTS__",
		Platform:    api.PlatformNeutral,
		Target:      api.ES2017,
		Loader: map[string]api.Loader{
			".json": api.LoaderJSON,
			".ts":   api.LoaderTS,
			".js":   api.LoaderJS,
		},

		MinifyWhitespace:  true,
		MinifyIdentifiers: false,
		MinifySyntax:      true,
	}

	result := api.Build(options)

	if len(result.Errors) > 0 {
		var errMsgs []string
		for _, err := range result.Errors {
			if err.Location != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("%s:%d: %s", err.Location.File, err.Location.Line, err.Text))
			} else {
				errMsgs = append(errMsgs, fmt.Sprintf("esbuild error: %s", err.Text))
			}
		}
		return nil, fmt.Errorf("build errors:\n%s", strings.Join(errMsgs, "\n"))
	}

	if len(result.OutputFiles) == 0 {
		return nil, fmt.Errorf("esbuild returned no output")
	}

	return result.OutputFiles[0].Contents, nil
}
