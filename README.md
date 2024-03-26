# Text-to-Speech Starter (Go)

This example app demonstrates how to use the Deepgram Text-to-Speech API with Go.

## What is Deepgram?

[Deepgram](https://deepgram.com/) is a voice AI company providing speech-to-text and language understanding capabilities to make data readable and actionable by human or machines.

## Sign-up to Deepgram

Before you start, it's essential to generate a Deepgram API key to use in this project. [Sign-up now for Deepgram and create an API key](https://console.deepgram.com/signup?jump=keys).

## Quickstart

### Manual

Follow these steps to get started with this starter application.

#### Clone the repository

Go to GitHub and [clone the repository](https://github.com/deepgram-devs/text-to-speech-starter-go).

#### Install dependencies

Install the project dependencies.

```bash
go get
```

#### Select branch

The `main` branch demonstrates a basic implementation: text is sent to the API and an audio file response with synthesized text-to-speech is returned.

Checkout the other branches to see added functionality:

- [output streaming](https://github.com/deepgram-starters/text-to-speech-starter-go/tree/output-streaming): Demonstrates how to take advantage of Deepgram's output streaming feature. This example streams the audio response to the client as it is being generated. Read more at [Deepgram's output streaming guide].(https://developers.deepgram.com/docs/streaming-the-audio-output)

```bash
git checkout output-streaming
```

#### Set your Deepgram API key

If using bash, this can be done in your `~/.bash_profile` like so:

```bash
export DEEPGRAM_API_KEY="YOUR_DEEPGRAM_API_KEY"
```

or this could also be done by a simple export before executing your Go application:

```bash
DEEPGRAM_API_KEY="YOUR_DEEPGRAM_API_KEY" go run main.go
```

#### Run the application

The `dev` script will run a web and API server concurrently. Once running, you can [access the application in your browser](http://localhost:3000/).

```bash
go run .
```

## Issue Reporting

If you have found a bug or if you have a feature request, please report them at this repository issues section. Please do not report security vulnerabilities on the public GitHub issue tracker. The [Security Policy](./SECURITY.md) details the procedure for contacting Deepgram.

## Getting Help

We love to hear from you so if you have questions, comments or find a bug in the project, let us know! You can either:

- [Open an issue in this repository](https://github.com/deepgram-starters/live-node-starter/issues/new)
- [Join the Deepgram Github Discussions Community](https://github.com/orgs/deepgram/discussions)
- [Join the Deepgram Discord Community](https://discord.gg/xWRaCDBtW4)

## Author

[Deepgram](https://deepgram.com)

## License

This project is licensed under the MIT license. See the [LICENSE](./LICENSE) file for more info.
