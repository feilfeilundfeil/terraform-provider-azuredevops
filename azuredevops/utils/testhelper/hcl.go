package testhelper

import (
	"fmt"
	"strings"
)

// TestAccAzureGitRepoResource HCL describing an AzDO GIT repository resource
func TestAccAzureGitRepoResource(projectName string, gitRepoName string, initType string) string {
	azureGitRepoResource := fmt.Sprintf(`
resource "azuredevops_git_repository" "gitrepo" {
	project_id      = azuredevops_project.project.id
	name            = "%s"
	initialization {
		init_type = "%s"
	}
}`, gitRepoName, initType)

	projectResource := TestAccProjectResource(projectName)
	return fmt.Sprintf("%s\n%s", projectResource, azureGitRepoResource)
}

// TestAccAzureForkedGitRepoResource HCL describing an AzDO GIT repository resource
func TestAccAzureForkedGitRepoResource(projectName string, gitRepoName string, gitForkedRepoName string, initType string, forkedInitType string) string {
	azureGitRepoResource := fmt.Sprintf(`
resource "azuredevops_git_repository" "gitforkedrepo" {
	project_id      		= azuredevops_project.project.id
	parent_repository_id    = azuredevops_git_repository.gitrepo.id
	name            		= "%s"
	initialization {
		init_type = "%s"
	}
}`, gitForkedRepoName, forkedInitType)

	gitRepoResource := TestAccAzureGitRepoResource(projectName, gitRepoName, initType)
	return fmt.Sprintf("%s\n%s", gitRepoResource, azureGitRepoResource)
}

// TestAccGroupDataSource HCL describing an AzDO Group Data Source
func TestAccGroupDataSource(projectName string, groupName string) string {
	dataSource := fmt.Sprintf(`
data "azuredevops_group" "group" {
	project_id = azuredevops_project.project.id
	name       = "%s"
}`, groupName)

	projectResource := TestAccProjectResource(projectName)
	return fmt.Sprintf("%s\n%s", projectResource, dataSource)
}

// TestAccProjectResource HCL describing an AzDO project
func TestAccProjectResource(projectName string) string {
	if strings.EqualFold(projectName, "") {
		return ""
	}
	return fmt.Sprintf(`
resource "azuredevops_project" "project" {
	project_name       = "%s"
	description        = "%s-description"
	visibility         = "private"
	version_control    = "Git"
	work_item_template = "Agile"
}`, projectName, projectName)
}

// TestAccProjectDataSource HCL describing a data source for an AzDO project
func TestAccProjectDataSource(projectName string) string {
	return fmt.Sprintf(`
data "azuredevops_project" "project" {
	project_name = "%s"
}`, projectName)
}

// TestAccUserEntitlementResource HCL describing an AzDO UserEntitlement
func TestAccUserEntitlementResource(principalName string) string {
	return fmt.Sprintf(`
resource "azuredevops_user_entitlement" "user" {
	principal_name     = "%s"
	account_license_type = "express"
}`, principalName)
}

// TestAccServiceEndpointGitHubResource HCL describing an AzDO service endpoint
func TestAccServiceEndpointGitHubResource(projectName string, serviceEndpointName string) string {
	serviceEndpointResource := fmt.Sprintf(`
resource "azuredevops_serviceendpoint_github" "serviceendpoint" {
	project_id             = azuredevops_project.project.id
	service_endpoint_name  = "%s"
	auth_personal {
	}
}`, serviceEndpointName)

	projectResource := TestAccProjectResource(projectName)
	return fmt.Sprintf("%s\n%s", projectResource, serviceEndpointResource)
}

// TestAccServiceEndpointDockerHubResource HCL describing an AzDO service endpoint
func TestAccServiceEndpointDockerHubResource(projectName string, serviceEndpointName string) string {
	serviceEndpointResource := fmt.Sprintf(`
resource "azuredevops_serviceendpoint_dockerhub" "serviceendpoint" {
	project_id             = azuredevops_project.project.id
	service_endpoint_name  = "%s"
}`, serviceEndpointName)

	projectResource := TestAccProjectResource(projectName)
	return fmt.Sprintf("%s\n%s", projectResource, serviceEndpointResource)
}

// TestAccServiceEndpointAzureRMResource HCL describing an AzDO service endpoint
func TestAccServiceEndpointAzureRMResource(projectName string, serviceEndpointName string) string {
	serviceEndpointResource := fmt.Sprintf(`
resource "azuredevops_serviceendpoint_azurerm" "serviceendpointrm" {
	project_id             = azuredevops_project.project.id
	service_endpoint_name  = "%s"
	azurerm_spn_clientid 	="e318e66b-ec4b-4dff-9124-41129b9d7150"
	azurerm_spn_tenantid      = "9c59cbe5-2ca1-4516-b303-8968a070edd2"
    azurerm_subscription_id   = "3b0fee91-c36d-4d70-b1e9-fc4b9d608c3d"
    azurerm_subscription_name = "Microsoft Azure DEMO"
    azurerm_scope             = "/subscriptions/3b0fee91-c36d-4d70-b1e9-fc4b9d608c3d"
	azurerm_spn_clientsecret ="d9d210dd-f9f0-4176-afb8-a4df60e1ae72"

}`, serviceEndpointName)

	projectResource := TestAccProjectResource(projectName)
	return fmt.Sprintf("%s\n%s", projectResource, serviceEndpointResource)
}

// TestAccVariableGroupResource HCL describing an AzDO variable group
func TestAccVariableGroupResource(projectName string, variableGroupName string, allowAccess bool) string {
	variableGroupResource := fmt.Sprintf(`
resource "azuredevops_variable_group" "vg" {
	project_id  = azuredevops_project.project.id
	name        = "%s"
	description = "A sample variable group."
	allow_access = %t
	variable {
		name      = "key1"
		value     = "value1"
		is_secret = true
	}

	variable {
		name  = "key2"
		value = "value2"
	}

	variable {
		name = "key3"
	}
}`, variableGroupName, allowAccess)

	projectResource := TestAccProjectResource(projectName)
	return fmt.Sprintf("%s\n%s", projectResource, variableGroupResource)
}

// TestAccVariableGroupResourceNoSecrets Similar to TestAccVariableGroupResource, but without a secret variable
func TestAccVariableGroupResourceNoSecrets(projectName string, variableGroupName string, allowAccess bool) string {
	variableGroupResource := fmt.Sprintf(`
resource "azuredevops_variable_group" "vg" {
	project_id  = azuredevops_project.project.id
	name        = "%s"
	description = "A sample variable group."
	allow_access = %t
	variable {
		name      = "key1"
		value     = "value1"
	}
}`, variableGroupName, allowAccess)

	projectResource := TestAccProjectResource(projectName)
	return fmt.Sprintf("%s\n%s", projectResource, variableGroupResource)
}

// TestAccAgentPoolResource HCL describing an AzDO Agent Pool
func TestAccAgentPoolResource(poolName string) string {
	return fmt.Sprintf(`
resource "azuredevops_agent_pool" "pool" {
	name           = "%s"
	auto_provision = false
	pool_type      = "automation"
	}`, poolName)
}

// TestAccBuildDefinitionResourceGitHub HCL describing an AzDO build definition sourced from GitHub
func TestAccBuildDefinitionResourceGitHub(projectName string, buildDefinitionName string, buildPath string) string {
	return TestAccBuildDefinitionResource(
		projectName,
		buildDefinitionName,
		buildPath,
		"GitHub",
		"repoOrg/repoName",
		"master",
		"path/to/yaml",
		"")
}

// TestAccBuildDefinitionResourceBitbucket HCL describing an AzDO build definition sourced from Bitbucket
func TestAccBuildDefinitionResourceBitbucket(projectName string, buildDefinitionName string, buildPath string, serviceConnectionID string) string {
	return TestAccBuildDefinitionResource(
		projectName,
		buildDefinitionName,
		buildPath,
		"Bitbucket",
		"repoOrg/repoName",
		"master",
		"path/to/yaml",
		serviceConnectionID)
}

// TestAccBuildDefinitionResource HCL describing an AzDO build definition
func TestAccBuildDefinitionResource(
	projectName string,
	buildDefinitionName string,
	buildPath string,
	repoType string,
	repoName string,
	branchName string,
	yamlPath string,
	serviceConnectionID string,
) string {
	repositoryBlock := fmt.Sprintf(`
repository {
	repo_type             = "%s"
	repo_name             = "%s"
	branch_name           = "%s"
	yml_path              = "%s"
	service_connection_id = "%s"
}`, repoType, repoName, branchName, yamlPath, serviceConnectionID)

	buildDefinitionResource := fmt.Sprintf(`
resource "azuredevopsffuf_build_definition" "build" {
	project_id      = azuredevops_project.project.id
	name            = "%s"
	agent_pool_name = "Hosted Ubuntu 1604"
	path			= "%s"

	%s
}`, buildDefinitionName, strings.ReplaceAll(buildPath, `\`, `\\`), repositoryBlock)

	projectResource := TestAccProjectResource(projectName)
	return fmt.Sprintf("%s\n%s", projectResource, buildDefinitionResource)
}

// TestAccReleaseDefinitionResource HCL describing an AzDO release definition
func TestAccReleaseDefinitionResource(projectName string, releaseDefinitionName string, releasePath string) string {
	tasks := TestAccReleaseDefinitionTasks()
	releaseDefinitionResource := fmt.Sprintf(`
resource "azuredevopsffuf_release_definition" "release" {
  project_id = "DevOps"
  name = "%s"
  path = "\\"

  stage {
    name = "Stage 1"
    rank = 1

    agent_job {
      name = "Agent job 1"
      rank = 1
 	  demand {
		name =  "equals_condition_name"
		value = "x"
	  }
      demand {
		name =  "exists_condition_name"
	  }
      agent_pool_hosted_azure_pipelines {
        agent_pool_id = 2069
        agent_specification = "ubuntu-18.04"
      }
      timeout_in_minutes = 0
      max_execution_time_in_minutes = 1
      condition = "succeeded()"
		// overrideInput {} // TODO
		// enable_access_token ? Do we need this on this level?
    }

	agent_job {
      name = "Agent job 3"
      rank = 3
      agent_pool_hosted_azure_pipelines {
        agent_pool_id = 2069
        agent_specification = "ubuntu-18.04"
      }
	
	  multi_configuration {
		multipliers = "OperatingSystem"
		number_of_agents = 1
	  }

      condition = "succeeded()"
		// overrideInput {} // TODO
		// enable_access_token ? Do we need this on this level?
    }

	agent_job {
      name = "Agent job 2"
      rank = 2
      agent_pool_hosted_azure_pipelines {
        agent_pool_id = 2069
        agent_specification = "ubuntu-18.04"
      }
	  multi_agent {
		max_number_of_agents = 1
	  }
      condition = "succeeded()"
   	  // overrideInput {} // TODO
	  // enable_access_token ? Do we need this on this level?

	  dynamic "task" {
	    for_each = local.tasks
		content {
		  always_run =           lookup(task.value, "alwaysRun", null)
		  condition =            lookup(task.value, "condition", null)
		  continue_on_error =    lookup(task.value, "continueOnError", null)
		  enabled =              lookup(task.value, "enabled", null)
		  display_name =         lookup(task.value, "displayName", null)
          override_input =       lookup(task.value, "overrideInput", null)
          inputs =               lookup(task.value, "inputs", null)
          timeout_in_minutes =   lookup(task.value, "timeoutInMinutes", null)
          // task MUST come last as it overwites the task
          task =                 lookup(task.value, "task", null)
        }
	  }
    }

    pre_deploy_approval {
      approval {
        is_automated = true
        rank = 1
      }
    }

    post_deploy_approval {
      approval {
        is_automated = true
        rank = 1
      }
    }

    retention_policy {
      days_to_keep = 1
      releases_to_keep = 1
    }
  }
}

locals {
  tasks = yamldecode(<<YAML
%s
YAML
  )
}

`, releaseDefinitionName, tasks)

	projectResource := TestAccProjectResource(projectName)

	return fmt.Sprintf("%s\n%s", projectResource, releaseDefinitionResource)
}

// TestAccReleaseDefinitionTasks yaml of release tasks
func TestAccReleaseDefinitionTasks() string {
	return `
#refName: ''
# definitionType: task
# taskId: a433f589-fce1-4460-9ee6-44a624aeb1fb
# version: 0.*
- task: DownloadBuildArtifacts@0
  displayName: "Download Build Artifacts"        # name: Download Build Artifacts
  environment:			                        # default nil
    FOO: bar
  enabled: true				                    # default
#  alwaysRun: false			                    # default
  continueOnError: false	                     # default
  timeoutInMinutes: 0
  overrideInputs:			                    # default nil
    version: abc
  condition: succeeded()	                    # default
  inputs:
    buildType: current                          # default
    project: 0350d34d-fc00-4e9d-b1c5-78f8a7350b25
    definition: 80
    specificBuildWithTriggering: false          # default
    buildVersionToDownload: specific            # default
    allowPartiallySucceededBuilds: false        # default
    branchName: refs/heads/master
    buildId: ''                                 # default
    tags: ''                                    # default
    downloadType: specific                      # default
    artifactName: ''
    itemPattern: "**"
    downloadPath: "$(System.DefaultWorkingDirectory)"
    parallelizationLimit: '8'
# taskId: d9bafed4-0b18-4f58-968d-86655b4d2ce9
# version: 2.*
- task: CmdLine@2
  displayName: Add Permissions to AWS Provider Plugin Facility
  inputs:
    script: |
      ls -R
      chmod 0777 -R .terraform/plugins
    workingDirectory: "$(System.DefaultWorkingDirectory)/Infrastructure/AWS/environments/dev-1/us-west-2/service_facility/"
    failOnStderr: false
- task: TerraformInstaller@0
  displayName: "Install Terraform 0.12.13"
  inputs:
    terraformVersion: '0.12.13'
- task: AWSShellScript@1
  displayName: "Terraform Apply Tenant"
  inputs:
    awsCredentials: 'merlin-shared-azure-pipelines-build'
    regionName: 'us-west-2'
    scriptType: 'inline'
    inlineScript: |
      terraform apply tfplan
    disableAutoCwd: true
    workingDirectory: '$(System.DefaultWorkingDirectory)/Infrastructure/AWS/environments/dev-1/us-west-2/service_tenant/'
`
}

// TestAccReleaseDefinitionResourceTemp full terraform stanza to standup a release pipeline
func TestAccReleaseDefinitionResourceTemp(projectName string, releaseDefinitionName string, releasePath string) string {
	releaseDefinitionResource := fmt.Sprintf(`
resource "azuredevopsffuf_release_definition" "release" {
  project_id = "DevOps" // TODO: revert this back to azuredevops_project.project.id
  name = "%s"
  path = "\\"

  stage {
    name = "Stage 1"
    rank = 1

    deployment_group_job {
      name = "Deployment group job 1"
      rank = 1
      deployment_group_id = 1619 // QueueID
      //DeploymentHealthOption: OneAtATime
      tags = ["deployment_group_job_1"]
      timeout_in_minutes = 0
      max_execution_time_in_minutes = 1
      // download_artifact = {}
      allow_scripts_to_access_oauth_token = true
      condition = "succeedeed()"
      // tasks = file('foo.yml')
    }

	deployment_group_job {
      name = "Deployment group job 2"
      rank = 2
      deployment_group_id = 1619
      tags = ["deployment_group_job_2"]
      multiple { //DeploymentHealthOption: Custom
        max_targets_in_parallel = 0 // HealthPercent
      }
      timeout_in_minutes = 0
      max_execution_time_in_minutes = 1
      // download_artifact = {}
      allow_scripts_to_access_oauth_token = true
      condition = "succeedeed()"
      // tasks = file('foo.yml')
    }

    pre_deploy_approval {
      approval {
        is_automated = true
        rank = 1
      }
    }

    post_deploy_approval {
      approval {
        is_automated = true
        rank = 1
      }
    }

    retention_policy {
      days_to_keep = 1
      releases_to_keep = 1
    }
  }
}`, releaseDefinitionName)

	projectResource := TestAccProjectResource(projectName)
	return fmt.Sprintf("%s\n%s", projectResource, releaseDefinitionResource)
}

// TestAccReleaseDefinitionResourceAgentless full terraform stanza to standup a release pipeline
func TestAccReleaseDefinitionResourceAgentless(projectName string, releaseDefinitionName string, releasePath string) string {
	releaseDefinitionResource := fmt.Sprintf(`
resource "azuredevopsffuf_release_definition" "release" {
  project_id = "DevOps"
  name = "%s"
  path = "\\"

  stage {
    name = "Stage 1"
    rank = 1

    agentless_job {
      name = "Agentless job 1"
      rank = 1
      
      max_execution_time_in_minutes = 1
      timeout_in_minutes = 0
      condition = "succeeded()"
    }

    agentless_job {
      name = "Agentless job 2"
      rank = 2

      multi_configuration {
        multipliers = "OperatingSystem"
      }
	  max_execution_time_in_minutes = 1
      timeout_in_minutes = 0
      condition = "succeeded()"
    }

    pre_deploy_approval {
      approval {
        is_automated = true
        rank = 1
      }
    }

    post_deploy_approval {
      approval {
        is_automated = true
        rank = 1
      }
    }

    retention_policy {
      days_to_keep = 1
      releases_to_keep = 1
    }
  }
}`, releaseDefinitionName)

	projectResource := TestAccProjectResource(projectName)
	return fmt.Sprintf("%s\n%s", projectResource, releaseDefinitionResource)
}

// TestAccGroupMembershipResource full terraform stanza to standup a group membership
func TestAccGroupMembershipResource(projectName, groupName, userPrincipalName string) string {
	membershipDependenciesStanza := TestAccGroupMembershipDependencies(projectName, groupName, userPrincipalName)
	membershipStanza := `
resource "azuredevops_group_membership" "membership" {
	group = data.azuredevops_group.group.descriptor
	members = [azuredevops_user_entitlement.user.descriptor]
}`

	return membershipDependenciesStanza + "\n" + membershipStanza
}

// TestAccGroupMembershipDependencies all the dependencies needed to configure a group membership
func TestAccGroupMembershipDependencies(projectName, groupName, userPrincipalName string) string {
	return fmt.Sprintf(`
resource "azuredevops_project" "project" {
	project_name = "%s"
}
data "azuredevops_group" "group" {
	project_id = azuredevops_project.project.id
	name       = "%s"
}
resource "azuredevops_user_entitlement" "user" {
	principal_name       = "%s"
	account_license_type = "express"
}

output "group_descriptor" {
	value = data.azuredevops_group.group.descriptor
}
output "user_descriptor" {
	value = azuredevops_user_entitlement.user.descriptor
}
`, projectName, groupName, userPrincipalName)
}

// TestAccGroupResource HCL describing an AzDO group, if the projectName is empty, only a azuredevops_group instance is returned
func TestAccGroupResource(groupResourceName, projectName, groupName string) string {
	return fmt.Sprintf(`
%s

resource "azuredevops_group" "%s" {
	scope        = azuredevops_project.project.id
	display_name = "%s"
}

output "group_id_%s" {
	value = azuredevops_group.%s.id
}
`, TestAccProjectResource(projectName), groupResourceName, groupName, groupResourceName, groupResourceName)
}
