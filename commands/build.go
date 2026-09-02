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

func buildCommand(site *site.Site) error {
	watch := site.Config().Watch

	logger.path("Destination:", site.DestDir())
	logger.label("Generating...", "")
	var (
		count int
		err   error
	)
	if site.Config().Enabled() {
		count, err = site.LocalizedBuild(site)
	} else {
		count, err = site.Write()
	}
	switch {
	case err == nil:
		elapsed := time.Since(commandStartTime)
		logger.label("", "wrote %d files in %.2fs.", count, elapsed.Seconds())
		diag := site.Diagnostics()
		diag.FilesOutput = count
		if site.Config().Verbose || diag.FilesExcluded+diag.FilesStaticNoFM+diag.FilesUnpublished > 0 {
			logger.label("Diagnostics:", "%s", diag.DiagSummary())
		}
	case watch:
		fmt.Fprintln(os.Stderr, err)
	default:
		return err
	}

	// A localized watcher must rebuild one project generation. The existing
	// site watcher is intentionally single-site, so do not start it here.
	if watch && site.Config().Enabled() {
		return fmt.Errorf("localized watch is not supported by the single-site watcher")
	}

	// FIXME the watch will miss files that changed during the first build
	// server watch is implemented inside Server.Run, in contrast to this command
	if watch {
		events, err := site.WatchRebuild()
		if err != nil {
			return err
		}
		logger.label("Auto-regeneration:", "enabled for %q", site.SourceDir())
		for event := range events {
			fmt.Print(event)
		}
	} else {
		logger.label("Auto-regeneration:", "disabled. Use --watch to enable.")
	}
	return nil
}
