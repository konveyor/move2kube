// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots at
// 2021-01-14 19:15:18.288046 +0900 JST m=+0.007042331

/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clusters

var Constants = map[string]string{

	`aws-eks_yaml`: `apiVersion: move2kube.konveyor.io/v1alpha1
kind: ClusterMetadata
metadata:
  name: AWS-EKS
spec:
  storageClasses:
    - gp2
  apiKindVersionMap:
    APIService:
      - apiregistration.k8s.io/v1
      - apiregistration.k8s.io/v1beta1
    Binding:
      - v1
    CSIDriver:
      - storage.k8s.io/v1beta1
    CSINode:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    CertificateSigningRequest:
      - certificates.k8s.io/v1beta1
    ClusterRole:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ClusterRoleBinding:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ComponentStatus:
      - v1
    ConfigMap:
      - v1
    ControllerRevision:
      - apps/v1
    CronJob:
      - batch/v1beta1
    CustomResourceDefinition:
      - apiextensions.k8s.io/v1
      - apiextensions.k8s.io/v1beta1
    DaemonSet:
      - apps/v1
    Deployment:
      - apps/v1
    ENIConfig:
      - crd.k8s.amazonaws.com/v1alpha1
    EndpointSlice:
      - discovery.k8s.io/v1beta1
    Endpoints:
      - v1
    Event:
      - events.k8s.io/v1beta1
      - v1
    Eviction:
      - v1
    HorizontalPodAutoscaler:
      - autoscaling/v1
      - autoscaling/v2beta1
      - autoscaling/v2beta2
    Ingress:
      - networking.k8s.io/v1beta1
      - extensions/v1beta1
    Job:
      - batch/v1
    Lease:
      - coordination.k8s.io/v1
      - coordination.k8s.io/v1beta1
    LimitRange:
      - v1
    LocalSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    MutatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1
      - admissionregistration.k8s.io/v1beta1
    Namespace:
      - v1
    NetworkPolicy:
      - networking.k8s.io/v1
    Node:
      - v1
    NodeProxyOptions:
      - v1
    PersistentVolume:
      - v1
    PersistentVolumeClaim:
      - v1
    Pod:
      - v1
    PodAttachOptions:
      - v1
    PodDisruptionBudget:
      - policy/v1beta1
    PodExecOptions:
      - v1
    PodPortForwardOptions:
      - v1
    PodProxyOptions:
      - v1
    PodSecurityPolicy:
      - policy/v1beta1
    PodTemplate:
      - v1
    PriorityClass:
      - scheduling.k8s.io/v1
      - scheduling.k8s.io/v1beta1
    ReplicaSet:
      - apps/v1
    ReplicationController:
      - v1
    ResourceQuota:
      - v1
    Role:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RoleBinding:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RuntimeClass:
      - node.k8s.io/v1beta1
    Scale:
      - apps/v1
      - v1
    Secret:
      - v1
    SecurityGroupPolicy:
      - vpcresources.k8s.aws/v1beta1
    SelfSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    SelfSubjectRulesReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    Service:
      - v1
    ServiceAccount:
      - v1
    ServiceProxyOptions:
      - v1
    StatefulSet:
      - apps/v1
    StorageClass:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    SubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    TokenRequest:
      - v1
    TokenReview:
      - authentication.k8s.io/v1
      - authentication.k8s.io/v1beta1
    ValidatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1
      - admissionregistration.k8s.io/v1beta1
    VolumeAttachment:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
`,

	`azure-aks_yaml`: `apiVersion: move2kube.konveyor.io/v1alpha1
kind: ClusterMetadata
metadata:
  name: Azure-AKS
spec:
  storageClasses:
    - azurefile
    - azurefile-premium
    - default
    - managed-premium
  apiKindVersionMap:
    APIService:
      - apiregistration.k8s.io/v1
      - apiregistration.k8s.io/v1beta1
    Binding:
      - v1
    CSIDriver:
      - storage.k8s.io/v1beta1
    CSINode:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    CertificateSigningRequest:
      - certificates.k8s.io/v1beta1
    ClusterRole:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ClusterRoleBinding:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ComponentStatus:
      - v1
    ConfigMap:
      - v1
    ControllerRevision:
      - apps/v1
    CronJob:
      - batch/v1beta1
    CustomResourceDefinition:
      - apiextensions.k8s.io/v1
      - apiextensions.k8s.io/v1beta1
    DaemonSet:
      - apps/v1
    Deployment:
      - apps/v1
    EndpointSlice:
      - discovery.k8s.io/v1beta1
    Endpoints:
      - v1
    Event:
      - events.k8s.io/v1beta1
      - v1
    Eviction:
      - v1
    HealthState:
      - azmon.container.insights/v1
    HorizontalPodAutoscaler:
      - autoscaling/v1
      - autoscaling/v2beta1
      - autoscaling/v2beta2
    Ingress:
      - networking.k8s.io/v1beta1
      - extensions/v1beta1
    Job:
      - batch/v1
    Lease:
      - coordination.k8s.io/v1
      - coordination.k8s.io/v1beta1
    LimitRange:
      - v1
    LocalSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    MutatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1
      - admissionregistration.k8s.io/v1beta1
    Namespace:
      - v1
    NetworkPolicy:
      - networking.k8s.io/v1
    Node:
      - v1
    NodeMetrics:
      - metrics.k8s.io/v1beta1
    NodeProxyOptions:
      - v1
    PersistentVolume:
      - v1
    PersistentVolumeClaim:
      - v1
    Pod:
      - v1
    PodAttachOptions:
      - v1
    PodDisruptionBudget:
      - policy/v1beta1
    PodExecOptions:
      - v1
    PodMetrics:
      - metrics.k8s.io/v1beta1
    PodPortForwardOptions:
      - v1
    PodProxyOptions:
      - v1
    PodSecurityPolicy:
      - policy/v1beta1
    PodTemplate:
      - v1
    PriorityClass:
      - scheduling.k8s.io/v1
      - scheduling.k8s.io/v1beta1
    ReplicaSet:
      - apps/v1
    ReplicationController:
      - v1
    ResourceQuota:
      - v1
    Role:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RoleBinding:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RuntimeClass:
      - node.k8s.io/v1beta1
    Scale:
      - apps/v1
      - v1
    Secret:
      - v1
    SelfSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    SelfSubjectRulesReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    Service:
      - v1
    ServiceAccount:
      - v1
    ServiceProxyOptions:
      - v1
    StatefulSet:
      - apps/v1
    StorageClass:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    SubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    TokenRequest:
      - v1
    TokenReview:
      - authentication.k8s.io/v1
      - authentication.k8s.io/v1beta1
    ValidatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1
      - admissionregistration.k8s.io/v1beta1
    VolumeAttachment:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
`,

	`gcp-gke_yaml`: `apiVersion: move2kube.konveyor.io/v1alpha1
kind: ClusterMetadata
metadata:
  name: GCP-GKE
spec:
  storageClasses:
    - standard
  apiKindVersionMap:
    APIService:
      - apiregistration.k8s.io/v1
    BackendConfig:
      - cloud.google.com/v1
    Binding:
      - v1
    CSIDriver:
      - storage.k8s.io/v1beta1
    CSINode:
      - storage.k8s.io/v1beta1
    CertificateSigningRequest:
      - certificates.k8s.io/v1beta1
    ClusterRole:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ClusterRoleBinding:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ComponentStatus:
      - v1
    ConfigMap:
      - v1
    ControllerRevision:
      - apps/v1
    CronJob:
      - batch/v1beta1
    CustomResourceDefinition:
      - apiextensions.k8s.io/v1
    DaemonSet:
      - apps/v1
    Deployment:
      - apps/v1
    Endpoints:
      - v1
    Event:
      - v1
    HorizontalPodAutoscaler:
      - autoscaling/v1
      - autoscaling/v2beta1
      - autoscaling/v2beta2
    Ingress:
      - networking.k8s.io/v1beta1
      - extensions/v1beta1
    Job:
      - batch/v1
    Lease:
      - coordination.k8s.io/v1beta1
      - coordination.k8s.io/v1
    LimitRange:
      - v1
    LocalSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    ManagedCertificate:
      - networking.gke.io/v1beta2
    MutatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1beta1
      - admissionregistration.k8s.io/v1
    Namespace:
      - v1
    NetworkPolicy:
      - networking.k8s.io/v1
    Node:
      - v1
      - v1
    PersistentVolume:
      - v1
    PersistentVolumeClaim:
      - v1
    Pod:
      - v1
      - v1
    PodDisruptionBudget:
      - policy/v1beta1
    PodSecurityPolicy:
      - policy/v1beta1
    PodTemplate:
      - v1
    PriorityClass:
      - scheduling.k8s.io/v1beta1
      - scheduling.k8s.io/v1
    ReplicaSet:
      - apps/v1
    ReplicationController:
      - v1
    ResourceQuota:
      - v1
    Role:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RoleBinding:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RuntimeClass:
      - node.k8s.io/v1beta1
    ScalingPolicy:
      - scalingpolicy.kope.io/v1alpha1
    Secret:
      - v1
    SelfSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    SelfSubjectRulesReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    Service:
      - v1
    ServiceAccount:
      - v1
    StatefulSet:
      - apps/v1
    StorageClass:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    StorageState:
      - migration.k8s.io/v1alpha1
    StorageVersionMigration:
      - migration.k8s.io/v1alpha1
    SubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    TokenReview:
      - authentication.k8s.io/v1
      - authentication.k8s.io/v1beta1
    UpdateInfo:
      - nodemanagement.gke.io/v1alpha1
    ValidatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1beta1
      - admissionregistration.k8s.io/v1
    VolumeAttachment:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
`,

	`ibm-iks_yaml`: `apiVersion: move2kube.konveyor.io/v1alpha1
kind: ClusterMetadata
metadata:
  name: IBM-IKS
spec:
  storageClasses:
    - default
    - ibmc-block-bronze
    - ibmc-block-custom
    - ibmc-block-gold
    - ibmc-block-retain-bronze
    - ibmc-block-retain-custom
    - ibmc-block-retain-gold
    - ibmc-block-retain-silver
    - ibmc-block-silver
    - ibmc-file-bronze
    - ibmc-file-bronze-gid
    - ibmc-file-custom
    - ibmc-file-gold
    - ibmc-file-gold-gid
    - ibmc-file-retain-bronze
    - ibmc-file-retain-custom
    - ibmc-file-retain-gold
    - ibmc-file-retain-silver
    - ibmc-file-silver
    - ibmc-file-silver-gid
  apiKindVersionMap:
    APIService:
      - apiregistration.k8s.io/v1
    BGPConfiguration:
      - crd.projectcalico.org/v1
    BGPPeer:
      - crd.projectcalico.org/v1
    Binding:
      - v1
    BlockAffinity:
      - crd.projectcalico.org/v1
    CSIDriver:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    CSINode:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    CatalogSource:
      - operators.coreos.com/v1alpha1
    CertificateSigningRequest:
      - certificates.k8s.io/v1beta1
    ClusterInformation:
      - crd.projectcalico.org/v1
    ClusterRole:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ClusterRoleBinding:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ClusterServiceVersion:
      - operators.coreos.com/v1alpha1
    ComponentStatus:
      - v1
    ConfigMap:
      - v1
    ControllerRevision:
      - apps/v1
    CronJob:
      - batch/v1beta1
      - batch/v2alpha1
    CustomResourceDefinition:
      - apiextensions.k8s.io/v1
    DaemonSet:
      - apps/v1
    Deployment:
      - apps/v1
    EndpointSlice:
      - discovery.k8s.io/v1beta1
    Endpoints:
      - v1
    Event:
      - events.k8s.io/v1beta1
      - v1
    FelixConfiguration:
      - crd.projectcalico.org/v1
    GlobalNetworkPolicy:
      - crd.projectcalico.org/v1
    GlobalNetworkSet:
      - crd.projectcalico.org/v1
    HorizontalPodAutoscaler:
      - autoscaling/v1
      - autoscaling/v2beta1
      - autoscaling/v2beta2
    HostEndpoint:
      - crd.projectcalico.org/v1
    IPAMBlock:
      - crd.projectcalico.org/v1
    IPAMConfig:
      - crd.projectcalico.org/v1
    IPAMHandle:
      - crd.projectcalico.org/v1
    IPPool:
      - crd.projectcalico.org/v1
    Ingress:
      - networking.k8s.io/v1
      - networking.k8s.io/v1beta1
      - extensions/v1beta1
    IngressClass:
      - networking.k8s.io/v1
      - networking.k8s.io/v1beta1
    InstallPlan:
      - operators.coreos.com/v1alpha1
    Job:
      - batch/v1
    KubeControllersConfiguration:
      - crd.projectcalico.org/v1
    Lease:
      - coordination.k8s.io/v1beta1
      - coordination.k8s.io/v1
    LimitRange:
      - v1
    LocalSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    MutatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1beta1
      - admissionregistration.k8s.io/v1
    Namespace:
      - v1
    NetworkPolicy:
      - networking.k8s.io/v1
    NetworkSet:
      - crd.projectcalico.org/v1
    Node:
      - v1
    Operator:
      - operators.coreos.com/v1
    OperatorGroup:
      - operators.coreos.com/v1
    PersistentVolume:
      - v1
    PersistentVolumeClaim:
      - v1
    Pod:
      - v1
    PodDisruptionBudget:
      - policy/v1beta1
    PodSecurityPolicy:
      - policy/v1beta1
    PodTemplate:
      - v1
    PriorityClass:
      - scheduling.k8s.io/v1beta1
      - scheduling.k8s.io/v1
    RBACSync:
      - ibm.com/v1alpha1
    ReplicaSet:
      - apps/v1
    ReplicationController:
      - v1
    ResourceQuota:
      - v1
    Role:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RoleBinding:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    Secret:
      - v1
    SelfSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    SelfSubjectRulesReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    Service:
      - v1
    ServiceAccount:
      - v1
    StatefulSet:
      - apps/v1
    StorageClass:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    SubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    Subscription:
      - operators.coreos.com/v1alpha1
    TokenReview:
      - authentication.k8s.io/v1
      - authentication.k8s.io/v1beta1
    ValidatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1beta1
      - admissionregistration.k8s.io/v1
    VolumeAttachment:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
`,

	`ibm-openshift_yaml`: `apiVersion: move2kube.konveyor.io/v1alpha1
kind: ClusterMetadata
metadata: 
  name: IBM-Openshift
spec:
  storageClasses:
    - default
    - ibmc-block-bronze
    - ibmc-block-custom
    - ibmc-block-gold
    - ibmc-block-retain-bronze
    - ibmc-block-retain-custom
    - ibmc-block-retain-gold
    - ibmc-block-retain-silver
    - ibmc-block-silver
    - ibmc-file-bronze
    - ibmc-file-bronze-gid
    - ibmc-file-custom
    - ibmc-file-gold
    - ibmc-file-gold-gid
    - ibmc-file-retain-bronze
    - ibmc-file-retain-custom
    - ibmc-file-retain-gold
    - ibmc-file-retain-silver
    - ibmc-file-silver
    - ibmc-file-silver-gid
  apiKindVersionMap:
    APIService:
      - apiregistration.k8s.io/v1
      - apiregistration.k8s.io/v1beta1
    Alertmanager:
      - monitoring.coreos.com/v1
    AppliedClusterResourceQuota:
      - quota.openshift.io/v1
    BinaryBuildRequestOptions:
      - build.openshift.io/v1
    Binding:
      - v1
    BrokerTemplateInstance:
      - template.openshift.io/v1
    Build:
      - build.openshift.io/v1
    BuildConfig:
      - build.openshift.io/v1
    BuildLog:
      - build.openshift.io/v1
    BuildRequest:
      - build.openshift.io/v1
    Bundle:
      - automationbroker.io/v1alpha1
    BundleBinding:
      - automationbroker.io/v1alpha1
    BundleInstance:
      - automationbroker.io/v1alpha1
    CertificateSigningRequest:
      - certificates.k8s.io/v1beta1
    ClusterNetwork:
      - network.openshift.io/v1
    ClusterResourceQuota:
      - quota.openshift.io/v1
    ClusterRole:
      - authorization.openshift.io/v1
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ClusterRoleBinding:
      - authorization.openshift.io/v1
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ClusterServiceBroker:
      - servicecatalog.k8s.io/v1beta1
    ClusterServiceClass:
      - servicecatalog.k8s.io/v1beta1
    ClusterServicePlan:
      - servicecatalog.k8s.io/v1beta1
    ComponentStatus:
      - v1
    ConfigMap:
      - v1
    ControllerRevision:
      - apps/v1
      - apps/v1beta1
      - apps/v1beta2
    CronJob:
      - batch/v1beta1
    CustomResourceDefinition:
      - apiextensions.k8s.io/v1beta1
    DaemonSet:
      - apps/v1
      - apps/v1beta2
      - extensions/v1beta1
    Deployment:
      - apps/v1
      - apps/v1beta1
      - apps/v1beta2
      - extensions/v1beta1
    DeploymentConfig:
      - apps.openshift.io/v1
    DeploymentConfigRollback:
      - apps.openshift.io/v1
    DeploymentLog:
      - apps.openshift.io/v1
    DeploymentRequest:
      - apps.openshift.io/v1
    DeploymentRollback:
      - apps/v1beta1
      - extensions/v1beta1
    EgressNetworkPolicy:
      - network.openshift.io/v1
    Endpoints:
      - v1
    Event:
      - events.k8s.io/v1beta1
      - v1
    Eviction:
      - v1
    Group:
      - user.openshift.io/v1
    HorizontalPodAutoscaler:
      - autoscaling/v1
      - autoscaling/v2beta1
    HostSubnet:
      - network.openshift.io/v1
    Identity:
      - user.openshift.io/v1
    Image:
      - image.openshift.io/v1
    ImageSignature:
      - image.openshift.io/v1
    ImageStream:
      - image.openshift.io/v1
    ImageStreamImage:
      - image.openshift.io/v1
    ImageStreamImport:
      - image.openshift.io/v1
    ImageStreamLayers:
      - image.openshift.io/v1
    ImageStreamMapping:
      - image.openshift.io/v1
    ImageStreamTag:
      - image.openshift.io/v1
    Ingress:
      - extensions/v1beta1
    Job:
      - batch/v1
    LimitRange:
      - v1
    LocalResourceAccessReview:
      - authorization.openshift.io/v1
    LocalSubjectAccessReview:
      - authorization.openshift.io/v1
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    MutatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1beta1
    Namespace:
      - v1
    NetNamespace:
      - network.openshift.io/v1
    NetworkPolicy:
      - networking.k8s.io/v1
      - extensions/v1beta1
    Node:
      - v1
    OAuthAccessToken:
      - oauth.openshift.io/v1
    OAuthAuthorizeToken:
      - oauth.openshift.io/v1
    OAuthClient:
      - oauth.openshift.io/v1
    OAuthClientAuthorization:
      - oauth.openshift.io/v1
    PersistentVolume:
      - v1
    PersistentVolumeClaim:
      - v1
    Pod:
      - v1
    PodDisruptionBudget:
      - policy/v1beta1
    PodSecurityPolicy:
      - extensions/v1beta1
      - policy/v1beta1
    PodSecurityPolicyReview:
      - security.openshift.io/v1
    PodSecurityPolicySelfSubjectReview:
      - security.openshift.io/v1
    PodSecurityPolicySubjectReview:
      - security.openshift.io/v1
    PodTemplate:
      - v1
    PriorityClass:
      - scheduling.k8s.io/v1beta1
    Project:
      - project.openshift.io/v1
    ProjectRequest:
      - project.openshift.io/v1
    Prometheus:
      - monitoring.coreos.com/v1
    PrometheusRule:
      - monitoring.coreos.com/v1
    RangeAllocation:
      - security.openshift.io/v1
    ReplicaSet:
      - apps/v1
      - apps/v1beta2
      - extensions/v1beta1
    ReplicationController:
      - v1
    ReplicationControllerDummy:
      - extensions/v1beta1
    ResourceAccessReview:
      - authorization.openshift.io/v1
    ResourceQuota:
      - v1
    Role:
      - authorization.openshift.io/v1
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RoleBinding:
      - authorization.openshift.io/v1
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RoleBindingRestriction:
      - authorization.openshift.io/v1
    Route:
      - route.openshift.io/v1
    Scale:
      - apps.openshift.io/v1
      - apps/v1
      - apps/v1beta1
      - apps/v1beta2
      - extensions/v1beta1
      - v1
    Secret:
      - v1
    SecretList:
      - image.openshift.io/v1
    SecurityContextConstraints:
      - security.openshift.io/v1
      - v1
    SelfSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    SelfSubjectRulesReview:
      - authorization.openshift.io/v1
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    Service:
      - v1
    ServiceAccount:
      - v1
    ServiceBinding:
      - servicecatalog.k8s.io/v1beta1
    ServiceBroker:
      - servicecatalog.k8s.io/v1beta1
    ServiceClass:
      - servicecatalog.k8s.io/v1beta1
    ServiceInstance:
      - servicecatalog.k8s.io/v1beta1
    ServiceMonitor:
      - monitoring.coreos.com/v1
    ServicePlan:
      - servicecatalog.k8s.io/v1beta1
    StatefulSet:
      - apps/v1
      - apps/v1beta1
      - apps/v1beta2
    StorageClass:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    SubjectAccessReview:
      - authorization.openshift.io/v1
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    SubjectRulesReview:
      - authorization.openshift.io/v1
    Template:
      - template.openshift.io/v1
    TemplateInstance:
      - template.openshift.io/v1
    TokenReview:
      - authentication.k8s.io/v1
      - authentication.k8s.io/v1beta1
    User:
      - user.openshift.io/v1
    UserIdentityMapping:
      - user.openshift.io/v1
    ValidatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1beta1
    VolumeAttachment:
      - storage.k8s.io/v1beta1
`,

	`kubernetes_yaml`: `apiVersion: move2kube.konveyor.io/v1alpha1
kind: ClusterMetadata
metadata:
  name: Kubernetes
spec:
  storageClasses:
    - default
  apiKindVersionMap:
    APIService:
      - apiregistration.k8s.io/v1
    Binding:
      - v1
    CSIDriver:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    CSINode:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    CertificateSigningRequest:
      - certificates.k8s.io/v1
      - certificates.k8s.io/v1beta1
    ClusterRole:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ClusterRoleBinding:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ComponentStatus:
      - v1
    ConfigMap:
      - v1
    ControllerRevision:
      - apps/v1
    CronJob:
      - batch/v1beta1
      - batch/v2alpha1
    CustomResourceDefinition:
      - apiextensions.k8s.io/v1
    DaemonSet:
      - apps/v1
    Deployment:
      - apps/v1
    EndpointSlice:
      - discovery.k8s.io/v1beta1
    Endpoints:
      - v1
    Event:
      - events.k8s.io/v1beta1
      - v1
    HorizontalPodAutoscaler:
      - autoscaling/v1
      - autoscaling/v2beta1
      - autoscaling/v2beta2
    Ingress:
      - networking.k8s.io/v1
      - networking.k8s.io/v1beta1
      - extensions/v1beta1
    IngressClass:
      - networking.k8s.io/v1
      - networking.k8s.io/v1beta1
    Job:
      - batch/v1
    Lease:
      - coordination.k8s.io/v1beta1
      - coordination.k8s.io/v1
    LimitRange:
      - v1
    LocalSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    MutatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1beta1
      - admissionregistration.k8s.io/v1
    Namespace:
      - v1
    NetworkPolicy:
      - networking.k8s.io/v1
    Node:
      - v1
    PersistentVolume:
      - v1
    PersistentVolumeClaim:
      - v1
    Pod:
      - v1
    PodDisruptionBudget:
      - policy/v1beta1
    PodSecurityPolicy:
      - policy/v1beta1
    PodTemplate:
      - v1
    PriorityClass:
      - scheduling.k8s.io/v1beta1
      - scheduling.k8s.io/v1
    ReplicaSet:
      - apps/v1
    ReplicationController:
      - v1
    ResourceQuota:
      - v1
    Role:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RoleBinding:
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    Secret:
      - v1
    SelfSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    SelfSubjectRulesReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    Service:
      - v1
    ServiceAccount:
      - v1
    StatefulSet:
      - apps/v1
    StorageClass:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    SubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    TokenReview:
      - authentication.k8s.io/v1
      - authentication.k8s.io/v1beta1
    ValidatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1beta1
      - admissionregistration.k8s.io/v1
    VolumeAttachment:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
`,

	`openshift_yaml`: `apiVersion: move2kube.konveyor.io/v1alpha1
kind: ClusterMetadata
metadata: 
  name: Openshift
spec:
  storageClasses:
    - default
  apiKindVersionMap:
    APIService:
      - apiregistration.k8s.io/v1
      - apiregistration.k8s.io/v1beta1
    Alertmanager:
      - monitoring.coreos.com/v1
    AppliedClusterResourceQuota:
      - quota.openshift.io/v1
    BinaryBuildRequestOptions:
      - build.openshift.io/v1
    Binding:
      - v1
    BrokerTemplateInstance:
      - template.openshift.io/v1
    Build:
      - build.openshift.io/v1
    BuildConfig:
      - build.openshift.io/v1
    BuildLog:
      - build.openshift.io/v1
    BuildRequest:
      - build.openshift.io/v1
    Bundle:
      - automationbroker.io/v1alpha1
    BundleBinding:
      - automationbroker.io/v1alpha1
    BundleInstance:
      - automationbroker.io/v1alpha1
    CertificateSigningRequest:
      - certificates.k8s.io/v1beta1
    ClusterNetwork:
      - network.openshift.io/v1
    ClusterResourceQuota:
      - quota.openshift.io/v1
    ClusterRole:
      - authorization.openshift.io/v1
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ClusterRoleBinding:
      - authorization.openshift.io/v1
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    ClusterServiceBroker:
      - servicecatalog.k8s.io/v1beta1
    ClusterServiceClass:
      - servicecatalog.k8s.io/v1beta1
    ClusterServicePlan:
      - servicecatalog.k8s.io/v1beta1
    ComponentStatus:
      - v1
    ConfigMap:
      - v1
    ControllerRevision:
      - apps/v1
      - apps/v1beta1
      - apps/v1beta2
    CronJob:
      - batch/v1beta1
    CustomResourceDefinition:
      - apiextensions.k8s.io/v1beta1
    DaemonSet:
      - apps/v1
      - apps/v1beta2
      - extensions/v1beta1
    Deployment:
      - apps/v1
      - apps/v1beta1
      - apps/v1beta2
      - extensions/v1beta1
    DeploymentConfig:
      - apps.openshift.io/v1
    DeploymentConfigRollback:
      - apps.openshift.io/v1
    DeploymentLog:
      - apps.openshift.io/v1
    DeploymentRequest:
      - apps.openshift.io/v1
    DeploymentRollback:
      - apps/v1beta1
      - extensions/v1beta1
    EgressNetworkPolicy:
      - network.openshift.io/v1
    Endpoints:
      - v1
    Event:
      - events.k8s.io/v1beta1
      - v1
    Eviction:
      - v1
    Group:
      - user.openshift.io/v1
    HorizontalPodAutoscaler:
      - autoscaling/v1
      - autoscaling/v2beta1
    HostSubnet:
      - network.openshift.io/v1
    Identity:
      - user.openshift.io/v1
    Image:
      - image.openshift.io/v1
    ImageSignature:
      - image.openshift.io/v1
    ImageStream:
      - image.openshift.io/v1
    ImageStreamImage:
      - image.openshift.io/v1
    ImageStreamImport:
      - image.openshift.io/v1
    ImageStreamLayers:
      - image.openshift.io/v1
    ImageStreamMapping:
      - image.openshift.io/v1
    ImageStreamTag:
      - image.openshift.io/v1
    Ingress:
      - extensions/v1beta1
    Job:
      - batch/v1
    LimitRange:
      - v1
    LocalResourceAccessReview:
      - authorization.openshift.io/v1
    LocalSubjectAccessReview:
      - authorization.openshift.io/v1
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    MutatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1beta1
    Namespace:
      - v1
    NetNamespace:
      - network.openshift.io/v1
    NetworkPolicy:
      - networking.k8s.io/v1
      - extensions/v1beta1
    Node:
      - v1
    OAuthAccessToken:
      - oauth.openshift.io/v1
    OAuthAuthorizeToken:
      - oauth.openshift.io/v1
    OAuthClient:
      - oauth.openshift.io/v1
    OAuthClientAuthorization:
      - oauth.openshift.io/v1
    PersistentVolume:
      - v1
    PersistentVolumeClaim:
      - v1
    Pod:
      - v1
    PodDisruptionBudget:
      - policy/v1beta1
    PodSecurityPolicy:
      - extensions/v1beta1
      - policy/v1beta1
    PodSecurityPolicyReview:
      - security.openshift.io/v1
    PodSecurityPolicySelfSubjectReview:
      - security.openshift.io/v1
    PodSecurityPolicySubjectReview:
      - security.openshift.io/v1
    PodTemplate:
      - v1
    PriorityClass:
      - scheduling.k8s.io/v1beta1
    Project:
      - project.openshift.io/v1
    ProjectRequest:
      - project.openshift.io/v1
    Prometheus:
      - monitoring.coreos.com/v1
    PrometheusRule:
      - monitoring.coreos.com/v1
    RangeAllocation:
      - security.openshift.io/v1
    ReplicaSet:
      - apps/v1
      - apps/v1beta2
      - extensions/v1beta1
    ReplicationController:
      - v1
    ReplicationControllerDummy:
      - extensions/v1beta1
    ResourceAccessReview:
      - authorization.openshift.io/v1
    ResourceQuota:
      - v1
    Role:
      - authorization.openshift.io/v1
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RoleBinding:
      - authorization.openshift.io/v1
      - rbac.authorization.k8s.io/v1
      - rbac.authorization.k8s.io/v1beta1
    RoleBindingRestriction:
      - authorization.openshift.io/v1
    Route:
      - route.openshift.io/v1
    Scale:
      - apps.openshift.io/v1
      - apps/v1
      - apps/v1beta1
      - apps/v1beta2
      - extensions/v1beta1
      - v1
    Secret:
      - v1
    SecretList:
      - image.openshift.io/v1
    SecurityContextConstraints:
      - security.openshift.io/v1
      - v1
    SelfSubjectAccessReview:
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    SelfSubjectRulesReview:
      - authorization.openshift.io/v1
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    Service:
      - v1
    ServiceAccount:
      - v1
    ServiceBinding:
      - servicecatalog.k8s.io/v1beta1
    ServiceBroker:
      - servicecatalog.k8s.io/v1beta1
    ServiceClass:
      - servicecatalog.k8s.io/v1beta1
    ServiceInstance:
      - servicecatalog.k8s.io/v1beta1
    ServiceMonitor:
      - monitoring.coreos.com/v1
    ServicePlan:
      - servicecatalog.k8s.io/v1beta1
    StatefulSet:
      - apps/v1
      - apps/v1beta1
      - apps/v1beta2
    StorageClass:
      - storage.k8s.io/v1
      - storage.k8s.io/v1beta1
    SubjectAccessReview:
      - authorization.openshift.io/v1
      - authorization.k8s.io/v1
      - authorization.k8s.io/v1beta1
    SubjectRulesReview:
      - authorization.openshift.io/v1
    Template:
      - template.openshift.io/v1
    TemplateInstance:
      - template.openshift.io/v1
    TokenReview:
      - authentication.k8s.io/v1
      - authentication.k8s.io/v1beta1
    User:
      - user.openshift.io/v1
    UserIdentityMapping:
      - user.openshift.io/v1
    ValidatingWebhookConfiguration:
      - admissionregistration.k8s.io/v1beta1
    VolumeAttachment:
      - storage.k8s.io/v1beta1
`,
}
