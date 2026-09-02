package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/reidransom/jigyll/site"
)

// main sets this
var commandStartTime = time.Now()

var build = app.Command("build", "Build your site").Alias("b")

func init() {
	build.Flag("dry-run", "Dry run").Short('n').BoolVar(&options.DryRun)
}

func buildCommand(s *site.Site) error {
	watch := s.Config().Watch

	logger.path("Destination:", s.DestDir())
	logger.label("Generating...", "")
	var (
		count   int
		err     error
		project *site.LocalizedProject
	)
	if s.Config().Enabled() {
		project, count, err = site.BuildLocalizedProject(s)
	} else {
		count, err = s.Write()
	}
	switch {
	case err == nil:
		elapsed := time.Since(commandStartTime)
		logger.label("", "wrote %d files in %.2fs.", count, elapsed.Seconds())
		diag := s.Diagnostics()
		diag.FilesOutput = count
		if s.Config().Verbose || diag.FilesExcluded+diag.FilesStaticNoFM+diag.FilesUnpublished > 0 {
			logger.label("Diagnostics:", "%s", diag.DiagSummary())
		}
	case watch:
		fmt.Fprintln(os.Stderr, err)
	default:
		return err
	}

	if watch && s.Config().Enabled() {
		if project == nil {
			return err
		}
		return watchLocalizedProject(project)
	}

	// FIXME the watch will miss files that changed during the first build
	// server watch is implemented inside Server.Run, in contrast to this command
	if watch {
		events, err := s.WatchRebuild()
		if err != nil {
			return err
		}
		logger.label("Auto-regeneration:", "enabled for %q", s.SourceDir())
		for event := range events {
			fmt.Print(event)
		}
	} else {
		logger.label("Auto-regeneration:", "disabled. Use --watch to enable.")
	}
	return nil
}

// watchLocalizedProject observes one project source and replaces only complete
// published generations. It intentionally never enters the single-site
// incremental reload path.
func watchLocalizedProject(project *site.LocalizedProject) error {
	events, err := project.WatchFiles()
	if err != nil {
		return err
	}
	logger.label("Auto-regeneration:", "enabled for %q", project.Config().Source)
	for event := range events {
		fmt.Printf("Regenerating: %s...", event)
		start := time.Now()
		replacement, count, err := project.Rebuild()
		if err != nil {
			fmt.Println()
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		project = replacement
		logger.label("", "wrote %d files in %.2fs.", count, time.Since(start).Seconds())
	}
	return nil
}
