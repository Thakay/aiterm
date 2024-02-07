package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/atotto/clipboard"
	"log"
	"os"
	"os/exec"
	"strings"
)

type App struct {
	Client      APIProvider
	reader      *bufio.Reader
	userRequest string
	fetchCfg    FetchConfig
}

func NewApp(provider APIProvider, userReq string) *App {
	return &App{
		Client:      provider,
		reader:      bufio.NewReader(os.Stdin),
		userRequest: userReq,
		fetchCfg:    provider.newFetchConfig(true),
	}
}

func (a *App) Run() error {

	if ok, err := a.HandleEmptyAPIKey(); err != nil {
		return err

	} else if !ok {
		log.Println("No API Key was provided... Please make sure to set the Environment variable. Exiting... ")
		return nil
	}
	for {

		cmdstr, err := a.Client.fetch(a.userRequest, a.fetchCfg)

		if err != nil {
			err = a.HandleAPIError(err)
			log.Fatalf("failed criticial API calling: %v \n Exiting...\n", err)
		}
		//cmdstr := "ls -l asd"
		if ok, validCmdstr := a.ValidateCmd(cmdstr); !ok {
			fmt.Println("Please retry with a different prompt.")
			input, err := a.readInput()
			if err != nil {
				log.Printf("\"error: %v \n", err)
			}
			a.userRequest = input
			continue
		} else {
			if end, err := a.HandleCmd(validCmdstr); err != nil {
				log.Println("failed handling the command: ", err)
				return err
			} else if end {
				return nil
			}
		}
	}
}

func (a *App) HandleCmd(cmd string) (bool, error) {
	for {
		fmt.Print("\n\n\n")
		fmt.Printf("Here is the command --> %s <-- \n", cmd)
		fmt.Println("#####--------#####")
		fmt.Println("*) To execute it enter: y")
		fmt.Println("*) To copy to clipboard and exit to terminal enter: c")
		fmt.Println("*) To copy to clipboard and edit and run with aiterm enter: g")
		fmt.Println("*) To send a new request with context enter: r")
		fmt.Println("*) To send a new request without context enter: w")
		fmt.Println("*) To exit enter: q")
		fmt.Print("-> ")
		input, err := a.readInput()
		if err != nil {
			log.Printf("\"error: %v \n", err)
			return true, err
		}
		switch strings.ToLower(input) {
		case "y":
			if res, err := a.executeCmd(cmd); err != nil {
				a.ShowResult(res)
				return false, err
			} else {
				fmt.Println(res)
				return true, nil
			}

		case "c":
			err := clipboard.WriteAll(cmd)
			if err != nil {
				return true, fmt.Errorf("failed to copy to the clipboard: %w ", err)
			}
			fmt.Print("Command copied to clipboard. Exiting.")
			return true, nil
		case "g":
			err := clipboard.WriteAll(cmd)
			if err != nil {
				return true, fmt.Errorf("failed to copy to the clipboard: %w ", err)
			}
			fmt.Print("Command copied to clipboard. You can paste it into the terminal:")
			input, err := a.readInput()
			if err != nil {
				return true, err
			}
			cmd = input
		case "r":
			fmt.Print("(+c)Enter the new prompt:")
			input, err := a.readInput()
			if err != nil {
				return true, err
			}
			a.userRequest = input
			a.fetchCfg = a.Client.newFetchConfig(true)
			return false, nil
		case "w":
			fmt.Print("(-c)Enter the new prompt:")
			input, err := a.readInput()
			if err != nil {
				return true, err
			}
			a.userRequest = input
			a.fetchCfg = a.Client.newFetchConfig(false)
			return false, nil
		case "q":
			return true, nil
		default:
			fmt.Println("unknown command to exit press 'q'.")

		}
	}
}
func (a *App) ShowResult(res string) {

	fmt.Printf("\n Your command has been executed and this is the output: \n")
	fmt.Println(res)
}
func (a *App) HandleEmptyAPIKey() (bool, error) {
	if ok := a.Client.hasAPIKey(); !ok {

		fmt.Println("Please set the Environment Variable named OPENAI_KEY with you secret API key manually and retry. or enter it for temporary use.")
		fmt.Print("Enter your OpenAI API key or empty to exit: ")
		input, err := a.readInput()
		if err != nil {
			log.Printf("\"error: %v \n", err)
			return true, err
		} else if input != "" {
			a.Client.setAPIKey(input)
			return true, nil
		}
		return false, nil
	}
	return true, nil

}

func (a *App) HandleInvalidAPIKey() {
	fmt.Printf("\n\n\n!!!!!!!!!!!!")
	fmt.Println("Please set the correct API Key and retry.")
}

func (a *App) ValidateCmd(cmd string) (bool, string) {
	cmd = strings.TrimSpace(cmd)
	if strings.ToLower(cmd) == "not a command" || strings.ToLower(cmd) == "not a command." {
		return false, ""
	}
	return true, cmd
}

func (a *App) HandleAPIError(err error) error {
	fmt.Println("---------")
	log.Printf("\"failed connecting to the endpoint: --")
	err = a.Client.handleAPIError(err)
	var apiErr *APIKeyError
	if errors.As(err, &apiErr) {
		log.Printf("APIKey error: %v\n", apiErr)
	}
	log.Printf("Non-API error: %v\n", err)
	fmt.Println("---------")

	return err

}

func (a *App) readInput() (string, error) {
	if input, err := a.reader.ReadString('\n'); err == nil {
		return strings.TrimSpace(input), nil
	} else {
		return "", &InputReadError{err}
	}

}

func (a *App) executeCmd(cmd string) (string, error) {
	res := exec.Command("sh", "-c", cmd)
	var out bytes.Buffer
	res.Stdout = &out
	err := res.Run()
	if err != nil {
		return "", fmt.Errorf("failed reading response body: %w ", err)
	}
	return out.String(), nil
}
