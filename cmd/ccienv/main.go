package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"
	cli "github.com/threepipes/circleci-env"
	command "github.com/threepipes/circleci-env/commands"
)

const version = "1.4.5"

var cmd struct {
	Version kong.VersionFlag `short:"v" help:"Display version of this tool."`
	Org     string           `short:"o" help:"Set your CircleCI organization name. If not specified, the default value is used."`
	Repo    string           `short:"r" help:"Set your target repository name. If not specified, the origin URL of the current directory's git project is used."`

	Rm  command.RmCmd  `cmd:"" help:"Remove environment variables. Either environment variables or the interactive flag must be specified."`
	Ls  command.LsCmd  `cmd:"" help:"List environment variables."`
	Add command.AddCmd `cmd:"" help:"Add an environment variable."`

	Config  command.ConfigCmd  `cmd:"" help:"Commands for ccienv configurations."`
	Project command.ProjectCmd `cmd:"" help:"Commands for CircleCI projects."`
}

func handleErr(err error) {
	if err != nil {
		logrus.WithField("error", err).Error("Internal error occured.")
		os.Exit(1)
	}
}

func extractRepoName(uri string) (string, string, error) {
	repoURI := strings.TrimSpace(string(uri))
	rmSuffix := regexp.MustCompile(`.git/?$`)
	repo := rmSuffix.ReplaceAllString(repoURI, "")
	r := regexp.MustCompile(`.+github\.com[:/]([^/]+)/(.+)/?$`)
	match := r.FindStringSubmatch(repo)
	if len(match) < 3 {
		return "", "", fmt.Errorf("failed to parse repo name: %v", repo)
	}
	return match[2], match[1], nil
}

func getDefaultRepoName() (string, string) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", strings.Split("config --get remote.origin.url", " ")...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		logrus.Error("Failed to read git remote repository with git command. Please specify the repository by the `-r` option or go to the directory where .git is with git command.")
		handleErr(err)
	}
	repo := strings.TrimSpace(string(out))
	rn, org, err := extractRepoName(repo)
	if err != nil {
		handleErr(err)
	}
	return rn, org
}

func constructProjectSlug(org string, repo string) string {
	return fmt.Sprintf("gh/%s/%s", org, repo)
}

func getClient() (*cli.Client, error) {
	cfg, err := cli.ReadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	org := cmd.Org
	if org == "" {
		org = cfg.OrganizationName
	}
	repo := cmd.Repo
	if repo == "" {
		repo, org = getDefaultRepoName()
	}

	slug := constructProjectSlug(org, repo)
	client, err := cli.NewClient(cfg, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	return client, nil
}

func mainRun() {
	kc := kong.Parse(&cmd, kong.Vars{"version": "ccienv version " + version})

	ctx := context.Background()
	err := kc.Run(&command.Context{
		Ctx:             ctx,
		ClientGenerator: getClient,
	})
	handleErr(err)
}

func main() {
	mainRun()
}
