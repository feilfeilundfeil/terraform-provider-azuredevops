// +build all resource_build_definition

package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"github.com/microsoft/terraform-provider-azuredevops/azdosdkmocks"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/converter"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/testhelper"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/config"
	"github.com/stretchr/testify/require"
)

var testProjectID = uuid.New().String()

// This definition matches the overall structure of what a configured git repository would
// look like. Note that the ID and Name attributes match -- this is the service-side behavior
// when configuring a GitHub repo.
var testBuildDefinition = build.BuildDefinition{
	Id:       converter.Int(100),
	Revision: converter.Int(1),
	Name:     converter.String("Name"),
	Path:     converter.String("\\"),
	Repository: &build.BuildRepository{
		Url:           converter.String("https://github.com/RepoId.git"),
		Id:            converter.String("RepoId"),
		Name:          converter.String("RepoId"),
		DefaultBranch: converter.String("RepoBranchName"),
		Type:          converter.String("GitHub"),
		Properties: &map[string]string{
			"connectedServiceId": "ServiceConnectionID",
		},
	},
	Process: &build.YamlProcess{
		YamlFilename: converter.String("YamlFilename"),
	},
	Queue: &build.AgentPoolQueue{
		Name: converter.String("BuildPoolName"),
		Pool: &build.TaskAgentPoolReference{
			Name: converter.String("BuildPoolName"),
		},
	},
	QueueStatus: &build.DefinitionQueueStatusValues.Enabled,
	Type:        &build.DefinitionTypeValues.Build,
	Quality:     &build.DefinitionQualityValues.Definition,
	Triggers: &[]interface{}{
		map[string]interface{}{
			"branchFilters": &[]string{
				"+master",
				"+develop",
				"+feature",
				"-test",
			},
			//"pathFilters":                  &[]string{"+$/Root/Child Root/Child2", "-"},
			"batchChanges": converter.Bool(true),
			//"maxConcurrentBuildsPerBranch": 1,
			//"pollingInterval":              0,
			"triggerType": converter.String("continuousIntegration"),
		},
	},
	VariableGroups: &[]build.VariableGroup{},
}

/**
 * Begin unit tests
 */

// validates that all supported repo types are allowed by the schema
func TestAzureDevOpsBuildDefinition_RepoTypeListIsCorrect(t *testing.T) {
	expectedRepoTypes := []string{"GitHub", "TfsGit"}
	repoSchema := resourceBuildDefinition().Schema["repository"]
	repoTypeSchema := repoSchema.Elem.(*schema.Resource).Schema["repo_type"]

	for _, repoType := range expectedRepoTypes {
		_, errors := repoTypeSchema.ValidateFunc(repoType, "")
		require.Equal(t, 0, len(errors), "Repo type unexpectedly did not pass validation")
	}
}

// validates that an error is thrown if any of the un-supported path characters are used
func TestAzureDevOpsBuildDefinition_PathInvalidCharacterListIsError(t *testing.T) {
	expectedInvalidPathCharacters := []string{"<", ">", "|", ":", "$", "@", "\"", "/", "%", "+", "*", "?"}
	pathSchema := resourceBuildDefinition().Schema["path"]

	for _, invalidCharacter := range expectedInvalidPathCharacters {
		_, errors := pathSchema.ValidateFunc(`\`+invalidCharacter, "")
		require.Equal(t, "<>|:$@\"/%+*? are not allowed in path", errors[0].Error())
	}
}

// validates that an error is thrown if path does not start with slash
func TestAzureDevOpsBuildDefinition_PathInvalidStartingSlashIsError(t *testing.T) {
	pathSchema := resourceBuildDefinition().Schema["path"]
	_, errors := pathSchema.ValidateFunc("dir\\dir", "")
	require.Equal(t, "path must start with backslash", errors[0].Error())
}

// verifies that the flatten/expand round trip yields the same build definition
func TestAzureDevOpsBuildDefinition_ExpandFlatten_Roundtrip(t *testing.T) {
	resourceData := schema.TestResourceDataRaw(t, resourceBuildDefinition().Schema, nil)
	flattenBuildDefinition(resourceData, &testBuildDefinition, testProjectID)

	buildDefinitionAfterRoundTrip, projectID, err := expandBuildDefinition(resourceData)

	require.Nil(t, err)
	require.Equal(t, testBuildDefinition, *buildDefinitionAfterRoundTrip)
	require.Equal(t, testProjectID, projectID)
}

// verifies that an expand will fail if there is insufficient configuration data found in the resource
func TestAzureDevOpsBuildDefinition_Expand_FailsIfNotEnoughData(t *testing.T) {
	resourceData := schema.TestResourceDataRaw(t, resourceBuildDefinition().Schema, nil)
	_, _, err := expandBuildDefinition(resourceData)
	require.NotNil(t, err)
}

// verifies that if an error is produced on create, the error is not swallowed
func TestAzureDevOpsBuildDefinition_Create_DoesNotSwallowError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resourceData := schema.TestResourceDataRaw(t, resourceBuildDefinition().Schema, nil)
	flattenBuildDefinition(resourceData, &testBuildDefinition, testProjectID)

	buildClient := azdosdkmocks.NewMockBuildClient(ctrl)
	clients := &config.AggregatedClient{BuildClient: buildClient, Ctx: context.Background()}

	expectedArgs := build.CreateDefinitionArgs{Definition: &testBuildDefinition, Project: &testProjectID}
	buildClient.
		EXPECT().
		CreateDefinition(clients.Ctx, expectedArgs).
		Return(nil, errors.New("CreateDefinition() Failed")).
		Times(1)

	err := resourceBuildDefinitionCreate(resourceData, clients)
	require.Contains(t, err.Error(), "CreateDefinition() Failed")
}

// verifies that if an error is produced on a read, it is not swallowed
func TestAzureDevOpsBuildDefinition_Read_DoesNotSwallowError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resourceData := schema.TestResourceDataRaw(t, resourceBuildDefinition().Schema, nil)
	flattenBuildDefinition(resourceData, &testBuildDefinition, testProjectID)

	buildClient := azdosdkmocks.NewMockBuildClient(ctrl)
	clients := &config.AggregatedClient{BuildClient: buildClient, Ctx: context.Background()}

	expectedArgs := build.GetDefinitionArgs{DefinitionId: testBuildDefinition.Id, Project: &testProjectID}
	buildClient.
		EXPECT().
		GetDefinition(clients.Ctx, expectedArgs).
		Return(nil, errors.New("GetDefinition() Failed")).
		Times(1)

	err := resourceBuildDefinitionRead(resourceData, clients)
	require.Equal(t, "GetDefinition() Failed", err.Error())
}

// verifies that if an error is produced on a delete, it is not swallowed
func TestAzureDevOpsBuildDefinition_Delete_DoesNotSwallowError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resourceData := schema.TestResourceDataRaw(t, resourceBuildDefinition().Schema, nil)
	flattenBuildDefinition(resourceData, &testBuildDefinition, testProjectID)

	buildClient := azdosdkmocks.NewMockBuildClient(ctrl)
	clients := &config.AggregatedClient{BuildClient: buildClient, Ctx: context.Background()}

	expectedArgs := build.DeleteDefinitionArgs{DefinitionId: testBuildDefinition.Id, Project: &testProjectID}
	buildClient.
		EXPECT().
		DeleteDefinition(clients.Ctx, expectedArgs).
		Return(errors.New("DeleteDefinition() Failed")).
		Times(1)

	err := resourceBuildDefinitionDelete(resourceData, clients)
	require.Equal(t, "DeleteDefinition() Failed", err.Error())
}

// verifies that if an error is produced on an update, it is not swallowed
func TestAzureDevOpsBuildDefinition_Update_DoesNotSwallowError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resourceData := schema.TestResourceDataRaw(t, resourceBuildDefinition().Schema, nil)
	flattenBuildDefinition(resourceData, &testBuildDefinition, testProjectID)

	buildClient := azdosdkmocks.NewMockBuildClient(ctrl)
	clients := &config.AggregatedClient{BuildClient: buildClient, Ctx: context.Background()}

	expectedArgs := build.UpdateDefinitionArgs{
		Definition:   &testBuildDefinition,
		DefinitionId: testBuildDefinition.Id,
		Project:      &testProjectID,
	}

	buildClient.
		EXPECT().
		UpdateDefinition(clients.Ctx, expectedArgs).
		Return(nil, errors.New("UpdateDefinition() Failed")).
		Times(1)

	err := resourceBuildDefinitionUpdate(resourceData, clients)
	require.Equal(t, "UpdateDefinition() Failed", err.Error())
}

/**
 * Begin acceptance tests
 */

// validates that an apply followed by another apply (i.e., resource update) will be reflected in AzDO and the
// underlying terraform state.
func TestAccAzureDevOpsBuildDefinition_CreateAndUpdate(t *testing.T) {
	projectName := testhelper.TestAccResourcePrefix + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	buildDefinitionPathEmpty := `\`
	buildDefinitionNameFirst := testhelper.TestAccResourcePrefix + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	buildDefinitionNameSecond := testhelper.TestAccResourcePrefix + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	buildDefinitionPathFirst := `\` + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	buildDefinitionPathSecond := `\` + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	buildDefinitionPathThird := `\` + buildDefinitionNameFirst + `\` + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	buildDefinitionPathFourth := `\` + buildDefinitionNameSecond + `\` + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	tfBuildDefNode := "azuredevops_build_definition.build"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testhelper.TestAccPreCheck(t, nil) },
		Providers:    testAccProviders,
		CheckDestroy: testAccBuildDefinitionCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testhelper.TestAccBuildDefinitionResource(projectName, buildDefinitionNameFirst, buildDefinitionPathEmpty),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "project_id"),
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "revision"),
					resource.TestCheckResourceAttr(tfBuildDefNode, "name", buildDefinitionNameFirst),
					resource.TestCheckResourceAttr(tfBuildDefNode, "path", buildDefinitionPathEmpty),
					testAccCheckBuildDefinitionResourceExists(buildDefinitionNameFirst),
				),
			}, {
				Config: testhelper.TestAccBuildDefinitionResource(projectName, buildDefinitionNameSecond, buildDefinitionPathEmpty),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "project_id"),
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "revision"),
					resource.TestCheckResourceAttr(tfBuildDefNode, "name", buildDefinitionNameSecond),
					resource.TestCheckResourceAttr(tfBuildDefNode, "path", buildDefinitionPathEmpty),
					testAccCheckBuildDefinitionResourceExists(buildDefinitionNameSecond),
				),
			}, {
				Config: testhelper.TestAccBuildDefinitionResource(projectName, buildDefinitionNameFirst, buildDefinitionPathFirst),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "project_id"),
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "revision"),
					resource.TestCheckResourceAttr(tfBuildDefNode, "name", buildDefinitionNameFirst),
					resource.TestCheckResourceAttr(tfBuildDefNode, "path", buildDefinitionPathFirst),
					testAccCheckBuildDefinitionResourceExists(buildDefinitionNameFirst),
				),
			}, {
				Config: testhelper.TestAccBuildDefinitionResource(projectName, buildDefinitionNameFirst,
					buildDefinitionPathSecond),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "project_id"),
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "revision"),
					resource.TestCheckResourceAttr(tfBuildDefNode, "name", buildDefinitionNameFirst),
					resource.TestCheckResourceAttr(tfBuildDefNode, "path", buildDefinitionPathSecond),
					testAccCheckBuildDefinitionResourceExists(buildDefinitionNameFirst),
				),
			}, {
				Config: testhelper.TestAccBuildDefinitionResource(projectName, buildDefinitionNameFirst, buildDefinitionPathThird),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "project_id"),
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "revision"),
					resource.TestCheckResourceAttr(tfBuildDefNode, "name", buildDefinitionNameFirst),
					resource.TestCheckResourceAttr(tfBuildDefNode, "path", buildDefinitionPathThird),
					testAccCheckBuildDefinitionResourceExists(buildDefinitionNameFirst),
				),
			}, {
				Config: testhelper.TestAccBuildDefinitionResource(projectName, buildDefinitionNameFirst, buildDefinitionPathFourth),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "project_id"),
					resource.TestCheckResourceAttrSet(tfBuildDefNode, "revision"),
					resource.TestCheckResourceAttr(tfBuildDefNode, "name", buildDefinitionNameFirst),
					resource.TestCheckResourceAttr(tfBuildDefNode, "path", buildDefinitionPathFourth),
					testAccCheckBuildDefinitionResourceExists(buildDefinitionNameFirst),
				),
			}, {
				// Resource Acceptance Testing https://www.terraform.io/docs/extend/resources/import.html#resource-acceptance-testing-implementation
				ResourceName:      tfBuildDefNode,
				ImportStateIdFunc: testAccImportStateIDFunc(tfBuildDefNode),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Given the name of an AzDO build definition, this will return a function that will check whether
// or not the definition (1) exists in the state and (2) exist in AzDO and (3) has the correct name
func testAccCheckBuildDefinitionResourceExists(expectedName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		buildDef, ok := s.RootModule().Resources["azuredevops_build_definition.build"]
		if !ok {
			return fmt.Errorf("Did not find a build definition in the TF state")
		}

		buildDefinition, err := getBuildDefinitionFromResource(buildDef)
		if err != nil {
			return err
		}

		if *buildDefinition.Name != expectedName {
			return fmt.Errorf("Build Definition has Name=%s, but expected Name=%s", *buildDefinition.Name, expectedName)
		}

		return nil
	}
}

// verifies that all build definitions referenced in the state are destroyed. This will be invoked
// *after* terrafform destroys the resource but *before* the state is wiped clean.
func testAccBuildDefinitionCheckDestroy(s *terraform.State) error {
	for _, resource := range s.RootModule().Resources {
		if resource.Type != "azuredevops_build_definition" {
			continue
		}

		// indicates the build definition still exists - this should fail the test
		if _, err := getBuildDefinitionFromResource(resource); err == nil {
			return fmt.Errorf("Unexpectedly found a build definition that should be deleted")
		}
	}

	return nil
}

// given a resource from the state, return a build definition (and error)
func getBuildDefinitionFromResource(resource *terraform.ResourceState) (*build.BuildDefinition, error) {
	buildDefID, err := strconv.Atoi(resource.Primary.ID)
	if err != nil {
		return nil, err
	}

	projectID := resource.Primary.Attributes["project_id"]
	clients := testAccProvider.Meta().(*config.AggregatedClient)
	return clients.BuildClient.GetDefinition(clients.Ctx, build.GetDefinitionArgs{
		Project:      &projectID,
		DefinitionId: &buildDefID,
	})
}

func init() {
	InitProvider()
}
