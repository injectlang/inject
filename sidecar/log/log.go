package log

// 
// This library is optional.  Logging will work without it.
//
// Import this if you'd like support for setting log level with the environment variable
// "LOG_LEVEL"
//
// to import, import like so:
//
// import (
//     	_ "github.com/ryanchapman/config-container/sidecar/log"
// )
//

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	newLevelStr := os.Getenv("LOG_LEVEL")
	if newLevelStr != "" {
		prevLevel := zerolog.GlobalLevel()
		newLevel, err := zerolog.ParseLevel(newLevelStr)
		if err != nil {
			log.Warn().Msgf("could not get parse log level %d: %s", prevLevel, err)
			return
		}
		changeLogLevel(prevLevel, newLevel)
	}
}

func changeLogLevel(oldLevel, newLevel zerolog.Level) {
	if newLevel == oldLevel {
		// do nothing
		return
	}

	zerolog.SetGlobalLevel(newLevel)
	log.Info().Msgf("changed log level from %s to %s", oldLevel.String(), newLevel.String())
}
