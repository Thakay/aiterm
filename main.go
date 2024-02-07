package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

const (
	varKeyName = "OPENAI_KEY"
)

func main() {

	apiKey := os.Getenv(varKeyName)
	flagKey := flag.String("key", apiKey, "API key from OpenAI")
	flagURL := flag.String("url", defaultEndPoint, "custom API endpoints to interact with")

	flag.Parse()
	apiKey = *flagKey
	apiURL := *flagURL

	//userRequest := "a command to find all the occurrences of the word foo in all the files of this directory" //"list all the files and folders of parent directory with extra details" //"copy the currernt directory to a new folder in desktop named copied"
	positionalArgs := flag.Args()
	if len(positionalArgs) == 0 {
		fmt.Println("No prompt was provided.")
		return
	}
	
	userRequest := positionalArgs[0]

	openAIClient := NewOpenAIProvider(apiKey, &OpenAIOptions{
		ProviderOptions: &ProviderOptions{
			URL:    apiURL,
			APIKey: apiKey,
		},
		model:            "gpt-3.5-turbo",
		temperature:      1.0,
		maxTokens:        256,
		topP:             1.0,
		frequencyPenalty: 0.0,
		presencePenalty:  0.0,
		withContext:      true,
	})

	app := NewApp(openAIClient, userRequest)

	err := app.Run()
	if err != nil {
		log.Fatalf("failed running the app: %v", err)
	}

}
