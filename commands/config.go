package command

import (
	cli "github.com/threepipes/circleci-env"
)

type ConfigCmd struct {
	Init ConfigInitCmd `cmd:"" help:"Initialize ccienv configurations."`
}

type ConfigInitCmd struct {
}

func (l *ConfigInitCmd) Run(c *Context) error {
	prompt := cli.Prompt{}
	org, err := prompt.ReadLine("Please set your default GitHub organization: ")
	if err != nil {
		return err
	}
	token, err := prompt.ReadSecret("Please set your personal API token: ")
	if err != nil {
		return err
	}
	cfg := cli.Config{
		OrganizationName: org,
		ApiToken:         token,
	}
	return cli.WriteConfig(&cfg)
}
