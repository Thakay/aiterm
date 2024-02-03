package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	varName := "OPENAI_KE"
	apiKey := os.Getenv(varName)
	apiKey, err := handleApiKey(apiKey)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(apiKey)

	cmd, err := api(apiKey, "https://api.openai.com/v1/chat/completions", "copy the currernt directory to a new folder in desktop named copied")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cmd)

}
func handleApiKey(apiKey string) (string, error) {
	if apiKey == "" {
		fmt.Println("Please set the Environment Variable named OPENAI_KEY with you secret API key manually and retry. or enter it for temporary use.")

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter your OpenAI API key or empty to exit: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("could not read the input value for API key:%w", err)
		}
		apiKey = strings.TrimSpace(input)
		if apiKey == "" {
			fmt.Println("No API Key was provided... Please make sure to set the Environment variable. Exiting... ")
			return "", nil
		}
	}
	return apiKey, nil
}
