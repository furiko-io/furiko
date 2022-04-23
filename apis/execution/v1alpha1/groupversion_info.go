/*
 * Copyright 2022 The Furiko Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	"github.com/furiko-io/furiko/apis/execution"
)

const (
	Version = "v1alpha1"

	KindJob           = "Job"
	KindJobConfig     = "JobConfig"
	KindJobList       = "JobList"
	KindJobConfigList = "JobConfigList"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: execution.GroupName, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// SchemeGroupVersion is group version used to register these objects.
var SchemeGroupVersion = schema.GroupVersion{
	Group:   execution.GroupName,
	Version: Version,
}

// Declare schema.GroupVersionKind for each Kind in this Group.
var (
	GVKJob           = SchemeGroupVersion.WithKind(KindJob)
	GVKJobConfig     = SchemeGroupVersion.WithKind(KindJobConfig)
	GVKJobList       = SchemeGroupVersion.WithKind(KindJobList)
	GVKJobConfigList = SchemeGroupVersion.WithKind(KindJobConfigList)
)

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
