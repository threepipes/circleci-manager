package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/grezar/go-circleci"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	mock_cli "github.com/threepipes/circleci-env/mock/cli"
)

func Test_getFoundAndNotFoundVariables(t *testing.T) {
	type args struct {
		vars  []string
		items []*circleci.ProjectVariable
	}
	tests := []struct {
		name  string
		args  args
		want  []*circleci.ProjectVariable
		want1 []string
	}{
		{
			name: "simple array",
			args: args{
				vars: []string{"ENV_0", "ENV_1", "ENV_2"},
				items: []*circleci.ProjectVariable{
					{Name: "ENV_1", Value: "xxxxenv"},
				},
			},
			want: []*circleci.ProjectVariable{
				{Name: "ENV_1", Value: "xxxxenv"},
			},
			want1: []string{
				"ENV_0", "ENV_2",
			},
		},
		{
			name: "two intersected items",
			args: args{
				vars: []string{"ENV_0", "ENV_1", "ENV_2"},
				items: []*circleci.ProjectVariable{
					{Name: "ENV_1", Value: "xxxxenv"},
					{Name: "ENV_2", Value: "xxxxenv"},
					{Name: "ENV_3", Value: "xxxxenv"},
				},
			},
			want: []*circleci.ProjectVariable{
				{Name: "ENV_1", Value: "xxxxenv"},
				{Name: "ENV_2", Value: "xxxxenv"},
			},
			want1: []string{
				"ENV_0",
			},
		},
		{
			name: "no intersection",
			args: args{
				vars: []string{"ENV_0", "ENV_1", "ENV_2"},
				items: []*circleci.ProjectVariable{
					{Name: "ENV_3", Value: "xxxxenv"},
				},
			},
			want: []*circleci.ProjectVariable{},
			want1: []string{
				"ENV_0", "ENV_1", "ENV_2",
			},
		},
		{
			name: "no envs",
			args: args{
				vars:  []string{"ENV_0", "ENV_1", "ENV_2"},
				items: []*circleci.ProjectVariable{},
			},
			want:  []*circleci.ProjectVariable{},
			want1: []string{"ENV_0", "ENV_1", "ENV_2"},
		},
		{
			name: "no vars",
			args: args{
				vars: []string{},
				items: []*circleci.ProjectVariable{
					{Name: "ENV_0", Value: "xxxxenv"},
				},
			},
			want:  []*circleci.ProjectVariable{},
			want1: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getFoundAndNotFoundVariables(tt.args.vars, tt.args.items)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDeletedAndNotDeletedVariables() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("getDeletedAndNotDeletedVariables() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

const projectSlug = "gh/testorg/testprj"
const apiBaseURL = "https://circleci.com/api/v2/project/" + projectSlug
const testAPIToken = "testtoken"

func TestClient_DeleteVariablesInteractive(t *testing.T) {
	config := circleci.DefaultConfig()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	expectedListURL := apiBaseURL + "/envvar"
	expectedDeleteURLs := []string{
		apiBaseURL + "/envvar/BAR",
		apiBaseURL + "/envvar/TEST1",
	}

	pvl := circleci.ProjectVariableList{
		Items: []*circleci.ProjectVariable{
			{Name: "FOO", Value: "xxxx_foo"},
			{Name: "BAR", Value: "xxxx_bar"},
			{Name: "TEST0", Value: "xxxxtest"},
			{Name: "TEST1", Value: "xxxxest1"},
			{Name: "TEST2", Value: "xxxxest2"},
		},
	}

	listResp, err := httpmock.NewJsonResponder(200, pvl)
	if err != nil {
		t.Error(err)
	}
	httpmock.RegisterResponder("GET", expectedListURL, listResp)

	for _, d := range expectedDeleteURLs {
		httpmock.RegisterResponder("DELETE", d,
			httpmock.NewStringResponder(200, `{"message":"OK"}`))
	}

	ctrl := gomock.NewController(t)
	ui := mock_cli.NewMockUI(ctrl)
	spv := convertToString(pvl.Items)
	ui.EXPECT().SelectFromList(gomock.Any(), spv).Return([]string{spv[1], spv[3]}, nil)
	ui.EXPECT().YesNo(gomock.Any()).Return(true, nil)

	config.HTTPClient = http.DefaultClient
	config.Token = testAPIToken
	ci, err := circleci.NewClient(config)
	if err != nil {
		t.Error(err)
	}

	c := &Client{
		ci:          ci,
		projectSlug: projectSlug,
		ui:          ui,
		token:       testAPIToken,
	}
	if err := c.DeleteVariablesInteractive(context.Background()); err != nil {
		t.Error(err)
	}
	info := httpmock.GetCallCountInfo()
	assert.Equal(t, 1, info["GET "+expectedListURL], "Expected number of list API call is wrong")
	for _, d := range expectedDeleteURLs {
		assert.Equal(t, 1, info["DELETE "+d], "Expected number of delete API call is wrong")
	}
}

func updateOrCreateScaffold(t *testing.T, expected *circleci.ProjectVariable, exists bool) func() {
	expectedGetURL := apiBaseURL + "/envvar/" + expected.Name
	expectedCreateURL := apiBaseURL + "/envvar"
	if exists {
		resp, err := httpmock.NewJsonResponder(200, expected)
		if err != nil {
			t.Errorf("Failed to convert the expected variable: %v", err)
		}
		httpmock.RegisterResponder("GET", expectedGetURL, resp)
	} else {
		httpmock.RegisterResponder("GET", expectedGetURL, httpmock.NewNotFoundResponder(nil))
	}
	httpmock.RegisterResponder("POST", expectedCreateURL,
		func(r *http.Request) (*http.Response, error) {
			var pv circleci.ProjectVariable
			err := json.NewDecoder(r.Body).Decode(&pv)
			if err != nil {
				msg := fmt.Sprintf("Failed to convert the expected variable: %v", err)
				return httpmock.NewStringResponse(500, msg), nil
			}
			resp, err := httpmock.NewJsonResponse(201, expected)
			if err != nil {
				msg := fmt.Sprintf("Failed to convert the expected variable: %v", err)
				return httpmock.NewStringResponse(500, msg), nil
			}
			return resp, nil
		})
	checker := func() {
		info := httpmock.GetCallCountInfo()
		assert.Equal(t, 1, info["GET "+expectedGetURL], "Expected number of get API call is wrong")
		assert.Equal(t, 1, info["POST "+expectedCreateURL], "Expected number of post API call is wrong")
	}
	return checker
}

func TestClient_UpdateOrCreateVariable(t *testing.T) {
	type args struct {
		pv            *circleci.ProjectVariable
		alreadyExists bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "The variable doesn't exist",
			args: args{
				pv: &circleci.ProjectVariable{
					Name:  "test1",
					Value: "xxxxEnv1",
				},
				alreadyExists: false,
			},
		},
		{
			name: "The variable already exists",
			args: args{
				pv: &circleci.ProjectVariable{
					Name:  "test2",
					Value: "xxxxEnv2",
				},
				alreadyExists: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			checker := updateOrCreateScaffold(t, tt.args.pv, tt.args.alreadyExists)
			defer checker()

			config := circleci.DefaultConfig()
			config.HTTPClient = http.DefaultClient
			config.Token = testAPIToken
			ci, err := circleci.NewClient(config)
			if err != nil {
				t.Error(err)
			}

			ctrl := gomock.NewController(t)
			ui := mock_cli.NewMockUI(ctrl)
			if tt.args.alreadyExists {
				ui.EXPECT().YesNo(gomock.Any()).Return(true, nil)
			}

			c := &Client{
				ci:          ci,
				projectSlug: projectSlug,
				ui:          ui,
				token:       testAPIToken,
			}
			if err := c.UpdateOrCreateVariable(context.Background(), tt.args.pv.Name, tt.args.pv.Value); err != nil {
				t.Errorf("Client.UpdateOrCreateVariable() error = %v", err)
			}
		})
	}
}
