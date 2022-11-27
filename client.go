package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/grezar/go-circleci"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=mock/$GOPACKAGE/$GOFILE
type UI interface {
	YesNo(msg string) (bool, error)
	SelectFromList(msg string, ls []string) ([]string, error)
	ReadSecret(msg string) (string, error)
	ReadInput(msg string) (string, error)
}

type Client struct {
	ci          *circleci.Client
	projectSlug string
	ui          UI

	token string
}

func NewClient(cfg *Config, prj string) (*Client, error) {
	config := circleci.DefaultConfig()
	config.Token = cfg.ApiToken
	ci, err := circleci.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("new client: %w", err)
	}
	return &Client{
		ci:          ci,
		projectSlug: prj,
		ui:          &Prompt{},
		token:       cfg.ApiToken,
	}, nil
}

func getMaxNameLength(pv []*circleci.ProjectVariable) int {
	maxlen := 0
	for _, v := range pv {
		if len(v.Name) > maxlen {
			maxlen = len(v.Name)
		}
	}
	return maxlen
}

func dumpVariables(pv []*circleci.ProjectVariable) {
	maxlen := getMaxNameLength(pv)
	for _, v := range pv {
		fmt.Printf("%-*s %s\n", maxlen, v.Name, v.Value)
	}
}

func convertToString(pv []*circleci.ProjectVariable) []string {
	maxlen := getMaxNameLength(pv)
	res := make([]string, len(pv))
	for i, v := range pv {
		res[i] = fmt.Sprintf("%-*s %s", maxlen, v.Name, v.Value)
	}
	return res
}

func getFoundAndNotFoundVariables(vars []string, items []*circleci.ProjectVariable) ([]*circleci.ProjectVariable, []string) {
	mp := make(map[string]*circleci.ProjectVariable, 0)
	for _, v := range items {
		mp[v.Name] = v
	}

	in := make([]*circleci.ProjectVariable, 0)
	out := make([]string, 0)
	for _, v := range vars {
		pv, prs := mp[v]
		if !prs {
			out = append(out, v)
		} else {
			in = append(in, pv)
		}
	}
	return in, out
}

func (c *Client) deleteVariables(ctx context.Context, dels []*circleci.ProjectVariable) error {
	if len(dels) == 0 {
		return fmt.Errorf("no values are specified")
	}

	fmt.Println("These variables will be removed.")
	fmt.Println()
	dumpVariables(dels)
	fmt.Println()

	yes, err := c.ui.YesNo("Do you want to continue?")
	if err != nil {
		return fmt.Errorf("delete vars: %w", err)
	}
	if !yes {
		fmt.Println("Cancelled.")
		return nil
	}

	for _, v := range dels {
		if err := c.ci.Projects.DeleteVariable(ctx, c.projectSlug, v.Name); err != nil {
			logrus.WithField("key", v).Errorf("Failed to delete: %v\n", err)
		} else {
			fmt.Printf("Deleted: %s\n", v.Name)
		}
	}
	return nil
}

func makeReverseResolutionMap(vs []string) map[string]int {
	mp := make(map[string]int, 0)
	for i, v := range vs {
		mp[v] = i
	}
	// For any `key` in vs, vs[mp[key]] == key
	return mp
}

func (c *Client) DeleteVariablesInteractive(ctx context.Context) error {
	pv, err := c.ci.Projects.ListVariables(ctx, c.projectSlug)
	if err != nil {
		return fmt.Errorf("delete vars: %w", err)
	}
	if pv.NextPageToken != "" {
		logrus.Warn("Warning! Not all variables are listed.")
	}
	spv := convertToString(pv.Items)
	sel, err := c.ui.SelectFromList("Choose variables to be deleted.", spv)
	if err != nil {
		return fmt.Errorf("delete vars: %w", err)
	}

	rrm := makeReverseResolutionMap(spv)
	dels := make([]*circleci.ProjectVariable, len(sel))
	for i, s := range sel {
		dels[i] = pv.Items[rrm[s]]
	}

	return c.deleteVariables(ctx, dels)
}

func (c *Client) DeleteVariables(ctx context.Context, vars []string) error {
	pv, err := c.ci.Projects.ListVariables(ctx, c.projectSlug)
	if err != nil {
		return fmt.Errorf("delete vars: %w", err)
	}
	if pv.NextPageToken != "" {
		logrus.Warn("Warning! Not all variables are listed.")
	}

	dels, nonDels := getFoundAndNotFoundVariables(vars, pv.Items)
	if len(nonDels) > 0 {
		fmt.Println("These variables are not found.")
		for _, v := range nonDels {
			fmt.Println("  " + v)
		}
		fmt.Println()
	}
	if len(dels) == 0 {
		fmt.Println("There are no deleted variables.")
		return nil
	}
	return c.deleteVariables(ctx, dels)
}

func (c *Client) UpdateOrCreateVariable(ctx context.Context, key string, val string) error {
	v, _ := c.ci.Projects.GetVariable(ctx, c.projectSlug, key)
	if v != nil {
		fmt.Printf("key:%s already exists as value=%s\n", v.Name, v.Value)
		yes, err := c.ui.YesNo("Do you want to overwrite?")
		if err != nil {
			return err
		}
		if !yes {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	pv, err := c.ci.Projects.CreateVariable(ctx, c.projectSlug, circleci.ProjectCreateVariableOptions{
		Name:  &key,
		Value: &val,
	})
	if err != nil {
		return fmt.Errorf("update or create variable for key=%s: %w", key, err)
	}
	fmt.Printf("%s=%s is created\n", pv.Name, pv.Value)
	return nil
}

func (c *Client) ListVariables(ctx context.Context) error {
	vars, err := c.ci.Projects.ListVariables(ctx, c.projectSlug)
	if err != nil {
		return fmt.Errorf("list vars: %w", err)
	}
	dumpVariables(vars.Items)
	if vars.NextPageToken != "" {
		logrus.WithField("NextPageToken", vars.NextPageToken).Warn("Not all values are displayed")
	}
	return nil
}

func (c *Client) request(ctx context.Context, path string) ([]byte, error) {
	url := fmt.Sprintf("https://circleci.com/api/v2/project/%s%s", c.projectSlug, path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	req.Header.Add("Circle-Token", c.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	return body, nil
}

func (c *Client) ShowProject(ctx context.Context) error {
	body, err := c.request(ctx, "")
	if err != nil {
		return fmt.Errorf("show project: %w", err)
	}
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		return fmt.Errorf("show project: %w", err)
	}
	bt, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("show project: %w", err)
	}
	fmt.Println(string(bt))
	return nil
}
