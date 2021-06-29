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

package ir

// EnhancedIR is IR with extra data specific to API resource sets
type EnhancedIR struct {
	IR
	Roles           []Role
	RoleBindings    []RoleBinding
	ServiceAccounts []ServiceAccount
	BuildConfigs    []BuildConfig
	TektonResources TektonResources
}

// ServiceAccount holds the details about the service account resource
type ServiceAccount struct {
	Name        string
	SecretNames []string
}

// RoleBinding holds the details about the role binding resource
type RoleBinding struct {
	Name               string
	RoleName           string
	ServiceAccountName string
}

// Role holds the details about the role resource
type Role struct {
	Name        string
	PolicyRules []PolicyRule
}

// PolicyRule holds the details about the policy rules for the service account resources
type PolicyRule struct {
	APIGroups []string
	Resources []string
	Verbs     []string
}

// BuildConfig contains the resources needed to create a BuildConfig
type BuildConfig struct {
	Name              string
	ImageStreamName   string
	ImageStreamTag    string
	SourceSecretName  string
	WebhookSecretName string
	ContainerBuild    ContainerBuild
}
