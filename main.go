package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dev-sandip/pokedex/internal/pokecache"
)

type config struct {
	commands             map[string]cliCommand
	nextLocationsURL     *string
	previousLocationsURL *string
	cache                pokecache.Cache
	pokedex              map[string]Pokemon
}

type cliCommand struct {
	name        string
	description string
	callback    func(*config, []string) error
}

type Pokemon struct {
	Name           string `json:"name"`
	BaseExperience int    `json:"base_experience"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
}

type pokemonResponse = Pokemon

type locationAreasResponse struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
}

type locationAreaDetailResponse struct {
	Name              string `json:"name"`
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"pokemon_encounters"`
}

const locationAreasURL = "https://pokeapi.co/api/v2/location-area?limit=20"

func commandExit(cfg *config, args []string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *config, args []string) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println()

	for _, cmd := range cfg.commands {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}

	return nil
}

func commandMap(cfg *config, args []string) error {
	url := locationAreasURL
	if cfg.nextLocationsURL != nil {
		url = *cfg.nextLocationsURL
	}

	data, err := getLocationAreas(url, &cfg.cache)
	if err != nil {
		return err
	}

	cfg.nextLocationsURL = data.Next
	cfg.previousLocationsURL = data.Previous

	for _, area := range data.Results {
		fmt.Println(area.Name)
	}

	return nil
}

func commandMapBack(cfg *config, args []string) error {
	if cfg.previousLocationsURL == nil {
		fmt.Println("you're on the first page")
		return nil
	}

	data, err := getLocationAreas(*cfg.previousLocationsURL, &cfg.cache)
	if err != nil {
		return err
	}

	cfg.nextLocationsURL = data.Next
	cfg.previousLocationsURL = data.Previous

	for _, area := range data.Results {
		fmt.Println(area.Name)
	}

	return nil
}

func commandExplore(cfg *config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: explore <area_name>")
	}

	areaName := args[0]
	fmt.Printf("Exploring %s...\n", areaName)

	area, err := getLocationArea(areaName, &cfg.cache)
	if err != nil {
		return err
	}

	fmt.Println("Found Pokemon:")
	for _, encounter := range area.PokemonEncounters {
		fmt.Printf(" - %s\n", encounter.Pokemon.Name)
	}

	return nil
}

func commandCatch(cfg *config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: catch <pokemon_name>")
	}

	name := args[0]
	fmt.Printf("Throwing a Pokeball at %s...\n", name)

	pokemon, err := getPokemon(name, &cfg.cache)
	if err != nil {
		return err
	}

	// Higher base experience means a lower catch chance.
	// This formula keeps the chance reasonable for early-game Pokemon.
	catchChance := 1.0 / (1.0 + float64(pokemon.BaseExperience)/100.0)

	if rand.Float64() < catchChance {
		fmt.Printf("%s was caught!\n", pokemon.Name)
		cfg.pokedex[pokemon.Name] = pokemon
	} else {
		fmt.Printf("%s escaped!\n", pokemon.Name)
	}

	return nil
}

func startRepl(cfg *config) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Pokedex > ")

		if !scanner.Scan() {
			return
		}

		words := cleanInput(scanner.Text())
		if len(words) == 0 {
			continue
		}

		commandName := words[0]
		cmd, ok := cfg.commands[commandName]
		if !ok {
			fmt.Println("Unknown command")
			continue
		}

		if err := cmd.callback(cfg, words[1:]); err != nil {
			fmt.Println("Error:", err)
		}
	}
}

func main() {
	cfg := &config{
		commands: make(map[string]cliCommand),
		cache:    pokecache.NewCache(5 * time.Second),
		pokedex:  make(map[string]Pokemon),
	}

	cfg.commands["exit"] = cliCommand{
		name:        "exit",
		description: "Exit the Pokedex",
		callback:    commandExit,
	}

	cfg.commands["help"] = cliCommand{
		name:        "help",
		description: "Displays a help message",
		callback:    commandHelp,
	}

	cfg.commands["map"] = cliCommand{
		name:        "map",
		description: "Display the next 20 location areas",
		callback:    commandMap,
	}

	cfg.commands["mapb"] = cliCommand{
		name:        "mapb",
		description: "Display the previous 20 location areas",
		callback:    commandMapBack,
	}

	cfg.commands["explore"] = cliCommand{
		name:        "explore",
		description: "Explore a location area and find its Pokemon",
		callback:    commandExplore,
	}

	cfg.commands["catch"] = cliCommand{
		name:        "catch",
		description: "Catch a Pokemon and add it to your Pokedex",
		callback:    commandCatch,
	}

	startRepl(cfg)
}

func getLocationAreas(
	url string,
	cache *pokecache.Cache,
) (locationAreasResponse, error) {
	var data locationAreasResponse

	if cachedData, ok := cache.Get(url); ok {
		if err := json.Unmarshal(cachedData, &data); err != nil {
			return data, fmt.Errorf("decoding cached location areas: %w", err)
		}
		return data, nil
	}

	res, err := http.Get(url)
	if err != nil {
		return data, fmt.Errorf("requesting location areas: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return data, fmt.Errorf("PokeAPI returned status %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return data, fmt.Errorf("reading location areas response: %w", err)
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return data, fmt.Errorf("decoding location areas: %w", err)
	}

	cache.Add(url, body)
	return data, nil
}

func getLocationArea(
	areaName string,
	cache *pokecache.Cache,
) (locationAreaDetailResponse, error) {
	var data locationAreaDetailResponse

	url := "https://pokeapi.co/api/v2/location-area/" + areaName + "/"

	if cachedData, ok := cache.Get(url); ok {
		if err := json.Unmarshal(cachedData, &data); err != nil {
			return data, fmt.Errorf("decoding cached location area: %w", err)
		}
		return data, nil
	}

	res, err := http.Get(url)
	if err != nil {
		return data, fmt.Errorf("requesting location area: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return data, fmt.Errorf("PokeAPI returned status %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return data, fmt.Errorf("reading location area response: %w", err)
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return data, fmt.Errorf("decoding location area: %w", err)
	}

	cache.Add(url, body)
	return data, nil
}

func getPokemon(
	name string,
	cache *pokecache.Cache,
) (Pokemon, error) {
	var pokemon Pokemon

	url := "https://pokeapi.co/api/v2/pokemon/" + name + "/"

	if cachedData, ok := cache.Get(url); ok {
		if err := json.Unmarshal(cachedData, &pokemon); err != nil {
			return pokemon, fmt.Errorf("decoding cached Pokemon: %w", err)
		}
		return pokemon, nil
	}

	res, err := http.Get(url)
	if err != nil {
		return pokemon, fmt.Errorf("requesting Pokemon: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return pokemon, fmt.Errorf("PokeAPI returned status %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return pokemon, fmt.Errorf("reading Pokemon response: %w", err)
	}

	if err := json.Unmarshal(body, &pokemon); err != nil {
		return pokemon, fmt.Errorf("decoding Pokemon: %w", err)
	}

	cache.Add(url, body)
	return pokemon, nil
}

func cleanInput(text string) []string {
	return strings.Fields(strings.ToLower(text))
}
