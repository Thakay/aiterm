package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/atotto/clipboard"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	varKeyName = "OPENAI_KEY"
	endPoint   = "https://api.openai.com/v1/chat/completions"
)

func main() {

	apiKey := os.Getenv(varKeyName)
	flagKey := flag.String("key", apiKey, "API key from OpenAI")
	flagURL := flag.String("url", endPoint, "custom API endpoints to interact with")

	flag.Parse()
	apiKey = *flagKey
	apiURL := *flagURL

	userRequest := "a command to find all the occurrences of the word foo in all the files of this directory" //"list all the files and folders of parent directory with extra details" //"copy the currernt directory to a new folder in desktop named copied"
	positionalArgs := flag.Args()
	if len(positionalArgs) == 0 {
		fmt.Println("No prompt was provided.")
		return
	} else {
		userRequest = positionalArgs[0]
	}

	reader := bufio.NewReader(os.Stdin)
	apiKey, err := handleApiKey(apiKey, reader)
	if err != nil {
		log.Println(err)
	}

	for {
		cmdstr, err := api(apiKey, apiURL, userRequest)
		if err != nil {
			if retry, newApiKey := handleAPIError(err, reader); retry {
				apiKey = newApiKey
				continue
			}
			log.Fatalf("failed criticial API calling: %v \n Exiting...\n", err)
		}
		//cmdstr := "ls -l asd"
		if ok, cmdstr := validateCmd(cmdstr); !ok {
			fmt.Println("Please retry with a different prompt.")
			input, err := readInput(reader)
			if err != nil {
				log.Printf("\"error: %v \n", err)
			}
			userRequest = input
			continue
		} else {
			if res, err := handleCmd(cmdstr, reader); err == nil {
				fmt.Println(res)
				break
			}
			log.Println("failed handling the command: ", err)
			break
		}
	}

}
func handleApiKey(apiKey string, reader *bufio.Reader) (string, error) {
	if apiKey == "" {

		fmt.Println("Please set the Environment Variable named OPENAI_KEY with you secret API key manually and retry. or enter it for temporary use.")
		fmt.Print("Enter your OpenAI API key or empty to exit: ")

		if input, err := readInput(reader); err == nil {
			if input != "" {
				return input, nil
			}
			fmt.Println("No API Key was provided... Please make sure to set the Environment variable. Exiting... ")
			return "", nil
		} else {
			log.Printf("\"error: %v \n", err)
		}
	}
	return apiKey, nil
}

func validateCmd(cmdstr string) (bool, string) {
	cmdstr = strings.TrimSpace(cmdstr)
	if strings.ToLower(cmdstr) == "not a command" || strings.ToLower(cmdstr) == "not a command." {
		return false, ""
	}
	return true, cmdstr
}

func handleCmd(cmdstr string, reader *bufio.Reader) (string, error) {

	for {
		fmt.Print("\n\n\n")
		fmt.Printf("Here is the command --> %s <-- \n", cmdstr)
		fmt.Println("#####--------#####")
		fmt.Println("*) To execute it enter: y")
		fmt.Println("*) To copy to clipboard and exit to terminal enter: c")
		fmt.Println("*) To copy to clipboard and edit and run with goterm enter: g")
		fmt.Println("*) To send a new request with a new prompt enter: r")
		fmt.Println("*) To exit enter: q")
		fmt.Print("-> ")
		input, err := readInput(reader)
		if err != nil {
			log.Printf("\"error: %v \n", err)
		}
		switch strings.ToLower(input) {
		case "y":
			return executeCmd(cmdstr)
		case "c":
			err := clipboard.WriteAll(cmdstr)
			if err != nil {
				return "", fmt.Errorf("failed to copy to the clipboard: %w ", err)
			}
			fmt.Print("Command copied to clipboard. Exiting.")
			return "", nil
		case "g":
			err := clipboard.WriteAll(cmdstr)
			if err != nil {
				return "", fmt.Errorf("failed to copy to the clipboard: %w ", err)
			}
			fmt.Print("Command copied to clipboard. You can paste it into the terminal:")
			input, err := readInput(reader)
			if err != nil {
				return "", err
			}
			cmdstr = input
		case "r":
		case "q":
			return "", nil
		default:
			fmt.Println("unknown command to exit press 'q'.")

		}
	}
}

func handleAPIError(err error, reader *bufio.Reader) (retry bool, newAPIKey string) {
	fmt.Println("---------")
	log.Printf("\"failed connecting to the endpoint: --")
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "invalid_api_key":
			log.Printf("API error: %v\n", apiErr)
			newKey, keyErr := handleApiKey("", reader)
			if keyErr != nil {
				log.Printf("failed obtaining a new key %v\n", keyErr)
				return false, ""
			}
			return true, newKey
		default:
			log.Printf("API error: %v\n", apiErr)
			return false, ""
		}
	} else {
		log.Printf("Non-API error: %v\n", err)
		fmt.Println("---------")
		return false, ""
	}

}

func readInput(reader *bufio.Reader) (string, error) {

	if input, err := reader.ReadString('\n'); err == nil {
		return strings.TrimSpace(input), nil
	} else {
		return "", &InputReadError{err}
	}
}

func executeCmd(cmdstr string) (string, error) {
	cmd := exec.Command("sh", "-c", cmdstr)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed reading response body: %w ", err)
	}
	return out.String(), nil
}
