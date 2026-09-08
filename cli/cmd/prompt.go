package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

type prompter struct {
	r *bufio.Reader
}

func newPrompter(r *bufio.Reader) *prompter {
	return &prompter{r: r}
}

func (p *prompter) line() string {
	text, _ := p.r.ReadString('\n')
	return strings.TrimSpace(text)
}

// ask prints a question and returns the trimmed answer, or def if the
// answer is empty.
func (p *prompter) ask(question, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", question, def)
	} else {
		fmt.Printf("%s: ", question)
	}
	ans := p.line()
	if ans == "" {
		return def
	}
	return ans
}

// askRequired keeps asking until a non-empty answer is given.
func (p *prompter) askRequired(question string) string {
	for {
		ans := p.ask(question, "")
		if ans != "" {
			return ans
		}
		fmt.Println("This field is required.")
	}
}

func (p *prompter) askYesNo(question string, defYes bool) bool {
	suffix := "y/N"
	if defYes {
		suffix = "Y/n"
	}
	fmt.Printf("%s [%s]: ", question, suffix)
	ans := strings.ToLower(p.line())
	if ans == "" {
		return defYes
	}
	return ans == "y" || ans == "yes"
}

func (p *prompter) askInt(question string, def int) int {
	for {
		ans := p.ask(question, strconv.Itoa(def))
		n, err := strconv.Atoi(ans)
		if err == nil {
			return n
		}
		fmt.Println("Please enter a whole number.")
	}
}

func (p *prompter) askFloat(question string, def float64) float64 {
	for {
		ans := p.ask(question, strconv.FormatFloat(def, 'g', -1, 64))
		n, err := strconv.ParseFloat(ans, 64)
		if err == nil {
			return n
		}
		fmt.Println("Please enter a number.")
	}
}
