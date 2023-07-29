package main

import (
	"errors"
	"fmt"
	"github.com/chzyer/readline"
	"github.com/rivo/uniseg"
	"golang.design/x/clipboard"
	"gptrp/internal/config"
	"gptrp/internal/gpt"
	"io"
	"log"
	"os"
	"strings"
)

var completer = readline.NewPrefixCompleter(
	readline.PcItem("new",
		readline.PcItem("dungeon"),
		readline.PcItem("room"),
	),
	readline.PcItem("say"),
	readline.PcItem("continue"),
	readline.PcItem("show",
		readline.PcItem("context"),
	),
	readline.PcItem("undo"),
	readline.PcItem("retry"),
	readline.PcItem("copy"),
	readline.PcItem("help"),
	readline.PcItem("quit"),
)

func usage(w io.Writer) {
	io.WriteString(w, "commands:\n")
	io.WriteString(w, completer.Tree("    "))
}

const (
	DoYouWantToContinue = "Do you want to continue?"
	DoYouWantToAddMore  = "Do you want to add additional room detail building?"
)

func runTalkLoop(gpt gpt.GPT, cfg *config.Config) {
	println("Alright let's start. Type 'quit' to exit or press Ctrl + C.\n")
	println("Type 'new dungeon' to enter into new dungeon room. (room where fighting happens)")
	println("Type 'new room' to enter into new room. (non-combat rooms anywhere in the world)\n")
	println("You're going to play scenario: " + gpt.GetScenario().Name)
	println(gpt.GetScenario().Description + "\n")

	var line string
	var temp *os.File
	var err error
	var ok bool
	var chatMessage string
	var chatMessages []string

	temp, err = os.CreateTemp(os.TempDir(), "gptrp-history-"+strings.ReplaceAll(gpt.GetScenario().Name, " ", "")+"-*.tmp")
	if err != nil {
		panic(err)
	}
	l, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[31m»\033[0m ",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "quit",
		HistoryFile:     temp.Name(),
	})
	if err != nil {
		panic(err)
	}
	defer func(l *readline.Instance) {
		err = l.Close()
		if err != nil {
			panic(err)
		}
	}(l)
	l.CaptureExitSignal()
	log.SetOutput(l.Stderr())

	err = clipboard.Init()
	if err != nil {
		log.Print(err)
	}

	for {
		line, err = l.Readline()
		if errors.Is(err, readline.ErrInterrupt) {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "new "):
			switch line[4:] {
			case "dungeon":
				var extraInput string
				ok, err = getYesNoInput(l, DoYouWantToAddMore)
				if err != nil {
					log.Print(err)
					continue
				}
				if ok {
					for {
						println("Enter additional room detail building:")
						line, err = l.Readline()
						if err != nil && !errors.Is(err, readline.ErrInterrupt) {
							log.Print(err)
							continue
						}
						ok, err = getYesNoInput(l, DoYouWantToContinue)
						if err != nil {
							log.Print(err)
							continue
						}
						if ok {
							extraInput = line
							break
						}
					}
				}

				gpt.NewContext(true, extraInput)
				l.Clean()
				println("Created new dungeon room.")
				continue
			case "room":
				var extraInput string
				ok, err = getYesNoInput(l, DoYouWantToAddMore)
				if err != nil {
					log.Print(err)
					continue
				}
				if ok {
					for {
						println("Enter additional room detail building:")
						line, err = l.Readline()
						if err != nil && !errors.Is(err, readline.ErrInterrupt) {
							log.Print(err)
							continue
						}
						ok, err = getYesNoInput(l, DoYouWantToContinue)
						if err != nil {
							log.Print(err)
							continue
						}
						if ok {
							extraInput = line
							break
						}
					}
				}

				gpt.NewContext(false, extraInput)
				l.Clean()
				println("Created new room.")
				continue
			default:
				println("invalid target:", line[4:])
			}
			continue
		case line == "continue":
			if !gpt.WasLastMessageFromUser() {
				println("You can't continue yet. Use command 'say' to say something.")
			}
			gpt.ChatCompletionStream(func(msg string) {
				print(msg)
				chatMessage += msg
			})
			print("\n")
			continue
		case strings.HasPrefix(line, "show "):
			switch line[5:] {
			case "context":
				println("Current context building:")
				for i, m := range gpt.GetContextMessages() {
					fmt.Printf(
						"%d)\n\trole:\t%s\n\ttext: %s\n",
						i+1,
						m.Role,
						wrapTextForCli(m.Content),
					)
				}
				continue
			default:
				println("show what?")
			}
			continue
		case line == "copy":
			if len(chatMessage) == 0 || len(chatMessages) == 0 {
				println("nothing to copy")
				continue
			}

			if len(chatMessages) == 0 {
				if cfg.Clipboard.WordWrap {
					chatMessages = messageSentences(chatMessage, cfg.Clipboard.MaxSize)
				} else {
					chatMessages = []string{chatMessage}
				}
				chatMessage = ""
			}

			var message string
			message, chatMessages = chatMessages[0], chatMessages[1:]
			clipboard.Write(clipboard.FmtText, []byte(message))

			println("copied to clipboard")
			continue
		case line == "undo":
			if gpt.WasLastMessageFromAssistant() {
				chatMessage = ""
			}
			gpt.Undo()
			continue
		case line == "help":
			usage(l.Stderr())
			continue
		case strings.HasPrefix(line, "say"):
			line = strings.TrimSpace(line[3:])
			if len(line) == 0 {
				println("say what?")
				break
			}

			ok, err = getYesNoInput(l, DoYouWantToContinue)
			if err != nil {
				log.Print(err)
				continue
			}
			if !ok {
				continue
			}

			gpt.AddMessage(line).ChatCompletionStream(func(msg string) {
				print(msg)
				chatMessage += msg
			})
			print("\n")
			continue
		case line == "redo":
			chatMessage = ""
			gpt.RedoLastMessage().ChatCompletionStream(func(msg string) {
				print(msg)
				chatMessage += msg
			})
			print("\n")
			continue
		case line == "quit":
			return
		case line == "":
		default:
			usage(l.Stderr())
		}
	}
}

func isYes(input string) bool {
	return isOneOfSpecialCommands(input, []string{"y", "yes"})
}

func getYesNoInput(l *readline.Instance, question string) (bool, error) {
	println(question + " (y/n) ")
	input, err := l.ReadlineWithDefault("n")
	if err != nil && !errors.Is(err, readline.ErrInterrupt) {
		return false, err
	}
	return isYes(input), nil
}

func isOneOfSpecialCommands(input string, commandList []string) bool {
	for _, command := range commandList {
		if strings.ToLower(input) == command {
			return true
		}
	}
	return false
}

func messageSentences(str string, maxSize int) []string {
	if maxSize < 1 {
		return []string{str}
	}

	state := -1
	var (
		c        string
		messages []string
		message  string
	)
	for len(str) > 0 {
		c, str, _, state = uniseg.StepString(str, state)
		for len(c) > maxSize {
			messages = append(messages, c[:maxSize])
			c = c[maxSize:]
		}
		if (len(message) + len(c)) > maxSize {
			messages = append(messages, message)
			message = ""
		}
		message += c
	}
	if len(message) > 0 {
		messages = append(messages, message)
	}
	return messages
}

func wrapTextForCli(str string) string {
	state := -1
	var (
		c      string
		b      int
		pb     int
		result string
		line   string
	)
	w := readline.GetScreenWidth()
	for len(str) > 0 {
		c, str, b, state = uniseg.StepString(str, state)
		if (len(line)+len(c)) > (w-w/3) && pb&uniseg.MaskLine == uniseg.LineCanBreak {
			result += line + "\n\t      "
			line = ""
		}
		line += c
		pb = b
	}
	if len(line) > 0 {
		result += line
	}
	return result
}
