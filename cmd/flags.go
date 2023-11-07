package cmd

const (
	// sourceFlag is the name of the flag that contains path to the source folder
	sourceFlag = "source"
	// outputFlag is the name of the flag that contains path to the output folder
	outputFlag = "output"
	// nameFlag is the name of the flag that contains the project name
	nameFlag = "name"
	// planFlag is the name of the flag that contains the path to the plan file
	planFlag = "plan"
	// profileFlag is the name of the flag that contains the path where the CPU profile file should be generated
	profileFlag = "profile"
	// ignoreEnvFlag is the name of the flag that tells us whether to use data collected from the local machine
	ignoreEnvFlag = "ignore-env"
	// qaSkipFlag is the name of the flag that lets you skip all the question answers
	qaSkipFlag = "qa-skip"
	// qaPersistPasswords is the name of the flag that lets choose to persist passwords
	qaPersistPasswords = "qa-persist-passwords"
	// configOutFlag is the name of the flag that will point the location to output the config file
	configOutFlag = "config-out"
	// qaCacheOutFlag is the name of the flag that will point the location to output the cache file
	qaCacheOutFlag = "qa-cache-out"
	// configFlag is the name of the flag that contains list of config files
	configFlag = "config"
	// setConfigFlag is the name of the flag that contains list of key-value configs
	setConfigFlag = "set-config"
	// preSetFlag is the name of the flag that contains list of preset configurations to use
	preSetFlag = "preset"
	// overwriteFlag is the name of the flag that lets you overwrite the output directory if it exists
	overwriteFlag = "overwrite"
	// maxIterationsFlag is the name of the flag that lets you set the maximum number of iterations to allow
	maxIterationsFlag = "max-iterations"
	// customizationsFlag is the path to customizations directory
	customizationsFlag      = "customizations"
	qadisablecliFlag        = "qa-disable-cli"
	qaportFlag              = "qa-port"
	planProgressPortFlag    = "plan-progress-port"
	transformerSelectorFlag = "transformer-selector"
)

type qaflags struct {
	// qadisablecli disables the CLI engine. To be used with HTTP REST engine
	qadisablecli bool
	// qaport contains the port where the Question Answer HTTP REST engine server is started
	qaport int
	// configOut contains the location to output the config
	configOut string
	// qaCacheOut contains the location to output the cache
	qaCacheOut string
	// configs contains a list of config files
	configs []string
	// Configs contains a list of key-value configs
	setconfigs []string
	// qaskip lets you skip all the question answers
	qaskip bool
	// preSets contains a list of preset configurations
	preSets []string
	// persistPasswords sets whether to persist the password or not
	persistPasswords bool
}
