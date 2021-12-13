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

// TektonResources contains the resources for CI/CD
type TektonResources struct {
	EventListeners   []EventListener
	TriggerBindings  []TriggerBinding
	TriggerTemplates []TriggerTemplate
	Pipelines        []Pipeline
}

// Ingress holds the details about the git event ingress resource
// type Ingress struct {
// 	Name              string
// 	EventListenerName string
// 	HostName          string
// 	Port              int32
// 	ServiceName       string
// 	TLSSecretName     string
// }

// EventListener holds the details about the git event listener resource
type EventListener struct {
	Name                string
	ServiceAccountName  string
	TriggerBindingName  string
	TriggerTemplateName string
}

// TriggerBinding holds the details about the git event trigger binding resource
type TriggerBinding struct {
	Name string
}

// TriggerTemplate holds the details about the git event trigger template resource
type TriggerTemplate struct {
	Name               string
	PipelineName       string
	PipelineRunName    string
	ServiceAccountName string
	WorkspaceName      string
	StorageClassName   string
}

// Pipeline holds the details about the clone build push pipeline resource
type Pipeline struct {
	Name          string
	WorkspaceName string
}
