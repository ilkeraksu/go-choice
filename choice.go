package gochoice

import (
	"errors"
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	"strings"
)

var defaultConfig = Config{
	TextColor:         White.toTcellColor(),
	BackgroundColor:   Black.toTcellColor(),
	SelectedTextColor: White.toTcellColor(),
	SelectedTextBold:  false,
}

func Pick(question string, choicesToPickFrom []string, options ...Option) (string, error) {
	config := defaultConfig
	for _, option := range options {
		option(&config)
	}
	screen, err := createScreen()
	if err != nil {
		return "", err
	}
	defer screen.Fini()
	screen.SetStyle(tcell.StyleDefault.Background(config.BackgroundColor))
	screen.Show()
	return pick(question, choicesToPickFrom, screen, &config)
}

func pick(question string, choicesToPickFrom []string, screen tcell.Screen, config *Config) (string, error) {
	if len(choicesToPickFrom) == 0 {
		return "", errors.New("no choices to choose from")
	}
	var choices []*Choice
	for i, choice := range choicesToPickFrom {
		choices = append(choices, &Choice{Id: i, Value: choice, Selected: i == 0})
	}

	quit := make(chan struct{})
	var selectedChoice = choices[0]

	go func() {
		for {
			go render(screen, question, choices, config, selectedChoice)
			ev := screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyUp:
					selectedChoice = moveUp(choices)
				case tcell.KeyDown:
					selectedChoice = moveDown(choices)
				case tcell.KeyEnter, tcell.KeyRight:
					// The current selected choice is already set, so we just quit
					close(quit)
					return
				case tcell.KeyEscape:
					// No choices were selected, so we'll set selectedChoice to nil and quit
					selectedChoice = nil
					close(quit)
					return
				case tcell.KeyRune:
					switch ev.Rune() {
					case 'k', 'w': // Up
						selectedChoice = moveUp(choices)
					case 'j', 's': // Down
						selectedChoice = moveDown(choices)
					case ' ', 'l', 'd': // Select
						// The current selected choice is already set, so we just quit
						close(quit)
						return
					case 'q': // Quit
						// No choices were selected, so we'll set selectedChoice to nil and quit
						selectedChoice = nil
						close(quit)
						return
					}
				}
			case *tcell.EventResize:
				screen.Sync()
			}
		}
	}()

	<-quit

	if selectedChoice == nil {
		return "", errors.New("aborted")
	}
	return selectedChoice.Value, nil
}

func createScreen() (tcell.Screen, error) {
	tcell.SetEncodingFallback(tcell.EncodingFallbackASCII)
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create new screen: %v", err)
	}
	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize screen: %v", err)
	}
	return screen, nil
}

func move(choices []*Choice, increment int) *Choice {
	for i, choice := range choices {
		if choice.Selected {
			if i+increment < len(choices) && i+increment >= 0 {
				choices[i].Selected = false
				choices[i+increment].Selected = true
				return choices[i+increment]
			}
			// Choice didn't change, return it
			return choice
		}
	}
	panic("Something went wrong")
}

func moveUp(choices []*Choice) *Choice {
	return move(choices, -1)
}

func moveDown(choices []*Choice) *Choice {
	return move(choices, 1)
}

func render(screen tcell.Screen, question string, options []*Choice, config *Config, selectedChoice *Choice) {
	screen.Clear()
	_, maximumThatCanBeDisplayed := screen.Size()
	lineNumber := 0
	questionLines := strings.Split(question, "\n")
	for _, line := range questionLines {
		printText(screen, 1, lineNumber, line, config.TextColor, config.BackgroundColor, config.SelectedTextBold)
		lineNumber += 1
	}
	min := selectedChoice.Id + len(questionLines)
	max := maximumThatCanBeDisplayed
	if selectedChoice.Id > max {
		min += 1
		max += 1
	}
	for _, option := range options {
		if option.Id <= (min+1)-maximumThatCanBeDisplayed && !(option.Id > (min+1)-maximumThatCanBeDisplayed) {
			continue
		}
		if option.Selected {
			printText(screen, 1, lineNumber, fmt.Sprintf("> %s", option.Value), config.SelectedTextColor, config.BackgroundColor, config.SelectedTextBold)
		} else {
			printText(screen, 3, lineNumber, option.Value, config.TextColor, config.BackgroundColor, config.SelectedTextBold)
		}
		lineNumber += 1
	}
	screen.Sync()
}

func printText(screen tcell.Screen, x, y int, text string, fg, bg tcell.Color, bold bool) {
	for _, character := range text {
		screen.SetCell(x, y, tcell.StyleDefault.Background(bg).Foreground(fg), character)
		x += runewidth.RuneWidth(character)
	}
}
