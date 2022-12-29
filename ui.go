package cli

import (
	"bufio"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
)

type Prompt struct {
}

var _ UI = &Prompt{}

func (p *Prompt) YesNo(msg string) (bool, error) {
	ans := false
	pmt := &survey.Confirm{
		Message: msg,
	}
	err := survey.AskOne(pmt, &ans)
	if err != nil {
		return false, fmt.Errorf("prompt: %w", err)
	}
	return ans, nil
}

func (p *Prompt) SelectFromList(msg string, ls []string) ([]string, error) {
	ans := []string{}
	pmt := &survey.MultiSelect{
		Message: msg,
		Options: ls,
	}
	err := survey.AskOne(pmt, &ans)
	if err != nil {
		return nil, fmt.Errorf("select from list: %w", err)
	}
	return ans, nil
}

func (p *Prompt) ReadSecret(msg string) (string, error) {
	ans := ""
	pmt := &survey.Password{
		Message: msg,
	}
	err := survey.AskOne(pmt, &ans)
	if err != nil {
		return "", fmt.Errorf("read secret: %w", err)
	}
	return ans, nil
}

func (p *Prompt) ReadLine(msg string) (string, error) {
	ans := ""
	pmt := &survey.Input{
		Message: msg,
	}
	err := survey.AskOne(pmt, &ans)
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	return ans, nil
}

func (p *Prompt) ReadAll(msg string) (string, error) {
	fmt.Println(msg)
	scn := bufio.NewScanner(os.Stdin)
	ans := ""
	for scn.Scan() {
		ans += scn.Text() + "\n"
	}
	if err := scn.Err(); err != nil {
		return "", fmt.Errorf("read input without tty: %w", err)
	}
	fmt.Println(ans)
	return ans, nil
}
