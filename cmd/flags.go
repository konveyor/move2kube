/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

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
	// profileTypeFlag is the name of the flag that contains the type of profiling that should be performed
	profileTypeFlag = "profile-type"
	// profileIntervalFlag is the name of the flag that contains the frequency with which we profile the heap memory
	profileIntervalFlag = "profile-interval-ms"
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
	customizationsFlag       = "customizations"
	qadisablecliFlag         = "qa-disable-cli"
	qaportFlag               = "qa-port"
	planProgressPortFlag     = "plan-progress-port"
	maxCloneSizeBytesFlag    = "max-clone-size"
	transformerSelectorFlag  = "transformer-selector"
	qaEnabledCategoriesFlag  = "qa-enable"
	qaDisabledCategoriesFlag = "qa-disable"
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
	// qaEnabledCategories contains list of categories to be enabled
	qaEnabledCategories []string
	// qaDisabledCategories contains list of categories to be disabled
	qaDisabledCategories []string
}
