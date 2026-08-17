package plugins

func init() {
	// The tag is registered in tags.AddJekyllTags, which owns the site and theme
	// include directories. This entry marks the plugin as emulated.
	register("jekyll-include-cache", plugin{})
}
