package dotnetcoresdk

import (
	"io"
	"strconv"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type LogEmitter struct {
	// Emitter is embedded and therefore delegates all of its functions to the
	// LogEmitter.
	scribe.Emitter
}

func NewLogEmitter(output io.Writer) LogEmitter {
	return LogEmitter{
		Emitter: scribe.NewEmitter(output),
	}
}

func (e LogEmitter) Candidates(entries []packit.BuildpackPlanEntry) {
	e.Subprocess("Candidate version sources (in priority order):")

	var (
		sources [][2]string
		maxLen  int
	)

	for _, entry := range entries {
		versionSource, ok := entry.Metadata["version-source"].(string)
		if !ok {
			versionSource = "<unknown>"
		}

		if len(versionSource) > maxLen {
			maxLen = len(versionSource)
		}

		version, ok := entry.Metadata["version"].(string)
		if version == "" || !ok {
			version = "*"
		}

		sources = append(sources, [2]string{versionSource, version})
	}

	for _, source := range sources {
		e.Action(("%-" + strconv.Itoa(maxLen) + "s -> %q"), source[0], source[1])
	}

	e.Break()
}

func (l LogEmitter) Environment(env packit.Environment) {
	l.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(env))
	l.Break()
}

func (l LogEmitter) SelectedDependency(entry packit.BuildpackPlanEntry, dependency postal.Dependency, now time.Time) {
	dependency.Name = ".NET Core SDK"
	l.Emitter.SelectedDependency(entry, dependency, now)
}
