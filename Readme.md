# aiterm

![MIT License](https://img.shields.io/badge/license-MIT-green.svg)

## Introduction

`aiterm` is a command-line application that translate natural 
language to Unix commands, leveraging AI APIs. Designed for developers, system administrators, 
and anyone accustomed to the terminal but seeking a more intuitive way 
to interact with their systems, with `aiterm` instead of googling the commands you can have them 
readily in your terminal. Developed in Go, `aiterm` is currently 
in beta, offering a glimpse into a future where commands are more accessible 
and user-friendly.

## Features

- **Natural Language Processing**: Translate natural language commands into Unix commands using advanced AI APIs.
- **Flexible Configuration**: Use an environment variable or a flag to set the OpenAI API key.
- **Interactive Options**: After translating a command, choose to run it directly, copy it to the clipboard, or edit it further for customization.
- **Contextual Awareness**: (Upcoming) Send subsequent requests without context or with the current context for refined command suggestions.

## Installation

Since `aiterm` is in beta, it can be installed by cloning the repository and building the project with Go. Here are the steps:

```bash
git clone https://github.com/yourgithubusername/aiterm.git
cd aiterm
go build -o aiterm
```
## Configuration

To use `aiterm`, you need to set the OpenAI API key. This can be done in two ways:

1. Set an environment variable `OPENAI_KEY` with your OpenAI API key.
2. Use the `-key` flag when running `aiterm` to provide the API key.

If the API key is not set, `aiterm` will prompt you to enter it manually or offer to set it up for future use.

## Usage

To start `aiterm`, simply run the built executable. Here's a basic example:

```bash
./aiterm "find all the files that has the word foo in them in the previous directory"
```
<p align="center">
  <img alig src="https://github.com/Thakay/aiterm/blob/main/usage.gif" />
</p>
## Contributing

Contributions are welcome! If you're interested in improving `aiterm`, please follow these steps:

1. Fork the repository.
2. Create a new branch for your feature (`git checkout -b feature-branch`).
3. Make your changes.
4. Commit your changes (`git commit -am 'Add some feature'`).
5. Push to the branch (`git push origin feature-branch`).
6. Submit a Pull Request.


## License

`aiterm` is licensed under the MIT License. This permits personal and commercial use, modification, distribution, and private use of the software under the condition that the license and copyright notice are included in all copies or substantial portions of the software. For the full license text, see the LICENSE file in the project root.

## Contact Information

For questions, support, or contributions, please contact [My email](mailto:your.email@example.com).

