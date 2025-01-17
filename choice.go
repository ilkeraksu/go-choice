package gochoice

import (
	"errors"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var (
	// ErrNoChoiceSelected is the error returned when no choices have been selected.
	// This can happen when the user quits the application by terminating the process (e.g. CTRL+C)
	// or by exiting the application through the ESC key or the left arrow key.
	ErrNoChoiceSelected = errors.New("no choice selected")

	// ErrNoChoice is the error returned when there are no choices to pick from
	ErrNoChoice = errors.New("no choices to choose from")

	defaultConfig = Config{
		TextColor:         White.toTcellColor(),
		BackgroundColor:   Black.toTcellColor(),
		SelectedTextColor: White.toTcellColor(),
		SelectedTextBold:  false,
	}
)

// Pick prompts the user to choose an option from a list of choices
func Pick(question string, choicesToPickFrom []string, options ...Option) (string, int, error) {
	config := defaultConfig
	for _, option := range options {
		option(&config)
	}
	screen, err := createScreen()
	if err != nil {
		return "", 0, err
	}
	defer screen.Fini()
	screen.SetStyle(tcell.StyleDefault.Background(config.BackgroundColor))
	return pick(question, choicesToPickFrom, screen, &config)
}

func pick(question string, choicesToPickFrom []string, screen tcell.Screen, config *Config) (string, int, error) {
	if len(choicesToPickFrom) == 0 {
		return "", 0, ErrNoChoice
	}
	var choices []*Choice
	for i, choice := range choicesToPickFrom {
		choices = append(choices, &Choice{Id: i, Value: choice, Selected: i == 0})
	}
	quit := make(chan struct{})
	selectedChoice := choices[0]
	var searchQuery string
	go func() {
		for {
			render(screen, question, choices, config, selectedChoice, searchQuery)
			ev := screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyUp:
					selectedChoice = moveUp(choices, 1)
				case tcell.KeyDown:
					selectedChoice = moveDown(choices, 1)
				case tcell.KeyHome:
					selectedChoice = moveUp(choices, len(choices))
				case tcell.KeyEnd:
					selectedChoice = moveDown(choices, len(choices))
				case tcell.KeyPgUp:
					selectedChoice = moveUp(choices, computePageSize(screen, question))
				case tcell.KeyPgDn:
					selectedChoice = moveDown(choices, computePageSize(screen, question))
				case tcell.KeyBackspace, tcell.KeyBackspace2:
					if len(searchQuery) > 0 {
						searchQuery = searchQuery[:len(searchQuery)-1]
						render(screen, question, choices, config, selectedChoice, searchQuery)
						selectedChoice = moveUp(choices, len(choices))
					}
				case tcell.KeyEnter, tcell.KeyRight:
					// The current selected choice is already set, so we just quit
					close(quit)
					return
				case tcell.KeyEscape, tcell.KeyCtrlC, tcell.KeyLeft:
					// No choices were selected, so we'll set selectedChoice to nil and quit
					selectedChoice = nil
					close(quit)
					return
				case tcell.KeyRune:
					searchQuery += string(ev.Rune())
					render(screen, question, choices, config, selectedChoice, searchQuery)
					selectedChoice = moveUp(choices, len(choices))
				}
			case *tcell.EventResize:
				screen.Sync()
			}
		}
	}()
	<-quit
	if selectedChoice == nil {
		return "", 0, ErrNoChoiceSelected
	}
	return selectedChoice.Value, selectedChoice.Id, nil
}

func computePageSize(screen tcell.Screen, question string) int {
	_, height := screen.Size()
	questionLines := len(strings.Split(question, "\n"))
	if height > questionLines {
		height -= questionLines + 1
	}
	return height
}

func move(choices []*Choice, increment int) *Choice {
	var choicesNotHidden []*Choice
	selectedChoiceExists := false
	for _, choice := range choices {
		if !choice.hidden {
			choicesNotHidden = append(choicesNotHidden, choice)
			if choice.Selected {
				selectedChoiceExists = true
			}
		} else {
			// If we have a hidden choice selected, we need to find the closest one
			if choice.Selected {
				choice.Selected = false
			}
		}
	}
	if len(choicesNotHidden) == 0 {
		return nil
	}
	if !selectedChoiceExists {
		choicesNotHidden[0].Selected = true
		return choicesNotHidden[0]
	}
	for i, choice := range choicesNotHidden {
		if choice.Selected {
			if i+increment < len(choicesNotHidden) && i+increment > 0 { // Between 0 and last choice
				choicesNotHidden[i].Selected = false
				choicesNotHidden[i+increment].Selected = true
				return choicesNotHidden[i+increment]
			} else if i+increment >= len(choicesNotHidden) { // Higher than last choice
				choicesNotHidden[i].Selected = false
				choicesNotHidden[len(choicesNotHidden)-1].Selected = true
				return choicesNotHidden[len(choicesNotHidden)-1]
			} else if i+increment <= 0 { // Lower than 0
				choicesNotHidden[i].Selected = false
				choicesNotHidden[0].Selected = true
				return choicesNotHidden[0]
			}
			// Choice didn't change, return it
			return choice
		}
	}
	panic("Something went wrong")
}

func moveUp(choices []*Choice, step int) *Choice {
	return move(choices, -step)
}

func moveDown(choices []*Choice, step int) *Choice {
	return move(choices, step)
}
