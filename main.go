package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"

	msgbroker "github.com/shapeblue/webhook-controller/messagebroker"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	commandPrefix = "/"
)

var RepoCommandsMap map[string]interface{}

func GetCommands() {
	jsonFile, err := os.Open("commands.json")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Successfully Opened commands.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	dataBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Println(err)
		return
	}
	json.Unmarshal(dataBytes, &RepoCommandsMap)
}

func getValidCommandsForRepo(repoName string) []string {
	repodata := RepoCommandsMap[repoName].(map[string]interface{})
	var commands = reflect.ValueOf(repodata).MapKeys()
	cmds := make([]string, len(commands))
	for i, cmd := range commands {
		cmds[i] = cmd.String()
	}
	return cmds
}

type JobData map[string]interface{}

type PRData struct {
	PR_ID        int
	Repo_URL     string
	RepoName     string
	ExchangeName string
}

func (pr PRData) getPRId() int {
	return pr.PR_ID
}

func (pr PRData) getRepoUrl() string {
	return pr.Repo_URL
}

func (pr PRData) getRepoName() string {
	return pr.RepoName
}

func (pr PRData) getExchangeName() string {
	return pr.ExchangeName
}

func contains(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func prettyPrint(eventData interface{}) {
	reqBodyBytes := new(bytes.Buffer)
	err := json.NewEncoder(reqBodyBytes).Encode(eventData)
	if err != nil {
		log.Println("Failed to encode event data")
	}
	var data bytes.Buffer
	err = json.Indent(&data, reqBodyBytes.Bytes(), "", "\t")
	if err != nil {
		log.Printf("error reading request body: err=%s\n", err)
		return
	}
	fmt.Printf("got webhook payload: %v", data.String())
}

func IfValidCommand(repoName, command string) bool {
	if contains(getValidCommandsForRepo(repoName), command) {
		return true
	}
	return false
}

func converToStringArray(data []interface{}) []string {
	s := make([]string, len(data))
	for i, v := range data {
		s[i] = fmt.Sprint(v)
	}
	return s
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	webhookBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading request body: err=%s\n", err)
		return
	}
	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), webhookBody)
	if err != nil {
		log.Printf("could not parse webhook: err=%s\n", err)
		return
	}
	switch event := event.(type) {
	case *github.PushEvent:
		log.Printf("Received a Push event from repo: %v", event.GetRepo().GetName())
		break

	case *github.PullRequestEvent:
		prEvent := *event
		log.Println("Recieved a pull request event")
		if event.Action != nil && contains([]string{"opened", "reopened"}, *event.Action) {
			repoUrl := strings.Split(event.GetPullRequest().GetHTMLURL(), "/pull")[0]
			prData := &PRData{
				PR_ID:    event.GetPullRequest().GetNumber(),
				Repo_URL: repoUrl,
				RepoName: repoUrl[strings.LastIndex(repoUrl, "/")+1:],
			}
			helpertext := PrintCommandsList(*prData)
			postResponseToGitHubRepo(*prData, helpertext)
		}

		if event.GetAction() == "closed" {
			log.Println("PR closed")
			if prEvent.PullRequest.GetMerged() {
				log.Println("PR Merged")
			} else {
				log.Println("PR closed")
			}
		}
		break

	case *github.IssueCommentEvent:
		comment := event.Comment.GetBody()
		log.Println("Recieved a pull request comment event")
		repoUrl := strings.Split(event.GetIssue().GetHTMLURL(), "/pull")[0]
		repoName := repoUrl[strings.LastIndex(repoUrl, "/")+1:]

		prData := &PRData{
			PR_ID:    event.GetIssue().GetNumber(),
			Repo_URL: repoUrl,
			RepoName: repoName,
		}

		if strings.HasPrefix(comment, commandPrefix) {
			splitCmd := strings.Split(comment, " ")
			command := splitCmd[0]
			args := splitCmd[1:]
			if IfValidCommand(repoName, command) {
				prData.ExchangeName = RepoCommandsMap[repoName].(map[string]interface{})[command].(map[string]interface{})["exchangeName"].(string)
				err = handleCommand(command, args, *prData)
			} else {
				log.Println("Not a command for me, ignoring..")
				// helpertext := PrintCommandsList(*prData)
				// err = postResponseToGitHubRepo(*prData, helpertext)
			}
			if err != nil {
				log.Printf("Error: %v", err.Error())
				postResponseToGitHubRepo(*prData, "Failed to create job to execute: "+command)
				return
			}
		} else {
			log.Println("Not a command, ignoring..")
		}
	}
}

func PrintCommandsList(prData PRData) string {
	repos := RepoCommandsMap[prData.getRepoName()].(map[string]interface{})
	resp := ""
	for cmd, data := range repos {
		if cmd == "owner" {
			continue
		}
		resp += fmt.Sprintf("\n<b>Command: `%s`</b> \n", cmd)
		validArgsList := converToStringArray(data.(map[string]interface{})["args"].([]interface{}))
		resp += helpText(prData, cmd, validArgsList, data.(map[string]interface{}))
	}

	response := fmt.Sprintf("<b>Valid Commands supported by this repo: </b>\n %v", resp)
	return response
}

func helpText(prData PRData, command string, validArgs []string, repoData map[string]interface{}) string {
	helpText := fmt.Sprintf("<b>Valid command format for `%s` is</b>: \n```\n%s ", command, command)
	for _, arg := range validArgs {
		helpText += fmt.Sprintf("[%s] ", arg)
	}

	helpText += fmt.Sprintf("\n```\nFollowing are supported values for each parameters: \n ```\n")
	for _, validArg := range validArgs {
		helpText += fmt.Sprintf("%s: %s\n", validArg, strings.Join(converToStringArray(repoData[validArg].([]interface{})), ","))
	}

	helpText += "```"

	return helpText
}

func validateCommandArgs(prData PRData, repoData map[string]interface{}, cmd string, args []string) (bool, []JobData) {
	validArgsList := repoData["args"].([]string)
	var dataParams []JobData
	for idx, arg := range args {
		validArg := validArgsList[idx]
		validArgVals := repoData[validArg].([]string)
		if !contains(validArgVals, arg) {
			resp := helpText(prData, cmd, validArgsList, repoData)
			postResponseToGitHubRepo(prData, resp)
			return false, nil
		}
		if validArg == "OS" {
			validArg = "TEMPLATE"
			arg = arg + "-kube"
		}
		dataParams = append(dataParams, JobData{
			"name":  validArg,
			"value": arg,
		})
	}

	return true, dataParams
}

func handleCommand(command string, args []string, prData PRData) error {
	repoData := RepoCommandsMap[prData.getRepoName()].(map[string]interface{})[command].(map[string]interface{})
	var dataParams []JobData
	if len(args) > 0 {
		isValid, params := validateCommandArgs(prData, repoData, command, args)
		if !isValid {
			return errors.New("Invalid command passed")
		}
		if len(params) > 0 {
			dataParams = append(dataParams, params...)
		}
	}

	dataParams = append(dataParams, JobData{
		"name":  "PR_ID",
		"value": prData.getPRId(),
	})

	dataParams = append(dataParams, JobData{
		"name":  "REPO_URL",
		"value": prData.getRepoUrl(),
	})

	var data = make(JobData)
	if data["parameter"] == nil {
		data["parameter"] = map[string]string{}
	}

	data["parameter"] = dataParams
	data["token"] = os.Getenv("GITHUB_WEBHOOK_SECRET")
	data["project"] = repoData["project"]

	msg := msgbroker.Message{
		ExchangeName: prData.getExchangeName(),
		Message:      data,
	}
	log.Println("Publishing job request to message broker")
	err := msgbroker.PublishMessage(msg)
	if err != nil {
		log.Println("FAILED to public message to broker")
		postResponseToGitHubRepo(prData, fmt.Sprintf("Failed to start Jenkins job for `%s`", command))
		return err
	}
	err = postResponseToGitHubRepo(prData, fmt.Sprintf("Jenkins job created for running `%s`, will keep you posted when the result is ready", command))
	return err
}

func postResponseToGitHubRepo(prData PRData, body string) error {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	d := int64(prData.getPRId())
	ic := &github.IssueComment{
		ID:   &d,
		Body: &body,
	}

	owner := RepoCommandsMap[prData.getRepoName()].(map[string]interface{})["owner"].(string)
	_, resp, err := client.Issues.CreateComment(ctx, owner, prData.getRepoName(), prData.getPRId(), ic)
	if err != nil {
		log.Printf("ERROR occured while printing response to repo: %v", err.Error())
		return err
	}
	log.Printf(": " + resp.Status)
	return nil
}

func main() {
	GetCommands()
	log.Println("server started")
	http.HandleFunc("/webhook", handleWebhook)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
