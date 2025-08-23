// Package paths contains constants for common directory and file paths used by bumpers.
package constants

const (
	// ClaudeDir is the primary Claude configuration directory name (.claude).
	ClaudeDir = ".claude"

	// AppSubDir is the bumpers-specific subdirectory within Claude directory.
	AppSubDir = "bumpers"

	// LogFilename is the default log file name for bumpers.
	LogFilename = "bumpers.log"

	// CacheFilename is the default cache database file name for bumpers.
	CacheFilename = "cache.db"

	// SettingsFilename is the Claude settings file name that bumpers modifies.
	SettingsFilename = "settings.local.json"
)
