/*
Copyright 2022 The KodeRover Authors.

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

package job

import (
	"fmt"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/config"
	commonmodels "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models"
	commonrepo "github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/mongodb"
	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/util"
	"github.com/koderover/zadig/v2/pkg/setting"
	e "github.com/koderover/zadig/v2/pkg/tool/errors"
)

type K8sPacthJob struct {
	job      *commonmodels.Job
	workflow *commonmodels.WorkflowV4
	spec     *commonmodels.K8sPatchJobSpec
}

func (j *K8sPacthJob) Instantiate() error {
	j.spec = &commonmodels.K8sPatchJobSpec{}
	if err := commonmodels.IToiYaml(j.job.Spec, j.spec); err != nil {
		return err
	}
	j.job.Spec = j.spec
	return nil
}

func (j *K8sPacthJob) SetPreset() error {
	j.spec = &commonmodels.K8sPatchJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return err
	}
	clusters, err := commonrepo.NewK8SClusterColl().List(&commonrepo.ClusterListOpts{})
	if err != nil {
		return fmt.Errorf("failed to list clusters, error: %s", err)
	}
	options := make([]*commonmodels.ClusterBrief, 0)
	for _, cluster := range clusters {
		options = append(options, &commonmodels.ClusterBrief{
			ClusterID:   cluster.ID.Hex(),
			ClusterName: cluster.Name,
		})

		strategies := make([]*commonmodels.ClusterStrategyBrief, 0)

		if cluster.AdvancedConfig != nil {
			for _, strategy := range cluster.AdvancedConfig.ScheduleStrategy {
				strategies = append(strategies, &commonmodels.ClusterStrategyBrief{
					StrategyID:   strategy.StrategyID,
					StrategyName: strategy.StrategyName,
				})
			}
		}
	}

	j.spec.ClusterOptions = options
	j.job.Spec = j.spec
	return nil
}

func (j *K8sPacthJob) MergeArgs(args *commonmodels.Job) error {
	if j.job.Name == args.Name && j.job.JobType == args.JobType {
		j.spec = &commonmodels.K8sPatchJobSpec{}
		if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
			return err
		}
		j.job.Spec = j.spec
		argsSpec := &commonmodels.K8sPatchJobSpec{}
		if err := commonmodels.IToi(args.Spec, argsSpec); err != nil {
			return err
		}
		j.spec.PatchItems = argsSpec.PatchItems
		j.job.Spec = j.spec
	}
	return nil
}

func (j *K8sPacthJob) ToJobs(taskID int64) ([]*commonmodels.JobTask, error) {
	// logger := log.SugaredLogger()
	resp := []*commonmodels.JobTask{}
	j.spec = &commonmodels.K8sPatchJobSpec{}
	if err := commonmodels.IToi(j.job.Spec, j.spec); err != nil {
		return resp, err
	}
	jobTask := &commonmodels.JobTask{
		Name: j.job.Name,
		Key:  j.job.Name,
		JobInfo: map[string]string{
			JobNameKey: j.job.Name,
		},
		JobType: string(config.JobK8sPatch),
		Spec:    patchJobToTaskJob(j.spec),
	}
	resp = append(resp, jobTask)
	j.job.Spec = j.spec
	return resp, nil
}

func (j *K8sPacthJob) LintJob() error {
	if err := util.CheckZadigProfessionalLicense(); err != nil {
		return e.ErrLicenseInvalid.AddDesc("")
	}

	return nil
}

func patchJobToTaskJob(job *commonmodels.K8sPatchJobSpec) *commonmodels.JobTasK8sPatchSpec {
	resp := &commonmodels.JobTasK8sPatchSpec{
		ClusterID: job.ClusterID,
		Namespace: job.Namespace,
	}
	for _, patch := range job.PatchItems {
		patchTaskItem := &commonmodels.PatchTaskItem{
			ResourceName:    patch.ResourceName,
			ResourceKind:    patch.ResourceKind,
			ResourceGroup:   patch.ResourceGroup,
			ResourceVersion: patch.ResourceVersion,
			PatchContent:    renderString(patch.PatchContent, setting.RenderValueTemplate, patch.Params),
			PatchStrategy:   patch.PatchStrategy,
			Params:          patch.Params,
		}
		resp.PatchItems = append(resp.PatchItems, patchTaskItem)
	}
	return resp
}
