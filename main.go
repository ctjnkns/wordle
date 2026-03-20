package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	gwordleWordsURL  = "https://gist.githubusercontent.com/cfreshman/8b92bc418b43096094cf5d1b0eea8f84/raw/2519c8c22e3274b7a665fe11ab233a96416defc2/nyt-wordle-allowed-guesses-2026-03-06.txt"
	gwordleWordsFile = "gwordle.txt"
)

const (
	reset  = "\033[0m"
	green  = "\033[32m"
	yellow = "\033[33m"
	grey   = "\033[90m"
	red    = "\033[31m"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug level logging")
	nocache := flag.Bool("no-cache", false, "set no-cache to re-fetch the world word list from github")
	setWord := flag.String("set-word", "", "use to set a specific word as the answer for debugging")

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintln(out, "Play gwordle in the terminal.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Options:")
		flag.PrintDefaults()
		fmt.Fprintln(out)
	}

	flag.Parse()

	if *nocache {
		err := downloadWords(debug)
		if err != nil {
			log.Fatal(err)
		}
	}

	words, err := loadWords(debug)
	if err != nil {
		log.Fatal(err)
	}

	wordsMap := buildWordsMap(debug, words)

	selectedWord, err := selectWord(debug, setWord, words, wordsMap)
	if err != nil {
		log.Fatal(err)
	}

	err = playGame(debug, selectedWord, wordsMap)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nThe correct word was: %s\n", selectedWord)
}

func playGame(debug *bool, selectedWord string, wordsMap map[string]bool) error {
	board := make([]string, 0, 6)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if len(board) == 6 {
			fmt.Printf("Ran out of guesses\n")
			break
		}
		fmt.Fprintf(os.Stdout, "\nEnter a 5-letter word: ")

		if !scanner.Scan() {
			break
		}

		guess := strings.TrimSpace(scanner.Text())
		upperGuess := strings.ToUpper(guess)
		if err := validateGuess(upperGuess, wordsMap); err != nil {
			fmt.Println(err.Error())

			continue
		}

		statuses, upperGuessRune, won := gradeGuess(upperGuess, selectedWord)
		encodedGuess := encodeGuess(upperGuessRune, statuses)
		board = append(board, encodedGuess)

		renderBoard(board)
		if won {
			fmt.Println("YOU WON!!!!!")
			break
		}
	}

	if *debug {
		fmt.Printf("correct word was: %s", selectedWord)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func downloadWords(debug *bool) error {
	resp, err := http.Get(gwordleWordsURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to download words: %d", resp.StatusCode)
	}

	file, err := os.Create(gwordleWordsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	if *debug {
		log.Println("Successfully downloaded words from github")
	}

	return nil
}

func loadWords(debug *bool) ([]string, error) {
	file, err := os.Open(gwordleWordsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open gwordle words file, run with --no-cache to re-download: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read gwordle file: %w", err)
	}

	words := strings.Split(strings.TrimSpace(string(data)), "\n")

	if *debug {
		log.Println("Successfully loaded words")
	}

	return words, nil
}

func gradeGuess(upperGuess string, selectedWord string) ([]string, []rune, bool) {
	gr := []rune(upperGuess)
	sr := []rune(selectedWord)
	numRight := 0
	statuses := make([]string, len(gr))
	unMatched := map[rune]int{}
	for i := range gr {
		if gr[i] == sr[i] {
			statuses[i] = green
			numRight++
		} else {
			unMatched[sr[i]]++
		}
	}

	for i := range gr {
		if statuses[i] != green && unMatched[gr[i]] > 0 {
			unMatched[gr[i]]--
			statuses[i] = yellow
		} else if statuses[i] != green {
			statuses[i] = grey
		}
	}

	return statuses, gr, numRight == 5
}

func validateGuess(upperGuess string, wordsMap map[string]bool) error {
	gr := []rune(upperGuess)
	if len(gr) != 5 {
		return fmt.Errorf("Enter a 5 letter word (you entered %d)", len(gr))
	}
	if !wordsMap[upperGuess] {
		return fmt.Errorf("Enter a valid word (%s is not in the dict)", upperGuess)
	}

	return nil
}

func buildWordsMap(debug *bool, words []string) map[string]bool {
	wordMap := make(map[string]bool, len(words))
	for _, word := range words {
		upperWord := strings.ToUpper(word)
		wordMap[upperWord] = true
	}

	if *debug {
		log.Println("Created word map")
	}

	return wordMap
}

func selectWord(debug *bool, setWord *string, words []string, wordsMap map[string]bool) (string, error) {
	if len(words) == 0 {
		return "", errors.New("words list cannot be empty")
	}
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(words))
	selectedWord := words[index]
	if *setWord != "" {
		selectedWord = strings.ToUpper(*setWord)
		if !wordsMap[selectedWord] {
			return "", fmt.Errorf("invalid set word, not in dict: %s", selectedWord)
		}
	}

	upperSelected := strings.ToUpper(selectedWord)
	if *debug {
		log.Printf("Selected word: %s", upperSelected)
	}

	return upperSelected, nil
}

func encodeGuess(upperGuessRune []rune, statuses []string) string {
	encodedGuess := ""
	for i := range upperGuessRune {
		encodedGuess += statuses[i] + string(upperGuessRune[i]) + reset
	}
	return encodedGuess
}

func renderBoard(board []string) {
	for _, guess := range board {
		fmt.Println(guess)
	}
}
