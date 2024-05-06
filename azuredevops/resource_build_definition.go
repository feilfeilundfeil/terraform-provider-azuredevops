package azuredevops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/builddefinitionhelper"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/config"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/converter"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/tfhelper"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/validate"
)

// RepoType the type of the repository
type RepoType string

type repoTypeValuesType struct {
	GitHub    RepoType
	TfsGit    RepoType
	Bitbucket RepoType
}

// RepoTypeValues enum of the type of the repository
var RepoTypeValues = repoTypeValuesType{
	GitHub:    "GitHub",
	TfsGit:    "TfsGit",
	Bitbucket: "Bitbucket",
}

func resourceBuildDefinition() *schema.Resource {
	filterSchema := map[string]*schema.Schema{
		"include": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.NoZeroValues,
			},
		},
		"exclude": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.NoZeroValues,
			},
		},
	}

	branchFilter := &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		MinItems: 1,
		Elem: &schema.Resource{
			Schema: filterSchema,
		},
	}

	pathFilter := &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: filterSchema,
		},
	}

	return &schema.Resource{
		Create: resourceBuildDefinitionCreate,
		Read:   resourceBuildDefinitionRead,
		Update: resourceBuildDefinitionUpdate,
		Delete: resourceBuildDefinitionDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, buildDefinitionID, err := ParseImportedProjectIDAndID(meta.(*config.AggregatedClient), d.Id())
				if err != nil {
					return nil, fmt.Errorf("error parsing the build definition ID from the Terraform resource data: %v", err)
				}
				d.Set("project_id", projectID)
				d.SetId(fmt.Sprintf("%d", buildDefinitionID))

				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			// FFUF Start - Shared libs
			"allow_scripts_auth_access_option": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			// FFUF End
			"revision": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"path": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      `\`,
				ValidateFunc: validate.Path,
			},
			"variable_groups": {
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type:         schema.TypeInt,
					ValidateFunc: validation.IntAtLeast(1),
				},
			},
			"agent_pool_name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Azure Pipelines",
			},
			"agent_specification": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ubuntu-20.04",
				ValidateFunc: validation.StringInSlice([]string{
					"macOS-11",
					"macOS-12",
					"macos-13",
					"macOS-14",
					"macos-latest",
					"ubuntu-20.04",
					"ubuntu-22.04",
					"ubuntu-latest",
					"windows-2019",
					"windows-2022",
					"windows-latest",
				}, false),
			},
			"repository": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"yml_path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"repo_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"repo_type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"GitHub", "TfsGit", "Bitbucket"}, false),
						},
						"branch_name": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "master",
						},
						"service_connection_id": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"report_build_status": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
					},
				},
			},
			"build_completion_trigger": {
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"use_yaml": {
							Type:          schema.TypeBool,
							Optional:      true,
							Default:       false,
							ConflictsWith: []string{"build_completion_trigger.0.override"},
						},
						"override": {
							Type:     schema.TypeSet,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"project_id": {
										Type:     schema.TypeString,
										Required: true,
									},
									"build_id": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"branch_filter": branchFilter,
								},
							},
						},
					},
				},
			},
			"ci_trigger": {
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"use_yaml": {
							Type:          schema.TypeBool,
							Optional:      true,
							Default:       false,
							ConflictsWith: []string{"ci_trigger.0.override"},
						},
						"override": {
							Type:     schema.TypeSet,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"batch": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  true,
									},
									"branch_filter": branchFilter,
									"max_concurrent_builds_per_branch": {
										Type:     schema.TypeInt,
										Optional: true,
										Default:  1,
									},
									"path_filter": pathFilter,
									"polling_interval": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"polling_job_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"pull_request_trigger": {
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"use_yaml": {
							Type:          schema.TypeBool,
							Optional:      true,
							Default:       false,
							ConflictsWith: []string{"pull_request_trigger.0.override"},
						},
						"override": {
							Type:     schema.TypeSet,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"auto_cancel": {
										Type:     schema.TypeBool,
										Optional: true,
									},
									"branch_filter": branchFilter,
									"path_filter":   pathFilter,
								},
							},
						},
						"forks": {
							Type:     schema.TypeSet,
							Required: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Required: true,
									},
									"share_secrets": {
										Type:     schema.TypeBool,
										Required: true,
									},
								},
							},
						},
						"comment_required": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"All", "NonTeamMembers"}, false),
						},
					},
				},
			},
			"step": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"task": {
							Type:     schema.TypeSet,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"identifier": {
										Type:     schema.TypeString,
										Required: true,
									},
									"definition_type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"version": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"ref_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"inputs": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"tags": TagsSchema,
			"schedules": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"branch_filter": branchFilter,
						"days_to_build": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}, false),
							},
						},
						"schedule_only_with_changes": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"start_hours": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validation.IntBetween(0, 23),
						},
						"start_minutes": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      0,
							ValidateFunc: validation.IntBetween(0, 59),
						},
						"time_zone": {
							Optional:     true,
							Type:         schema.TypeString,
							ValidateFunc: validation.StringInSlice(builddefinitionhelper.TimeZones, false),
							Default:      "(UTC) Coordinated Universal Time",
						},
						"schedule_job_id": {
							Computed: true,
							Type:     schema.TypeString,
						},
					},
				},
			},
		},
	}
}

func resourceBuildDefinitionCreate(d *schema.ResourceData, m interface{}) error {
	clients := m.(*config.AggregatedClient)
	buildDefinition, projectID, err := expandBuildDefinition(d)
	if err != nil {
		return fmt.Errorf("error creating resource Build Definition: %+v", err)
	}

	createdBuildDefinition, err := createBuildDefinition(clients, buildDefinition, projectID)
	if err != nil {
		return fmt.Errorf("error creating resource Build Definition: %+v", err)
	}

	body, _ := json.Marshal(*createdBuildDefinition)
	s := string(body[:])
	println("BUILD DEFINITION CREATE RESPONSE")
	println(s)

	flattenBuildDefinition(d, createdBuildDefinition, projectID)
	return resourceBuildDefinitionRead(d, m)
}

func flattenBuildDefinition(d *schema.ResourceData, buildDefinition *build.BuildDefinition, projectID string) {
	d.SetId(strconv.Itoa(*buildDefinition.Id))

	d.Set("project_id", projectID)
	d.Set("name", *buildDefinition.Name)
	d.Set("path", *buildDefinition.Path)
	d.Set("repository", flattenRepository(buildDefinition))
	d.Set("agent_pool_name", *buildDefinition.Queue.Pool.Name)
	d.Set("path", *buildDefinition.Path)

	d.Set("variable_groups", flattenVariableGroups(buildDefinition))
	d.Set("step", flattenStep(buildDefinition.Process.(map[string]interface{})))

	if buildDefinition.Triggers != nil {
		yamlCiTrigger := hasSettingsSourceType(buildDefinition.Triggers, build.DefinitionTriggerTypeValues.ContinuousIntegration, 2)
		d.Set("ci_trigger", flattenBuildDefinitionTrigger(buildDefinition.Triggers, yamlCiTrigger, build.DefinitionTriggerTypeValues.ContinuousIntegration))

		yamlPrTrigger := hasSettingsSourceType(buildDefinition.Triggers, build.DefinitionTriggerTypeValues.PullRequest, 2)
		d.Set("pull_request_trigger", flattenBuildDefinitionTrigger(buildDefinition.Triggers, yamlPrTrigger, build.DefinitionTriggerTypeValues.PullRequest))

		yamlBuildTrigger := hasSettingsSourceType(buildDefinition.Triggers, build.DefinitionTriggerTypeValues.BuildCompletion, 2)
		d.Set("build_completion_trigger", flattenBuildDefinitionTrigger(buildDefinition.Triggers, yamlBuildTrigger, build.DefinitionTriggerTypeValues.BuildCompletion))

		yamlScheduleTrigger := hasSettingsSourceType(buildDefinition.Triggers, build.DefinitionTriggerTypeValues.Schedule, 2)
		d.Set("schedules", flattenBuildDefinitionTrigger(buildDefinition.Triggers, yamlScheduleTrigger, build.DefinitionTriggerTypeValues.Schedule))
	}

	revision := 0
	if buildDefinition.Revision != nil {
		revision = *buildDefinition.Revision
	}

	d.Set("revision", revision)
}

func flattenStep(process map[string]interface{}) *schema.Set {

	phase := process["phases"].([]interface{})[0].(map[string]interface{})
	steps := phase["steps"].([]interface{})
	step := steps[0].(map[string]interface{})

	inputs := step["inputs"].(map[string]interface{})

	task := step["task"].(map[string]interface{})
	/*
		taskId := uuid.MustParse(m["identifier"].(string))
		return &build.TaskDefinitionReference{
			DefinitionType: converter.String(m["definition_type"].(string)),
			Id:             &taskId,
			VersionSpec:    converter.String(m["version"].(string)),
		}
	*/

	taskMap := map[string]interface{}{
		"definition_type": task["definitionType"].(string),
		"version":         task["versionSpec"].(string),
		"identifier":      task["id"].(string),
	}
	println(inputs)
	println(taskMap)

	set := schema.NewSet(schema.HashString, nil)
	for range steps {
	}
	return set
	/*map[string]interface{} {
		"inputs":              inputs,
		"task": taskMap,
	}*/
}

func flattenProcess(process interface{}) interface{} {
	val, _ := json.Marshal(process)
	valStr := string(val[:])
	return valStr
}

func flattenVariableGroups(buildDefinition *build.BuildDefinition) []int {
	if buildDefinition.VariableGroups == nil {
		return nil
	}

	variableGroups := make([]int, len(*buildDefinition.VariableGroups))

	for i, variableGroup := range *buildDefinition.VariableGroups {
		variableGroups[i] = *variableGroup.Id
	}

	return variableGroups
}

func createBuildDefinition(clients *config.AggregatedClient, buildDefinition *build.BuildDefinition, project string) (*build.BuildDefinition, error) {
	createdBuild, err := clients.BuildClient.CreateDefinition(clients.Ctx, build.CreateDefinitionArgs{
		Definition: buildDefinition,
		Project:    &project,
	})

	return createdBuild, err
}

func resourceBuildDefinitionRead(d *schema.ResourceData, m interface{}) error {
	clients := m.(*config.AggregatedClient)
	projectID, buildDefinitionID, err := tfhelper.ParseProjectIDAndResourceID(d)

	if err != nil {
		return err
	}

	buildDefinition, err := clients.BuildClient.GetDefinition(clients.Ctx, build.GetDefinitionArgs{
		Project:      &projectID,
		DefinitionId: &buildDefinitionID,
	})

	if err != nil {
		if utils.ResponseWasNotFound(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	flattenBuildDefinition(d, buildDefinition, projectID)
	return nil
}

func resourceBuildDefinitionDelete(d *schema.ResourceData, m interface{}) error {
	if d.Id() == "" {
		return nil
	}

	clients := m.(*config.AggregatedClient)
	projectID, buildDefinitionID, err := tfhelper.ParseProjectIDAndResourceID(d)
	if err != nil {
		return err
	}

	err = clients.BuildClient.DeleteDefinition(m.(*config.AggregatedClient).Ctx, build.DeleteDefinitionArgs{
		Project:      &projectID,
		DefinitionId: &buildDefinitionID,
	})

	return err
}

func resourceBuildDefinitionUpdate(d *schema.ResourceData, m interface{}) error {
	clients := m.(*config.AggregatedClient)
	buildDefinition, projectID, err := expandBuildDefinition(d)
	if err != nil {
		return err
	}

	updatedBuildDefinition, err := clients.BuildClient.UpdateDefinition(m.(*config.AggregatedClient).Ctx, build.UpdateDefinitionArgs{
		Definition:   buildDefinition,
		Project:      &projectID,
		DefinitionId: buildDefinition.Id,
	})

	if err != nil {
		return err
	}

	flattenBuildDefinition(d, updatedBuildDefinition, projectID)
	return nil
}

func flattenRepository(buildDefinition *build.BuildDefinition) interface{} {
	yamlFilePath := ""

	// The process member can be of many types -- the only typing information
	// available from the compiler is `interface{}` so we can probe for known
	// implementations
	if processMap, ok := buildDefinition.Process.(map[string]interface{}); ok {
		if processMap["yamlFilename"] != nil {
			yamlFilePath = processMap["yamlFilename"].(string)
		}
	}

	if yamlProcess, ok := buildDefinition.Process.(*build.YamlProcess); ok {
		yamlFilePath = *yamlProcess.YamlFilename
	}
	reportBuildStatus, _ := strconv.ParseBool((*buildDefinition.Repository.Properties)["reportBuildStatus"])

	if yamlFilePath != "" {
		return []map[string]interface{}{{
			"yml_path":              yamlFilePath,
			"repo_name":             *buildDefinition.Repository.Name,
			"repo_type":             *buildDefinition.Repository.Type,
			"branch_name":           *buildDefinition.Repository.DefaultBranch,
			"service_connection_id": (*buildDefinition.Repository.Properties)["connectedServiceId"],
			"report_build_status":   reportBuildStatus,
		}}
	} else {
		return []map[string]interface{}{{
			"repo_name":             *buildDefinition.Repository.Name,
			"repo_type":             *buildDefinition.Repository.Type,
			"branch_name":           *buildDefinition.Repository.DefaultBranch,
			"service_connection_id": (*buildDefinition.Repository.Properties)["connectedServiceId"],
			"report_build_status":   reportBuildStatus,
		}}
	}
}

func flattenBuildDefinitionBranchOrPathFilter(m []interface{}) []interface{} {
	var include []string
	var exclude []string

	for _, v := range m {
		if v2, ok := v.(string); ok {
			if strings.HasPrefix(v2, "-") {
				exclude = append(exclude, strings.TrimPrefix(v2, "-"))
			} else if strings.HasPrefix(v2, "+") {
				include = append(include, strings.TrimPrefix(v2, "+"))
			}
		}
	}
	sort.Strings(include)
	sort.Strings(exclude)
	return []interface{}{
		map[string]interface{}{
			"include": include,
			"exclude": exclude,
		},
	}
}

func flattenBuildDefinitionContinuousIntegrationTrigger(m interface{}, isYaml bool) interface{} {
	if ms, ok := m.(map[string]interface{}); ok {
		f := map[string]interface{}{
			"use_yaml": isYaml,
		}
		if !isYaml {
			f["override"] = []map[string]interface{}{{
				"batch":                            ms["batchChanges"],
				"branch_filter":                    flattenBuildDefinitionBranchOrPathFilter(ms["branchFilters"].([]interface{})),
				"max_concurrent_builds_per_branch": ms["maxConcurrentBuildsPerBranch"],
				"polling_interval":                 ms["pollingInterval"],
				"polling_job_id":                   ms["pollingJobId"],
				"path_filter":                      flattenBuildDefinitionBranchOrPathFilter(ms["pathFilters"].([]interface{})),
			}}
		}
		return f
	}
	return nil
}

func flattenBuildDefinitionPullRequestTrigger(m interface{}, isYaml bool) interface{} {
	if ms, ok := m.(map[string]interface{}); ok {
		forks := ms["forks"].(map[string]interface{})
		isCommentRequired := ms["isCommentRequiredForPullRequest"].(bool)
		isCommentRequiredNonTeam := ms["requireCommentsForNonTeamMembersOnly"].(bool)

		var commentRequired string
		if isCommentRequired {
			commentRequired = "All"
		}
		if isCommentRequired && isCommentRequiredNonTeam {
			commentRequired = "NonTeamMembers"
		}

		f := map[string]interface{}{
			"use_yaml":         isYaml,
			"comment_required": commentRequired,
			"forks": []map[string]interface{}{{
				"enabled":       forks["enabled"],
				"share_secrets": forks["allowSecrets"],
			}},
		}
		if !isYaml {
			f["override"] = []map[string]interface{}{{
				"auto_cancel":   ms["autoCancel"],
				"branch_filter": flattenBuildDefinitionBranchOrPathFilter(ms["branchFilters"].([]interface{})),
				"path_filter":   flattenBuildDefinitionBranchOrPathFilter(ms["pathFilters"].([]interface{})),
			}}
		}
		return f
	}
	return nil
}

func flattenBuildDefinitionBuildCompletionTrigger(m interface{}, isYaml bool) interface{} {
	if ms, ok := m.(map[string]interface{}); ok {

		f := map[string]interface{}{
			"use_yaml": isYaml,
		}
		if !isYaml {

			definition := ms["definition"].(map[string]interface{})
			project := definition["project"].(map[string]interface{})

			f["override"] = []map[string]interface{}{{
				"project_id":    project["id"],
				"build_id":      definition["id"],
				"branch_filter": flattenBuildDefinitionBranchOrPathFilter(ms["branchFilters"].([]interface{})),
			}}
		}
		return f
	}
	return nil
}

func flattenBuildDefinitionTrigger(m interface{}, isYaml bool, t build.DefinitionTriggerType) interface{} {
	if ms, ok := m.(map[string]interface{}); ok {
		if ms["triggerType"].(string) != string(t) {
			return nil
		}
		switch t {
		case build.DefinitionTriggerTypeValues.ContinuousIntegration:
			return flattenBuildDefinitionContinuousIntegrationTrigger(ms, isYaml)
		case build.DefinitionTriggerTypeValues.PullRequest:
			return flattenBuildDefinitionPullRequestTrigger(ms, isYaml)
		case build.DefinitionTriggerTypeValues.BuildCompletion:
			return flattenBuildDefinitionBuildCompletionTrigger(ms, isYaml)
		}
	}
	return nil
}

func hasSettingsSourceType(m *[]interface{}, t build.DefinitionTriggerType, sst int) bool {
	hasSetting := false
	for _, d := range *m {
		if ms, ok := d.(map[string]interface{}); ok {
			if ms["triggerType"].(string) == string(t) {
				if val, ok := ms["settingsSourceType"]; ok {
					hasSetting = int(val.(float64)) == sst
				}
			}
		}
	}
	return hasSetting
}

func expandBuildDefinitionBranchOrPathFilter(d map[string]interface{}) []interface{} {
	include := expandStringSet(d["include"].(*schema.Set))
	exclude := expandStringSet(d["exclude"].(*schema.Set))
	sort.Strings(include)
	sort.Strings(exclude)
	m := make([]interface{}, len(include)+len(exclude))
	i := 0
	for _, v := range include {
		m[i] = "+" + v
		i++
	}
	for _, v := range exclude {
		m[i] = "-" + v
		i++
	}
	return m
}
func expandBuildDefinitionBranchOrPathFilterList(d []interface{}) [][]interface{} {
	vs := make([][]interface{}, 0, len(d))
	for _, v := range d {
		if val, ok := v.(map[string]interface{}); ok {
			vs = append(vs, expandBuildDefinitionBranchOrPathFilter(val))
		}
	}
	return vs
}
func expandBuildDefinitionBranchOrPathFilterSet(configured *schema.Set) []interface{} {
	d2 := expandBuildDefinitionBranchOrPathFilterList(configured.List())
	if len(d2) != 1 {
		return nil
	}
	return d2[0]
}

func expandBuildDefinitionFork(d map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"allowSecrets": d["share_secrets"].(bool),
		"enabled":      d["enabled"].(bool),
	}
}
func expandBuildDefinitionForkList(d []interface{}) []map[string]interface{} {
	vs := make([]map[string]interface{}, 0, len(d))
	for _, v := range d {
		if val, ok := v.(map[string]interface{}); ok {
			vs = append(vs, expandBuildDefinitionFork(val))
		}
	}
	return vs
}
func expandBuildDefinitionForkSet(configured *schema.Set) map[string]interface{} {
	d2 := expandBuildDefinitionForkList(configured.List())
	if len(d2) != 1 {
		return nil
	}
	return d2[0]
}

func expandBuildDefinitionManualPullRequestTrigger(d map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"branchFilters": expandBuildDefinitionBranchOrPathFilterSet(d["branch_filter"].(*schema.Set)),
		"pathFilters":   expandBuildDefinitionBranchOrPathFilterSet(d["path_filter"].(*schema.Set)),
		"autoCancel":    d["auto_cancel"].(bool),
	}
}
func expandBuildDefinitionManualPullRequestTriggerList(d []interface{}) []map[string]interface{} {
	vs := make([]map[string]interface{}, 0, len(d))
	for _, v := range d {
		if val, ok := v.(map[string]interface{}); ok {
			vs = append(vs, expandBuildDefinitionManualPullRequestTrigger(val))
		}
	}
	return vs
}
func expandBuildDefinitionManualPullRequestTriggerSet(configured *schema.Set) map[string]interface{} {
	d2 := expandBuildDefinitionManualPullRequestTriggerList(configured.List())
	if len(d2) != 1 {
		return nil
	}
	return d2[0]
}

func expandBuildDefinitionManualContinuousIntegrationTrigger(d map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"batchChanges":                 d["batch"].(bool),
		"branchFilters":                expandBuildDefinitionBranchOrPathFilterSet(d["branch_filter"].(*schema.Set)),
		"maxConcurrentBuildsPerBranch": d["max_concurrent_builds_per_branch"].(int),
		"pathFilters":                  expandBuildDefinitionBranchOrPathFilterSet(d["path_filter"].(*schema.Set)),
		"triggerType":                  string(build.DefinitionTriggerTypeValues.ContinuousIntegration),
		"pollingInterval":              d["polling_interval"].(int),
	}
}
func expandBuildDefinitionManualContinuousIntegrationTriggerList(d []interface{}) []map[string]interface{} {
	vs := make([]map[string]interface{}, 0, len(d))
	for _, v := range d {
		if val, ok := v.(map[string]interface{}); ok {
			vs = append(vs, expandBuildDefinitionManualContinuousIntegrationTrigger(val))
		}
	}
	return vs
}
func expandBuildDefinitionManualContinuousIntegrationTriggerSet(configured *schema.Set) map[string]interface{} {
	d2 := expandBuildDefinitionManualContinuousIntegrationTriggerList(configured.List())
	if len(d2) != 1 {
		return nil
	}
	return d2[0]
}

func expandBuildDefinitionManualBuildCompletionTriggerSet(configured *schema.Set) map[string]interface{} {
	d2 := expandBuildDefinitionManualBuildCompletionTriggerList(configured.List())
	if len(d2) != 1 {
		return nil
	}
	return d2[0]
}

func expandBuildDefinitionManualBuildCompletionTriggerList(d []interface{}) []map[string]interface{} {
	vs := make([]map[string]interface{}, 0, len(d))
	for _, v := range d {
		if val, ok := v.(map[string]interface{}); ok {
			vs = append(vs, expandBuildDefinitionManualBuildCompletionTrigger(val))
		}
	}
	return vs
}

func expandBuildDefinitionManualBuildCompletionTrigger(d map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"branchFilters": expandBuildDefinitionBranchOrPathFilterSet(d["branch_filter"].(*schema.Set)),
		"definition": map[string]interface{}{
			"id": d["build_id"].(int),
			"project": map[string]interface{}{
				"id": d["project_id"].(string),
			},
		},
		"requiresSuccessfulBuild": true,
	}
}

func expandBuildDefinitionTrigger(d map[string]interface{}, t build.DefinitionTriggerType) interface{} {
	switch t {
	case build.DefinitionTriggerTypeValues.ContinuousIntegration:
		isYaml := d["use_yaml"].(bool)
		if isYaml {
			return map[string]interface{}{
				"batchChanges":                 false,
				"branchFilters":                []interface{}{},
				"maxConcurrentBuildsPerBranch": 1,
				"pathFilters":                  []interface{}{},
				"triggerType":                  string(t),
				"settingsSourceType":           float64(2),
			}
		}
		return expandBuildDefinitionManualContinuousIntegrationTriggerSet(d["override"].(*schema.Set))
	case build.DefinitionTriggerTypeValues.PullRequest:
		isYaml := d["use_yaml"].(bool)
		commentRequired := d["comment_required"].(string)
		vs := map[string]interface{}{
			"forks":                                expandBuildDefinitionForkSet(d["forks"].(*schema.Set)),
			"isCommentRequiredForPullRequest":      len(commentRequired) > 0,
			"requireCommentsForNonTeamMembersOnly": commentRequired == "NonTeamMembers",
			"triggerType":                          string(t),
		}
		if isYaml {
			vs["branchFilters"] = []interface{}{
				// TODO : how to get the source branch into here?
				"+develop",
			}
			vs["pathFilters"] = []interface{}{}
			vs["settingsSourceType"] = float64(2)
		} else {
			override := expandBuildDefinitionManualPullRequestTriggerSet(d["override"].(*schema.Set))
			vs["branchFilters"] = override["branchFilters"]
			vs["pathFilters"] = override["pathFilters"]
			vs["autoCancel"] = override["autoCancel"]
		}
		return vs
	case build.DefinitionTriggerTypeValues.BuildCompletion:
		isYaml := d["use_yaml"].(bool)
		if isYaml {
			return map[string]interface{}{
				"batchChanges":                 false,
				"branchFilters":                []interface{}{},
				"maxConcurrentBuildsPerBranch": 1,
				"pathFilters":                  []interface{}{},
				"triggerType":                  string(t),
				"settingsSourceType":           float64(2),
			}
		}
		vs := expandBuildDefinitionManualBuildCompletionTriggerSet(d["override"].(*schema.Set))
		vs["triggerType"] = string(t)
		return vs
	case build.DefinitionTriggerTypeValues.Schedule:
		scheduleConfig := map[string]interface{}{
			"branchFilters":           expandBuildDefinitionBranchOrPathFilterSet(d["branch_filter"].(*schema.Set)),
			"scheduleOnlyWithChanges": d["schedule_only_with_changes"],
			"startHours":              d["start_hours"],
			"startMinutes":            d["start_minutes"],
			"timeZoneId":              builddefinitionhelper.TimeZoneToID[d["time_zone"].(string)],
			"scheduleJobId":           nil,
		}
		scheduleConfig["daysToBuild"] = builddefinitionhelper.DateToDays(d["days_to_build"].([]interface{}))
		return scheduleConfig
	}
	return nil
}
func expandBuildDefinitionTriggerList(d []interface{}, t build.DefinitionTriggerType) []interface{} {
	vs := make([]interface{}, 0, len(d))
	for _, v := range d {
		val, ok := v.(map[string]interface{})
		if ok {
			vs = append(vs, expandBuildDefinitionTrigger(val, t))
		}
	}
	return vs
}
func expandBuildDefinitionTriggerSet(configured *schema.Set, t build.DefinitionTriggerType) []interface{} {
	return expandBuildDefinitionTriggerList(configured.List(), t)
}

func expandBuildDefinition(d *schema.ResourceData) (*build.BuildDefinition, string, error) {
	projectID := d.Get("project_id").(string)
	repositories := d.Get("repository").(*schema.Set).List()

	variableGroupsInterface := d.Get("variable_groups").(*schema.Set).List()
	variableGroups := make([]build.VariableGroup, len(variableGroupsInterface))

	for i, variableGroup := range variableGroupsInterface {
		variableGroups[i] = *buildVariableGroup(variableGroup.(int))
	}

	stepsInterface := d.Get("step").(*schema.Set).List()

	var steps []build.BuildDefinitionStep
	for _, step := range stepsInterface {
		stepBuilt := buildStep(step.(map[string]interface{}))
		if stepBuilt != nil {
			steps = append(steps, *stepBuilt)
			//steps = append([]build.BuildDefinitionStep{*stepBuilt}, steps...)
		}
	}

	// Note: If configured, this will be of length 1 based on the schema definition above.
	if len(repositories) != 1 {
		return nil, "", fmt.Errorf("Unexpectedly did not find repository metadata in the resource data")
	}

	repository := repositories[0].(map[string]interface{})

	repoName := repository["repo_name"].(string)
	repoType := repository["repo_type"].(string)
	repoURL := ""
	if strings.EqualFold(repoType, "github") {
		repoURL = fmt.Sprintf("https://github.com/%s.git", repoName)
	}
	if strings.EqualFold(repoType, "bitbucket") {
		repoURL = fmt.Sprintf("https://bitbucket.org/%s.git", repoName)
	}

	ciTriggers := expandBuildDefinitionTriggerSet(
		d.Get("ci_trigger").(*schema.Set),
		build.DefinitionTriggerTypeValues.ContinuousIntegration,
	)
	pullRequestTriggers := expandBuildDefinitionTriggerSet(
		d.Get("pull_request_trigger").(*schema.Set),
		build.DefinitionTriggerTypeValues.PullRequest,
	)
	buildCompletionTriggers := expandBuildDefinitionTriggerSet(
		d.Get("build_completion_trigger").(*schema.Set),
		build.DefinitionTriggerTypeValues.BuildCompletion,
	)

	var buildTriggers []interface{}
	if ciTriggers != nil && len(ciTriggers) > 0 {
		buildTriggers = append(buildTriggers, ciTriggers[0])
	}
	if pullRequestTriggers != nil && len(pullRequestTriggers) > 0 {
		buildTriggers = append(buildTriggers, pullRequestTriggers[0])
	}
	if buildCompletionTriggers != nil && len(buildCompletionTriggers) > 0 {
		buildTriggers = append(buildTriggers, buildCompletionTriggers[0])
	}

	schedules := expandBuildDefinitionTriggerList(
		d.Get("schedules").([]interface{}),
		build.DefinitionTriggerTypeValues.Schedule,
	)
	if len(schedules) > 0 {
		scheduleTriggers := map[string]interface{}{
			"schedules":   schedules,
			"triggerType": string(build.DefinitionTriggerTypeValues.Schedule),
		}
		buildTriggers = append(buildTriggers, scheduleTriggers)
	}

	// Look for the ID. This may not exist if we are within the context of a "create" operation,
	// so it is OK if it is missing.
	buildDefinitionID, err := strconv.Atoi(d.Id())
	var buildDefinitionReference *int
	if err == nil {
		buildDefinitionReference = &buildDefinitionID
	} else {
		buildDefinitionReference = nil
	}

	tags := expandStringList(d.Get("tags").([]interface{}))

	agentPoolName := d.Get("agent_pool_name").(string)
	agentPoolIsHosted := agentPoolName != "Default"
	// FFUF Start - Shared Libs
	var allowScriptsAuthAccessOption = d.Get("allow_scripts_auth_access_option").(bool)
	// FFUF End
	var process interface{}

	ymlPath := converter.String(repository["yml_path"].(string))
	if ymlPath != nil {
		process = &build.YamlProcess{
			YamlFilename: ymlPath,
		}
	} else {
		var phaseName = "Agent Job 1"
		var refName = "Job_1"
		var condition = "succeeded()"

		var Type = 1
		var target = &build.PhaseTarget{
			Type: &Type,
			// FFUF Start - Shared Libs
			AllowScriptsAuthAccessOption: &allowScriptsAuthAccessOption,
			// FFUF End
		}

		var phase = &build.Phase{
			Name:                  &phaseName,
			RefName:               &refName,
			Condition:             &condition,
			Steps:                 &steps,
			Target:                target,
			JobAuthorizationScope: &build.BuildAuthorizationScopeValues.ProjectCollection,
		}
		var phases = []build.Phase{
			*phase,
		}

		if agentPoolIsHosted {
			agentSpecificationId := d.Get("agent_specification").(string)
			var processTarget = &build.DesignerProcessTarget{
				AgentSpecification: &build.AgentSpecification{
					Identifier: &agentSpecificationId,
				},
			}

			process = &build.DesignerProcess{
				Phases: &phases,
				Target: processTarget,
			}
		} else {
			process = &build.DesignerProcess{
				Phases: &phases,
			}
		}
	}

	apiUrl := ""

	if strings.EqualFold(repoType, "github") {
		apiUrl = fmt.Sprintf("https://api.github.com/repos/%s", repoName)
	}
	if strings.EqualFold(repoType, "bitbucket") {
		apiUrl = fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s", repoName)
	}

	buildDefinition := build.BuildDefinition{
		Id:                    buildDefinitionReference,
		JobAuthorizationScope: &build.BuildAuthorizationScopeValues.ProjectCollection,
		Name:                  converter.String(d.Get("name").(string)),
		Path:                  converter.String(d.Get("path").(string)),
		Revision:              converter.Int(d.Get("revision").(int)),
		Repository: &build.BuildRepository{
			Url:           &repoURL,
			Id:            &repoName,
			Name:          &repoName,
			DefaultBranch: converter.String(repository["branch_name"].(string)),
			Type:          converter.String(string(repoType)),
			Properties: &map[string]string{
				"apiUrl":             apiUrl,
				"connectedServiceId": repository["service_connection_id"].(string),
				"reportBuildStatus":  strconv.FormatBool(repository["report_build_status"].(bool)),
			},
		},
		Process: process,
		Queue: &build.AgentPoolQueue{
			Name: &agentPoolName,
			Pool: &build.TaskAgentPoolReference{
				Name:     &agentPoolName,
				IsHosted: &agentPoolIsHosted,
			},
		},
		QueueStatus:    &build.DefinitionQueueStatusValues.Enabled,
		Type:           &build.DefinitionTypeValues.Build,
		Quality:        &build.DefinitionQualityValues.Definition,
		VariableGroups: &variableGroups,
		Triggers:       &buildTriggers,
		Tags:           &tags,
	}

	body, _ := json.Marshal(&buildDefinition)
	goStyleString := string(bytes.Split(body[:], []byte{0})[0])
	log.Print(goStyleString)

	return &buildDefinition, projectID, nil
}

func buildStep(m map[string]interface{}) *build.BuildDefinitionStep {

	inputs := m["inputs"].(map[string]interface{})
	inputStrings := make(map[string]string)

	for key, value := range inputs {
		strKey := fmt.Sprintf("%v", key)
		strValue := fmt.Sprintf("%v", value)
		inputStrings[strKey] = strValue
	}

	tasks := m["task"].(*schema.Set).List()
	if len(tasks) == 1 {

		enabled := true
		return &build.BuildDefinitionStep{
			Task:    buildTask(tasks[0].(map[string]interface{})),
			Inputs:  &inputStrings,
			Enabled: &enabled,
			RefName: converter.String(m["ref_name"].(string)),
		}
	}
	return nil
}

func buildTask(m map[string]interface{}) *build.TaskDefinitionReference {
	taskId := uuid.MustParse(m["identifier"].(string))
	return &build.TaskDefinitionReference{
		DefinitionType: converter.String(m["definition_type"].(string)),
		Id:             &taskId,
		VersionSpec:    converter.String(m["version"].(string)),
	}
}

func buildVariableGroup(id int) *build.VariableGroup {
	return &build.VariableGroup{
		Id: &id,
	}
}
